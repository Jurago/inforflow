package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	SourceIP   = "170.245.127.191"
	APIPort    = ":9090"
	MaxHistory = 800
)

type FlowCategory string

const (
	CategoryCDN       FlowCategory = "cdn"
	CategoryGlobo     FlowCategory = "globo"
	CategoryNetflix   FlowCategory = "netflix"
	CategoryStreaming FlowCategory = "streaming"
	CategoryPeer      FlowCategory = "peer"
	CategorySocial    FlowCategory = "social"
	CategoryGaming    FlowCategory = "gaming"
	CategoryDNS       FlowCategory = "dns"
	CategoryCloud     FlowCategory = "cloud"
	CategoryApple     FlowCategory = "apple"
	CategoryOther     FlowCategory = "other"
)

type FlowRecord struct {
	ID          string       `json:"id"`
	Timestamp   int64        `json:"timestamp"`
	SrcIP       string       `json:"src_ip"`
	DstIP       string       `json:"dst_ip"`
	SrcPort     int          `json:"src_port"`
	DstPort     int          `json:"dst_port"`
	Protocol    string       `json:"protocol"`
	Bytes       int64        `json:"bytes"`
	Packets     int64        `json:"packets"`
	Direction   string       `json:"direction"`
	Category    FlowCategory `json:"category"`
	Destination string       `json:"destination"`
	Origin      string       `json:"origin"`
	ASN         string       `json:"asn"`
	DstASN      string       `json:"dst_asn,omitempty"`
	Country     string       `json:"country"`
	PeerASN     string       `json:"peer_asn,omitempty"`
	PeerName    string       `json:"peer_name,omitempty"`
	PeerIP      string       `json:"peer_ip,omitempty"`
	NextHop     string       `json:"next_hop,omitempty"`
	InIf        int          `json:"in_if,omitempty"`
	OutIf       int          `json:"out_if,omitempty"`
	CacheIface  string       `json:"cache_iface,omitempty"`
	IfaceRole   string       `json:"iface_role,omitempty"`
	IfaceName   string       `json:"iface_name,omitempty"`
	IPVersion   string       `json:"ip_version,omitempty"`
}

type DestOriginCard struct {
	Name       string  `json:"name"`
	IP         string  `json:"ip"`
	Bytes      int64   `json:"bytes"`
	Packets    int64   `json:"packets"`
	Percentage float64 `json:"percentage"`
	Category   string  `json:"category"`
	Icon       string  `json:"icon"`
	Mbps       float64 `json:"mbps"`
	MbpsScaled float64 `json:"mbps_scaled,omitempty"`
}

type ServiceRate struct {
	Name       string  `json:"name"`
	Bytes      int64   `json:"bytes"`
	Mbps       float64 `json:"mbps"`
	MbpsScaled float64 `json:"mbps_scaled"`
	InMbps     float64 `json:"in_mbps,omitempty"`
	OutMbps    float64 `json:"out_mbps,omitempty"`
	Category   string  `json:"category"`
}

// ASNTraffic — tráfego agregado com destino a um ASN (NetFlow DstAS).
type ASNTraffic struct {
	ASN        string  `json:"asn"`
	Name       string  `json:"name"`
	Bytes      int64   `json:"bytes"`
	Flows      int64   `json:"flows"`
	Mbps       float64 `json:"mbps"`
	MbpsScaled float64 `json:"mbps_scaled"`
	InMbps     float64 `json:"in_mbps,omitempty"`
	OutMbps    float64 `json:"out_mbps,omitempty"`
	IPv4Mbps   float64 `json:"ipv4_mbps,omitempty"`
	IPv6Mbps   float64 `json:"ipv6_mbps,omitempty"`
	Percentage float64 `json:"percentage"`
	Category   string  `json:"category"`
	Icon       string  `json:"icon"`
}

type CategoryConsumption struct {
	Category   string  `json:"category"`
	Bytes      int64   `json:"bytes"`
	Percentage float64 `json:"percentage"`
	Mbps       float64 `json:"mbps"`        // NetFlow amostrado (janela)
	MbpsScaled float64 `json:"mbps_scaled"` // estimado vs SNMP (× sampling)
	Label      string  `json:"label"`
}

type AggregatedStats struct {
	TotalBytes      int64                 `json:"total_bytes"`
	TotalPackets    int64                 `json:"total_packets"`
	TotalFlows      int64                 `json:"total_flows"`
	BytesPerSec     float64               `json:"bytes_per_sec"`
	PacketsPerSec   float64               `json:"packets_per_sec"`
	Mbps            float64               `json:"mbps"`
	ByCategory      map[string]int64      `json:"by_category"`
	ByCategoryMbps       map[string]float64    `json:"by_category_mbps"`
	ByCategoryMbpsScaled map[string]float64    `json:"by_category_mbps_scaled,omitempty"`
	ByCategoryInMbps     map[string]float64    `json:"by_category_in_mbps,omitempty"`
	ByCategoryOutMbps    map[string]float64    `json:"by_category_out_mbps,omitempty"`
	ByDestination        map[string]int64      `json:"by_destination"`
	ByOrigin             map[string]int64      `json:"by_origin"`
	TopDestinations      []DestOriginCard      `json:"top_destinations"`
	TopOrigins           []DestOriginCard      `json:"top_origins"`
	TopTalkers           []TalkerStats         `json:"top_talkers,omitempty"`
	ByIfaceRole          map[string]float64    `json:"by_iface_role_mbps,omitempty"`
	CDNBreakdown         map[string]int64      `json:"cdn_breakdown"`
	StreamingBreak       map[string]int64      `json:"streaming_breakdown"`
	StreamingRates       []ServiceRate         `json:"streaming_rates,omitempty"`
	CDNRates             []ServiceRate         `json:"cdn_rates,omitempty"`
	Consumption          []CategoryConsumption `json:"consumption"`
	ClassifiedPct        float64               `json:"classified_pct"`
	Exporter             string                `json:"exporter"`
	Source               string                `json:"source"`
	SNMP                 *SNMPSnapshot         `json:"snmp,omitempty"`
	BGP                  *BGPSnapshot          `json:"bgp,omitempty"`
	PeerBreakdown        []DestOriginCard      `json:"peer_breakdown"`
	ASNBreakdown         []ASNTraffic          `json:"asn_breakdown,omitempty"`
	Sampling             *SamplingEstimate     `json:"sampling,omitempty"`
	Alerts               []Alert               `json:"alerts,omitempty"`
	MbpsScaled           float64               `json:"mbps_scaled"`
	IPv4Mbps             float64               `json:"ipv4_mbps,omitempty"`
	IPv6Mbps             float64               `json:"ipv6_mbps,omitempty"`
}

type rateSample struct {
	at        time.Time
	bytes     int64
	packets   int64
	category  string
	service   string
	peerASN   uint32
	dstASN    uint32
	direction string
	ifaceRole string
	ipVersion string
}

type HistoryPoint struct {
	Ts               int64              `json:"ts"`
	Mbps             float64            `json:"mbps"`
	MbpsScaled       float64            `json:"mbps_scaled,omitempty"`
	ByCategory       map[string]float64 `json:"by_category_mbps"`
	ByCategoryScaled map[string]float64 `json:"by_category_mbps_scaled,omitempty"`
	ByASN            map[string]float64 `json:"by_asn_mbps,omitempty"`
	ByASNScaled      map[string]float64 `json:"by_asn_mbps_scaled,omitempty"`
	SNMPIn           float64            `json:"snmp_in_mbps"`
	SNMPOut          float64            `json:"snmp_out_mbps"`
	SamplingFactor   float64            `json:"sampling_factor,omitempty"`
}

type FlowStore struct {
	mu           sync.RWMutex
	flows        []FlowRecord
	stats        AggregatedStats
	window       []rateSample
	seq          uint64
	started      time.Time
	history      []HistoryPoint
	peerASNBytes map[uint32]int64
	peerASNFlows map[uint32]int64
	dstASNBytes  map[uint32]int64
	dstASNFlows  map[uint32]int64
}

var store = &FlowStore{
	flows:        make([]FlowRecord, 0, MaxHistory),
	history:      make([]HistoryPoint, 0, 120),
	started:      time.Now(),
	peerASNBytes: make(map[uint32]int64),
	peerASNFlows: make(map[uint32]int64),
	dstASNBytes:  make(map[uint32]int64),
	dstASNFlows:  make(map[uint32]int64),
	stats: AggregatedStats{
		ByCategory:     make(map[string]int64),
		ByCategoryMbps: make(map[string]float64),
		ByDestination:  make(map[string]int64),
		ByOrigin:       make(map[string]int64),
		CDNBreakdown:   make(map[string]int64),
		StreamingBreak: make(map[string]int64),
		Exporter:       SourceIP,
		Source:         "netflow+snmp+bgp",
	},
}

var services = []struct {
	Name     string
	Category FlowCategory
	Icon     string
}{
	{"Netflix", CategoryNetflix, "netflix"},
	{"Globo", CategoryGlobo, "globo"},
	{"Cloudflare", CategoryCDN, "cloudflare"},
	{"Akamai", CategoryCDN, "akamai"},
	{"Fastly", CategoryCDN, "fastly"},
	{"AWS CloudFront", CategoryCDN, "cloudfront"},
	{"CDN77", CategoryCDN, "cdn"},
	{"BunnyCDN", CategoryCDN, "cdn"},
	{"Edgecast", CategoryCDN, "cdn"},
	{"Limelight", CategoryCDN, "cdn"},
	{"Imperva", CategoryCDN, "cdn"},
	{"Cachefly", CategoryCDN, "cdn"},
	{"G-Core", CategoryCDN, "cdn"},
	{"QUIC.cloud", CategoryCDN, "cdn"},
	{"Azure CDN", CategoryCDN, "cdn"},
	{"Google Cache", CategoryCDN, "cdn"},
	{"YouTube", CategoryStreaming, "youtube"},
	{"Spotify", CategoryStreaming, "spotify"},
	{"Twitch", CategoryStreaming, "twitch"},
	{"Disney+", CategoryStreaming, "disney"},
	{"HBO Max", CategoryStreaming, "hbo"},
	{"Meta / WhatsApp", CategorySocial, "social"},
	{"TikTok", CategorySocial, "social"},
	{"Twitter / X", CategorySocial, "social"},
	{"Steam", CategoryGaming, "gaming"},
	{"Xbox Live", CategoryGaming, "gaming"},
	{"PlayStation", CategoryGaming, "gaming"},
	{"Riot Games", CategoryGaming, "gaming"},
	{"Epic Games", CategoryGaming, "gaming"},
	{"Apple", CategoryApple, "apple"},
	{"AWS", CategoryCloud, "cloud"},
	{"Google Cloud", CategoryCloud, "cloud"},
	{"Azure", CategoryCloud, "cloud"},
	{"DNS", CategoryDNS, "dns"},
	{"Peer IX.br", CategoryPeer, "peer"},
	{"Peer Telia", CategoryPeer, "peer"},
	{"Peer Cogent", CategoryPeer, "peer"},
	{"Peer HE", CategoryPeer, "peer"},
}

var categoryLabels = map[string]string{
	"cdn": "CDN", "netflix": "Netflix", "globo": "Globo",
	"streaming": "Streaming", "peer": "Peers / IX",
	"social": "Social", "gaming": "Games", "dns": "DNS",
	"cloud": "Cloud", "apple": "Apple", "other": "Outros",
}

func looksLikeIP(s string) bool { return net.ParseIP(s) != nil }

func serviceNameFromFlow(f FlowRecord) string {
	if f.Destination != "" && !looksLikeIP(f.Destination) && f.Destination != SourceIP {
		return f.Destination
	}
	if f.Origin != "" && !looksLikeIP(f.Origin) && f.Origin != SourceIP {
		return f.Origin
	}
	return ""
}

func (s *FlowStore) AddFlow(f FlowRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.flows = append([]FlowRecord{f}, s.flows...)
	if len(s.flows) > MaxHistory {
		s.flows = s.flows[:MaxHistory]
	}

	s.stats.TotalBytes += f.Bytes
	s.stats.TotalPackets += f.Packets
	s.stats.TotalFlows++

	catKey := string(f.Category)
	s.stats.ByCategory[catKey] += f.Bytes

	svc := serviceNameFromFlow(f)
	if svc != "" {
		s.stats.ByDestination[svc] += f.Bytes
		s.stats.ByOrigin[svc] += f.Bytes
		if f.Category == CategoryCDN {
			s.stats.CDNBreakdown[svc] += f.Bytes
		}
		if f.Category == CategoryStreaming || f.Category == CategoryNetflix || f.Category == CategoryGlobo {
			s.stats.StreamingBreak[svc] += f.Bytes
		}
	} else {
		if isAggregateLabel(f.Destination) {
			s.stats.ByDestination[f.Destination] += f.Bytes
		}
		if isAggregateLabel(f.Origin) {
			s.stats.ByOrigin[f.Origin] += f.Bytes
		}
	}

	now := time.Now()
	peerAS := parseASNNum(f.PeerASN)
	dstAS := parseASNNum(f.DstASN)
	if dstAS == 0 {
		// Fallback: ASN do endpoint remoto quando o flow é outbound
		if f.Direction == "outbound" {
			dstAS = parseASNNum(f.ASN)
		}
	}
	if !isVerifiedASN(dstAS) {
		dstAS = 0
	}
	s.window = append(s.window, rateSample{
		now, f.Bytes, f.Packets, catKey, svc, peerAS, dstAS, f.Direction, f.IfaceRole, f.IPVersion,
	})
	if peerAS > 0 {
		s.peerASNBytes[peerAS] += f.Bytes
		s.peerASNFlows[peerAS]++
	}
	if dstAS > 0 {
		s.dstASNBytes[dstAS] += f.Bytes
		s.dstASNFlows[dstAS]++
	}
	cutoff := now.Add(-10 * time.Second)
	i := 0
	for i < len(s.window) && s.window[i].at.Before(cutoff) {
		i++
	}
	if i > 0 {
		s.window = s.window[i:]
	}

	var wb, wp int64
	catBytes := map[string]int64{}
	catIn := map[string]int64{}
	catOut := map[string]int64{}
	roleBytes := map[string]int64{}
	svcBytes := map[string]int64{}
	for _, sm := range s.window {
		wb += sm.bytes
		wp += sm.packets
		catBytes[sm.category] += sm.bytes
		if sm.direction == "inbound" {
			catIn[sm.category] += sm.bytes
		} else {
			catOut[sm.category] += sm.bytes
		}
		if sm.ifaceRole != "" {
			roleBytes[sm.ifaceRole] += sm.bytes
		}
		if sm.service != "" {
			svcBytes[sm.service] += sm.bytes
		}
	}
	const windowSec = 10.0
	s.stats.BytesPerSec = float64(wb) / windowSec
	s.stats.PacketsPerSec = float64(wp) / windowSec
	s.stats.Mbps = s.stats.BytesPerSec * 8 / 1e6
	s.stats.ByCategoryMbps = make(map[string]float64, len(catBytes))
	for k, v := range catBytes {
		s.stats.ByCategoryMbps[k] = float64(v) * 8 / windowSec / 1e6
	}
	s.stats.ByCategoryInMbps = make(map[string]float64, len(catIn))
	for k, v := range catIn {
		s.stats.ByCategoryInMbps[k] = float64(v) * 8 / windowSec / 1e6
	}
	s.stats.ByCategoryOutMbps = make(map[string]float64, len(catOut))
	for k, v := range catOut {
		s.stats.ByCategoryOutMbps[k] = float64(v) * 8 / windowSec / 1e6
	}
	s.stats.ByIfaceRole = make(map[string]float64, len(roleBytes))
	for k, v := range roleBytes {
		s.stats.ByIfaceRole[k] = float64(v) * 8 / windowSec / 1e6
	}
	s.rebuildConsumptionLocked(svcBytes)
}

func (s *FlowStore) rebuildConsumptionLocked(svcBytes map[string]int64) {
	total := s.stats.TotalBytes
	if total == 0 {
		total = 1
	}
	var classified int64
	cons := make([]CategoryConsumption, 0, len(s.stats.ByCategory))
	for cat, bytes := range s.stats.ByCategory {
		if cat != "other" {
			classified += bytes
		}
		label := categoryLabels[cat]
		if label == "" {
			label = cat
		}
		cons = append(cons, CategoryConsumption{
			Category:   cat,
			Bytes:      bytes,
			Percentage: float64(bytes) / float64(total) * 100,
			Mbps:       s.stats.ByCategoryMbps[cat],
			Label:      label,
		})
	}
	sort.Slice(cons, func(i, j int) bool { return cons[i].Bytes > cons[j].Bytes })
	s.stats.Consumption = cons
	s.stats.ClassifiedPct = float64(classified) / float64(total) * 100

	// annotate top cards with instantaneous mbps from svc window
	_ = svcBytes
}

func (s *FlowStore) rebuildTopCards(svcBytes map[string]int64) {
	total := s.stats.TotalBytes
	if total == 0 {
		total = 1
	}
	const windowSec = 10.0

	type kv struct {
		key  string
		val  int64
		named bool
	}
	toSorted := func(m map[string]int64) []kv {
		out := make([]kv, 0, len(m))
		for k, v := range m {
			if k == "" || k == SourceIP {
				continue
			}
			named := !looksLikeIP(k)
			out = append(out, kv{k, v, named})
		}
		sort.Slice(out, func(i, j int) bool {
			if out[i].named != out[j].named {
				return out[i].named
			}
			return out[i].val > out[j].val
		})
		return out
	}

	lookup := func(name string) (cat, icon string) {
		cat, icon = "other", "default"
		for _, svc := range services {
			if svc.Name == name {
				return string(svc.Category), svc.Icon
			}
		}
		c := classifyIP(net.ParseIP(name))
		if c.Category != CategoryOther {
			return string(c.Category), c.Icon
		}
		return cat, icon
	}

	s.stats.TopDestinations = nil
	for i, d := range toSorted(s.stats.ByDestination) {
		if i >= 10 {
			break
		}
		cat, icon := lookup(d.key)
		mbps := float64(svcBytes[d.key]) * 8 / windowSec / 1e6
		s.stats.TopDestinations = append(s.stats.TopDestinations, DestOriginCard{
			Name: d.key, Bytes: d.val, Mbps: mbps,
			Percentage: float64(d.val) / float64(total) * 100,
			Category:   cat, Icon: icon,
		})
	}

	s.stats.TopOrigins = nil
	for i, o := range toSorted(s.stats.ByOrigin) {
		if i >= 10 {
			break
		}
		cat, icon := lookup(o.key)
		mbps := float64(svcBytes[o.key]) * 8 / windowSec / 1e6
		s.stats.TopOrigins = append(s.stats.TopOrigins, DestOriginCard{
			Name: o.key, Bytes: o.val, Mbps: mbps,
			Percentage: float64(o.val) / float64(total) * 100,
			Category:   cat, Icon: icon,
		})
	}
}

func (s *FlowStore) GetStats() AggregatedStats {
	s.mu.Lock()
	defer s.mu.Unlock()

	svcBytes := map[string]int64{}
	svcIn := map[string]int64{}
	svcOut := map[string]int64{}
	var v4Bytes, v6Bytes int64
	for _, sm := range s.window {
		if sm.ipVersion == "6" {
			v6Bytes += sm.bytes
		} else {
			v4Bytes += sm.bytes
		}
		if sm.service != "" {
			svcBytes[sm.service] += sm.bytes
			if sm.direction == "inbound" {
				svcIn[sm.service] += sm.bytes
			} else {
				svcOut[sm.service] += sm.bytes
			}
		}
	}
	s.rebuildConsumptionLocked(svcBytes)
	s.rebuildTopCards(svcBytes)

	out := s.stats
	out.ByCategory = copyMap(s.stats.ByCategory)
	out.ByCategoryMbps = copyFloatMap(s.stats.ByCategoryMbps)
	out.ByCategoryInMbps = copyFloatMap(s.stats.ByCategoryInMbps)
	out.ByCategoryOutMbps = copyFloatMap(s.stats.ByCategoryOutMbps)
	out.ByIfaceRole = copyFloatMap(s.stats.ByIfaceRole)
	out.ByDestination = copyMap(s.stats.ByDestination)
	out.ByOrigin = copyMap(s.stats.ByOrigin)
	out.CDNBreakdown = copyMap(s.stats.CDNBreakdown)
	out.StreamingBreak = copyMap(s.stats.StreamingBreak)
	out.TopDestinations = append([]DestOriginCard(nil), s.stats.TopDestinations...)
	out.TopOrigins = append([]DestOriginCard(nil), s.stats.TopOrigins...)
	out.Consumption = append([]CategoryConsumption(nil), s.stats.Consumption...)

	snmp := snmpStore.Get()
	out.SNMP = &snmp

	bgp := bgpStore.Get()
	peerWin := map[uint32]int64{}
	for _, sm := range s.window {
		if sm.peerASN > 0 {
			peerWin[sm.peerASN] += sm.bytes
		}
	}
	bgp.Peers = annotatePeersWithTraffic(bgp.Peers, s.peerASNBytes, peerWin, s.peerASNFlows)
	out.BGP = &bgp
	out.PeerBreakdown = buildPeerBreakdown(bgp.Peers, s.stats.TotalBytes)

	dstWin := map[uint32]int64{}
	dstIn := map[uint32]int64{}
	dstOut := map[uint32]int64{}
	dstV4 := map[uint32]int64{}
	dstV6 := map[uint32]int64{}
	for _, sm := range s.window {
		if sm.dstASN == 0 {
			continue
		}
		dstWin[sm.dstASN] += sm.bytes
		if sm.direction == "inbound" {
			dstIn[sm.dstASN] += sm.bytes
		} else {
			dstOut[sm.dstASN] += sm.bytes
		}
		if sm.ipVersion == "6" {
			dstV6[sm.dstASN] += sm.bytes
		} else {
			dstV4[sm.dstASN] += sm.bytes
		}
	}
	// Inclui ASNs de peers BGP com tráfego (ex.: AS269096 N&K) mesmo quando
	// o DstAS do flow aponta ao destino final e não ao peer de interconexão.
	asnBytes := copyMapUint(s.dstASNBytes)
	asnFlows := copyMapUint(s.dstASNFlows)
	for as, b := range s.peerASNBytes {
		if !isVerifiedASN(as) || b <= 0 {
			continue
		}
		if asnBytes[as] == 0 {
			asnBytes[as] = b
			asnFlows[as] = s.peerASNFlows[as]
		}
		if dstWin[as] == 0 {
			dstWin[as] = peerWin[as]
		}
	}

	snmpAvg := (snmp.UplinkInMbps + snmp.UplinkOutMbps) / 2
	sampling.Update(s.stats.Mbps, snmpAvg)
	samp := sampling.Get()
	out.Sampling = &samp
	out.MbpsScaled = samp.ScaledMbps
	out.Alerts = alerts.Active()

	// Escala Mbps por categoria para aproximar tráfego real (proporção NetFlow × fator SNMP)
	eff := samp.Effective
	if eff < 1 {
		eff = 1
	}
	const windowSec = 10.0
	out.IPv4Mbps = float64(v4Bytes) * 8 / windowSec / 1e6 * eff
	out.IPv6Mbps = float64(v6Bytes) * 8 / windowSec / 1e6 * eff

	scaledCons := make([]CategoryConsumption, len(out.Consumption))
	copy(scaledCons, out.Consumption)
	for i := range scaledCons {
		scaledCons[i].MbpsScaled = scaledCons[i].Mbps * eff
	}
	out.Consumption = scaledCons

	scaledCatMbps := make(map[string]float64, len(out.ByCategoryMbps))
	for k, v := range out.ByCategoryMbps {
		scaledCatMbps[k] = v * eff
	}
	out.ByCategoryMbpsScaled = scaledCatMbps
	out.TopTalkers = talkers.Top(15)
	if out.ByCategoryInMbps != nil {
		inScaled := make(map[string]float64, len(out.ByCategoryInMbps))
		outScaled := make(map[string]float64, len(out.ByCategoryOutMbps))
		for k, v := range out.ByCategoryInMbps {
			inScaled[k] = v * eff
		}
		for k, v := range out.ByCategoryOutMbps {
			outScaled[k] = v * eff
		}
		out.ByCategoryInMbps = inScaled
		out.ByCategoryOutMbps = outScaled
	}
	if out.ByIfaceRole != nil {
		roleScaled := make(map[string]float64, len(out.ByIfaceRole))
		for k, v := range out.ByIfaceRole {
			roleScaled[k] = v * eff
		}
		out.ByIfaceRole = roleScaled
	}

	lookupCat := func(name string) string {
		for _, svc := range services {
			if svc.Name == name {
				return string(svc.Category)
			}
		}
		c := classifyIP(net.ParseIP(name))
		return string(c.Category)
	}
	out.StreamingRates = buildServiceRatesList(out.StreamingBreak, svcBytes, svcIn, svcOut, lookupCat, eff,
		[]string{"streaming", "netflix", "globo"})
	out.CDNRates = buildServiceRatesList(out.CDNBreakdown, svcBytes, svcIn, svcOut, lookupCat, eff,
		[]string{"cdn"})
	out.ASNBreakdown = buildASNBreakdown(asnBytes, asnFlows, dstWin, dstIn, dstOut, dstV4, dstV6, s.stats.TotalBytes, eff)
	for i := range out.TopDestinations {
		out.TopDestinations[i].MbpsScaled = out.TopDestinations[i].Mbps * eff
	}
	for i := range out.TopOrigins {
		out.TopOrigins[i].MbpsScaled = out.TopOrigins[i].Mbps * eff
	}
	return out
}

func buildServiceRatesList(
	breakdown map[string]int64,
	svcBytes, svcIn, svcOut map[string]int64,
	lookup func(string) string,
	eff float64,
	categories []string,
) []ServiceRate {
	const windowSec = 10.0
	catSet := make(map[string]bool, len(categories))
	for _, c := range categories {
		catSet[c] = true
	}
	names := make(map[string]struct{})
	for k, v := range breakdown {
		if v > 0 {
			names[k] = struct{}{}
		}
	}
	for name := range svcBytes {
		if catSet[lookup(name)] {
			names[name] = struct{}{}
		}
	}
	out := make([]ServiceRate, 0, len(names))
	for name := range names {
		mbps := float64(svcBytes[name]) * 8 / windowSec / 1e6
		inM := float64(svcIn[name]) * 8 / windowSec / 1e6
		outM := float64(svcOut[name]) * 8 / windowSec / 1e6
		out = append(out, ServiceRate{
			Name:       name,
			Bytes:      breakdown[name],
			Mbps:       mbps,
			MbpsScaled: mbps * eff,
			InMbps:     inM * eff,
			OutMbps:    outM * eff,
			Category:   lookup(name),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].MbpsScaled != out[j].MbpsScaled {
			return out[i].MbpsScaled > out[j].MbpsScaled
		}
		return out[i].Bytes > out[j].Bytes
	})
	return out
}

func buildASNBreakdown(
	byBytes, byFlows map[uint32]int64,
	winBytes, inBytes, outBytes, v4Bytes, v6Bytes map[uint32]int64,
	total int64,
	eff float64,
) []ASNTraffic {
	const windowSec = 10.0
	if total <= 0 {
		total = 1
	}
	if eff < 1 {
		eff = 1
	}
	seen := map[uint32]struct{}{}
	for as := range byBytes {
		seen[as] = struct{}{}
	}
	for as := range winBytes {
		seen[as] = struct{}{}
	}
	out := make([]ASNTraffic, 0, len(seen))
	for as := range seen {
		if !isVerifiedASN(as) {
			continue
		}
		bytes := byBytes[as]
		win := winBytes[as]
		if bytes == 0 && win == 0 {
			continue
		}
		mbps := float64(win) * 8 / windowSec / 1e6
		inM := float64(inBytes[as]) * 8 / windowSec / 1e6
		outM := float64(outBytes[as]) * 8 / windowSec / 1e6
		v4M := float64(v4Bytes[as]) * 8 / windowSec / 1e6
		v6M := float64(v6Bytes[as]) * 8 / windowSec / 1e6
		name := asnDisplayName(as)
		cat := "other"
		icon := "peer"
		if c := classifyByASN(as); c.Name != "" {
			cat = string(c.Category)
			icon = c.Icon
			if strings.HasPrefix(name, "AS") {
				name = c.Name
			}
		} else if p, ok := bgpStore.LookupAS(as); ok {
			name = p.Name
			cat = "peer"
		} else if n := ipapi.NameForASN(as); n != "" {
			name = n
		}
		out = append(out, ASNTraffic{
			ASN:        fmt.Sprintf("AS%d", as),
			Name:       name,
			Bytes:      bytes,
			Flows:      byFlows[as],
			Mbps:       mbps,
			MbpsScaled: mbps * eff,
			InMbps:     inM * eff,
			OutMbps:    outM * eff,
			IPv4Mbps:   v4M * eff,
			IPv6Mbps:   v6M * eff,
			Percentage: float64(bytes) / float64(total) * 100,
			Category:   cat,
			Icon:       icon,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].MbpsScaled != out[j].MbpsScaled {
			return out[i].MbpsScaled > out[j].MbpsScaled
		}
		return out[i].Bytes > out[j].Bytes
	})
	if len(out) > 40 {
		out = out[:40]
	}
	return out
}

// asnWindowScaledMbps — Mbps estimado por ASN (janela) para alertas.
func (s *FlowStore) asnWindowScaledMbps() map[string]float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	win := map[uint32]int64{}
	for _, sm := range s.window {
		if sm.dstASN > 0 {
			win[sm.dstASN] += sm.bytes
		}
	}
	eff := sampling.Get().Effective
	if eff < 1 {
		eff = 1
	}
	const windowSec = 10.0
	out := make(map[string]float64, len(win))
	for as, b := range win {
		out[fmt.Sprintf("AS%d", as)] = float64(b) * 8 / windowSec / 1e6 * eff
	}
	return out
}

// GetStatsLite — leitura leve para alertas (sem anotar BGP/sampling pesado).
func (s *FlowStore) GetStatsLite() AggregatedStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := AggregatedStats{
		Mbps:           s.stats.Mbps,
		ByCategoryMbps: copyFloatMap(s.stats.ByCategoryMbps),
	}
	return out
}

func buildPeerBreakdown(peers []BGPPeer, total int64) []DestOriginCard {
	if total <= 0 {
		total = 1
	}
	// Aggregate by ASN (multiple sessions per AS)
	type agg struct {
		name, asn, role string
		bytes           int64
		mbps            float64
		est             bool
	}
	m := map[uint32]*agg{}
	for _, p := range peers {
		a := m[p.RemoteAS]
		if a == nil {
			a = &agg{name: p.Name, asn: p.ASN, role: p.Role}
			m[p.RemoteAS] = a
		}
		a.bytes += p.Bytes
		a.mbps += p.Mbps
		if p.Established {
			a.est = true
		}
	}
	out := make([]DestOriginCard, 0, len(m))
	for _, a := range m {
		if !a.est && a.bytes == 0 {
			continue
		}
		cat := "peer"
		if a.role == "content" {
			cat = "peer"
		}
		out = append(out, DestOriginCard{
			Name:       a.name + " (" + a.asn + ")",
			Bytes:      a.bytes,
			Mbps:       a.mbps,
			Percentage: float64(a.bytes) / float64(total) * 100,
			Category:   cat,
			Icon:       "peer",
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Bytes == out[j].Bytes {
			return out[i].Mbps > out[j].Mbps
		}
		return out[i].Bytes > out[j].Bytes
	})
	if len(out) > 16 {
		out = out[:16]
	}
	return out
}

func copyMap(m map[string]int64) map[string]int64 {
	out := make(map[string]int64, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func copyMapUint(m map[uint32]int64) map[uint32]int64 {
	out := make(map[uint32]int64, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func copyFloatMap(m map[string]float64) map[string]float64 {
	out := make(map[string]float64, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func (s *FlowStore) GetHistory() []HistoryPoint {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]HistoryPoint, len(s.history))
	copy(out, s.history)
	return out
}

func (s *FlowStore) pushHistory() {
	s.mu.Lock()
	defer s.mu.Unlock()
	cats := copyFloatMap(s.stats.ByCategoryMbps)
	snmp := snmpStore.Get()
	snmpAvg := (snmp.UplinkInMbps + snmp.UplinkOutMbps) / 2
	sampling.Update(s.stats.Mbps, snmpAvg)
	samp := sampling.Get()
	eff := samp.Effective
	if eff < 1 {
		eff = 1
	}
	scaledCats := make(map[string]float64, len(cats))
	for k, v := range cats {
		scaledCats[k] = v * eff
	}
	asnWin := map[uint32]int64{}
	var v4b, v6b int64
	for _, sm := range s.window {
		if sm.dstASN > 0 {
			asnWin[sm.dstASN] += sm.bytes
		}
		if sm.ipVersion == "6" {
			v6b += sm.bytes
		} else {
			v4b += sm.bytes
		}
	}
	const ws = 10.0
	byASN := make(map[string]float64, len(asnWin))
	byASNScaled := make(map[string]float64, len(asnWin))
	type asnKV struct {
		k string
		v float64
	}
	ranked := make([]asnKV, 0, len(asnWin))
	for as, b := range asnWin {
		mbps := float64(b) * 8 / ws / 1e6
		key := fmt.Sprintf("AS%d", as)
		ranked = append(ranked, asnKV{key, mbps})
	}
	sort.Slice(ranked, func(i, j int) bool { return ranked[i].v > ranked[j].v })
	if len(ranked) > 15 {
		ranked = ranked[:15]
	}
	for _, kv := range ranked {
		byASN[kv.k] = kv.v
		byASNScaled[kv.k] = kv.v * eff
	}
	pt := HistoryPoint{
		Ts:               time.Now().Unix(),
		Mbps:             s.stats.Mbps,
		MbpsScaled:       s.stats.Mbps * eff,
		ByCategory:       cats,
		ByCategoryScaled: scaledCats,
		ByASN:            byASN,
		ByASNScaled:      byASNScaled,
		SNMPIn:           snmp.UplinkInMbps,
		SNMPOut:          snmp.UplinkOutMbps,
		SamplingFactor:   eff,
	}
	s.history = append(s.history, pt)
	if len(s.history) > 120 {
		s.history = s.history[len(s.history)-120:]
	}
	go appendHistoryFile(pt)
	go storageInsert(pt, float64(v4b)*8/ws/1e6*eff, float64(v6b)*8/ws/1e6*eff)
	go persistASNDaily()
}

func startHistorySampler() {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		for range ticker.C {
			store.pushHistory()
		}
	}()
}

func (s *FlowStore) GetRecentFlows(limit int) []FlowRecord {
	return s.GetRecentFlowsFiltered(limit, "", "", "", "")
}

func normalizeASNQuery(asn string) string {
	asn = strings.ToUpper(strings.TrimSpace(asn))
	if asn == "" {
		return ""
	}
	asn = strings.TrimPrefix(asn, "AS")
	if asn == "" {
		return ""
	}
	return "AS" + asn
}

func (s *FlowStore) GetRecentFlowsFiltered(limit int, ip, category, q, asn string) []FlowRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ip = strings.ToLower(strings.TrimSpace(ip))
	category = strings.ToLower(strings.TrimSpace(category))
	q = strings.ToLower(strings.TrimSpace(q))
	asn = normalizeASNQuery(asn)
	var result []FlowRecord
	for _, f := range s.flows {
		if ip != "" && !strings.Contains(strings.ToLower(f.SrcIP), ip) && !strings.Contains(strings.ToLower(f.DstIP), ip) {
			continue
		}
		if category != "" && string(f.Category) != category {
			continue
		}
		if asn != "" {
			if normalizeASNQuery(f.DstASN) != asn && normalizeASNQuery(f.ASN) != asn && normalizeASNQuery(f.PeerASN) != asn {
				continue
			}
		}
		if q != "" {
			hay := strings.ToLower(f.SrcIP + f.DstIP + f.Destination + f.Origin + f.ASN + f.DstASN + f.PeerASN + f.PeerName)
			if !strings.Contains(hay, q) {
				continue
			}
		}
		result = append(result, f)
		if len(result) >= limit {
			break
		}
	}
	return result
}

func nextID() string {
	n := atomic.AddUint64(&store.seq, 1)
	return fmt.Sprintf("nf-%d-%d", time.Now().Unix(), n)
}

func resolveBGPPeer(src, dst net.IP, srcAS, dstAS uint32, asnLabel string) (peerASN, peerName, peerIP string) {
	if p, ok := bgpStore.LookupIP(src.String()); ok {
		return p.ASN, p.Name, p.RemoteAddr
	}
	if p, ok := bgpStore.LookupIP(dst.String()); ok {
		return p.ASN, p.Name, p.RemoteAddr
	}
	asCandidates := []uint32{dstAS, srcAS, parseASNNum(asnLabel)}
	for _, as := range asCandidates {
		if as == 0 {
			continue
		}
		if p, ok := bgpStore.LookupAS(as); ok {
			return p.ASN, p.Name, p.RemoteAddr
		}
	}
	return "", "", ""
}

func ingestRaw(raw RawFlow) {
	src := raw.SrcIP
	dst := raw.DstIP
	if src == nil || dst == nil {
		return
	}

	srcName, srcCat, _, srcASN := labelForEndpoint(src, raw.SrcAS)
	dstName, dstCat, _, dstASN := labelForEndpoint(dst, raw.DstAS)

	// Prefere ASN resolvido por IP (ip-api.com) — mais confiável que offsets NetFlow
	if as, name := ipapi.ASNForIP(dst); as > 0 {
		dstASN = fmt.Sprintf("AS%d", as)
		if name != "" && (looksLikeIP(dstName) || dstName == "" || dstCat == CategoryOther) {
			if classifyByASN(as).Name == "" {
				dstName = name
			}
		}
	} else {
		ipapi.Enqueue(dst)
	}
	if as, name := ipapi.ASNForIP(src); as > 0 {
		srcASN = fmt.Sprintf("AS%d", as)
		if name != "" && (looksLikeIP(srcName) || srcName == "") {
			if classifyByASN(as).Name == "" {
				srcName = name
			}
		}
	} else {
		ipapi.Enqueue(src)
	}
	if !isVerifiedASN(parseASNNum(dstASN)) {
		dstASN = ""
	}

	direction := "outbound"
	origin := srcName
	destination := dstName
	category := dstCat
	asn := dstASN
	if dstASN == "" {
		asn = srcASN
	}

	srcLocal := isPrivateOrSpecial(src)
	dstLocal := isPrivateOrSpecial(dst)

	if srcCat != CategoryOther && (dstLocal || dstCat == CategoryOther) {
		direction = "inbound"
		category = srcCat
		asn = srcASN
		origin = srcName
		destination = dst.String()
	} else if dstCat != CategoryOther && (srcLocal || srcCat == CategoryOther) {
		direction = "outbound"
		category = dstCat
		asn = dstASN
		origin = src.String()
		destination = dstName
	} else if srcCat != CategoryOther && dstCat != CategoryOther {
		if priority(srcCat) >= priority(dstCat) {
			direction = "inbound"
			category = srcCat
			asn = srcASN
			origin = srcName
			destination = dstName
		} else {
			direction = "outbound"
			category = dstCat
			asn = dstASN
			origin = srcName
			destination = dstName
		}
	} else if srcLocal && !dstLocal {
		direction = "outbound"
		origin = src.String()
		destination = dstName
		category = dstCat
	} else if dstLocal && !srcLocal {
		direction = "inbound"
		origin = srcName
		destination = dst.String()
		category = srcCat
	}

	// Port-based fallback (DNS, games)
	if category == CategoryOther {
		if pc := classifyByPort(raw.SrcPort, raw.DstPort); pc.Name != "" {
			category = pc.Category
			if direction == "outbound" {
				destination = pc.Name
			} else {
				origin = pc.Name
			}
			if asn == "" {
				asn = pc.ASN
			}
		}
	}

	bytes := int64(raw.Bytes)
	pkts := int64(raw.Packets)
	if bytes < 0 {
		bytes = 0
	}
	if pkts < 0 {
		pkts = 0
	}

	peerASN, peerName, peerIP := resolveBGPPeer(src, dst, raw.SrcAS, raw.DstAS, asn)
	if peerASN == "" {
		for _, cand := range []string{dstASN, srcASN, asn} {
			n := parseASNNum(cand)
			if n == 0 {
				continue
			}
			if p, ok := bgpStore.LookupAS(n); ok {
				peerASN, peerName, peerIP = p.ASN, p.Name, p.RemoteAddr
				break
			}
		}
	}
	// Next-hop como peer BGP
	if peerASN == "" && raw.NextHop != nil && !raw.NextHop.IsUnspecified() {
		if p, ok := bgpStore.LookupIP(raw.NextHop.String()); ok {
			peerASN, peerName, peerIP = p.ASN, p.Name, p.RemoteAddr
		}
	}

	cacheIface := lookupCacheIface(raw.InIf, raw.OutIf)
	ifaceName, ifaceRole := lookupIfaceMeta(raw.InIf, raw.OutIf, direction)
	nh := ""
	if raw.NextHop != nil && !raw.NextHop.IsUnspecified() {
		nh = raw.NextHop.String()
	}

	// Top talkers CGNAT
	if isCGNATClient(src) {
		talkers.Add(src.String(), string(category), bytes)
	} else if isCGNATClient(dst) {
		talkers.Add(dst.String(), string(category), bytes)
	}

	store.AddFlow(FlowRecord{
		ID: nextID(), Timestamp: time.Now().Unix(),
		SrcIP: src.String(), DstIP: dst.String(),
		SrcPort: int(raw.SrcPort), DstPort: int(raw.DstPort),
		Protocol: protoName(raw.Proto), Bytes: bytes, Packets: pkts,
		Direction: direction, Category: category,
		Destination: destination, Origin: origin, ASN: asn,
		DstASN: dstASN,
		PeerASN: peerASN, PeerName: peerName, PeerIP: peerIP,
		NextHop: nh, InIf: int(raw.InIf), OutIf: int(raw.OutIf),
		CacheIface: cacheIface, IfaceRole: ifaceRole, IfaceName: ifaceName,
		IPVersion: ipVersionOf(src, dst),
	})
}

func lookupIfaceMeta(inIf, outIf uint32, direction string) (name, role string) {
	snap := snmpStore.Get()
	prefer := outIf
	if direction == "inbound" {
		prefer = inIf
	}
	lookup := func(idx uint32) (string, string, bool) {
		if idx == 0 {
			return "", "", false
		}
		for _, iface := range snap.Interfaces {
			if uint32(iface.Index) == idx {
				n := iface.Alias
				if n == "" {
					n = iface.Name
				}
				return n, iface.Role, true
			}
		}
		return "", "", false
	}
	if n, r, ok := lookup(prefer); ok {
		return n, r
	}
	if n, r, ok := lookup(inIf); ok {
		return n, r
	}
	if n, r, ok := lookup(outIf); ok {
		return n, r
	}
	return "", ""
}

func lookupCacheIface(inIf, outIf uint32) string {
	snap := snmpStore.Get()
	for _, iface := range snap.Interfaces {
		if iface.Role != "cache" {
			continue
		}
		idx := uint32(iface.Index)
		if (inIf > 0 && idx == inIf) || (outIf > 0 && idx == outIf) {
			if iface.Alias != "" {
				return iface.Alias
			}
			return iface.Name
		}
	}
	return ""
}

func priority(c FlowCategory) int {
	switch c {
	case CategoryNetflix, CategoryGlobo:
		return 6
	case CategoryStreaming, CategorySocial, CategoryGaming, CategoryApple:
		return 5
	case CategoryCDN:
		return 4
	case CategoryCloud, CategoryDNS:
		return 3
	case CategoryPeer:
		return 2
	default:
		return 0
	}
}

func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-API-Token")
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		c := GetConfig()
		if c.APIToken != "" || uiAuthEnabled() {
			got := r.Header.Get("X-API-Token")
			if got == "" {
				auth := r.Header.Get("Authorization")
				if strings.HasPrefix(auth, "Bearer ") {
					got = strings.TrimPrefix(auth, "Bearer ")
				}
			}
			if got == "" {
				got = r.URL.Query().Get("token")
			}
			if !validSessionToken(got) {
				w.WriteHeader(http.StatusUnauthorized)
				writeJSON(w, map[string]string{"error": "unauthorized"})
				return
			}
		}
		next(w, r)
	}
}

func handleStats(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, store.GetStats())
}

func ipVersionOf(src, dst net.IP) string {
	if src != nil && src.To4() == nil && len(src) == net.IPv6len {
		return "6"
	}
	if dst != nil && dst.To4() == nil && len(dst) == net.IPv6len {
		return "6"
	}
	return "4"
}

func handleFlows(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	ip := r.URL.Query().Get("ip")
	category := r.URL.Query().Get("category")
	q := r.URL.Query().Get("q")
	asn := r.URL.Query().Get("asn")
	writeJSON(w, store.GetRecentFlowsFiltered(limit, ip, category, q, asn))
}

func handleSNMP(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, snmpStore.Get())
}

func handleBGP(w http.ResponseWriter, r *http.Request) {
	snap := bgpStore.Get()
	store.mu.RLock()
	peerWin := map[uint32]int64{}
	for _, sm := range store.window {
		if sm.peerASN > 0 {
			peerWin[sm.peerASN] += sm.bytes
		}
	}
	snap.Peers = annotatePeersWithTraffic(snap.Peers, store.peerASNBytes, peerWin, store.peerASNFlows)
	store.mu.RUnlock()
	writeJSON(w, snap)
}

func handleHistory(w http.ResponseWriter, r *http.Request) {
	since := int64(0)
	hasSince := false
	hasHours := false
	if v := r.URL.Query().Get("since"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			since = n
			hasSince = true
		}
	}
	if v := r.URL.Query().Get("hours"); v != "" {
		if h, err := strconv.Atoi(v); err == nil && h > 0 {
			since = time.Now().Unix() - int64(h)*3600
			hasHours = true
		}
	}
	// Ao vivo (sem filtro): só memória (~10 min) — evita carregar 30d do SQLite/S3.
	if !hasSince && !hasHours {
		writeJSON(w, store.GetHistory())
		return
	}
	pts := queryHistorySince(since)
	if len(pts) == 0 {
		mem := store.GetHistory()
		if since > 0 {
			for _, pt := range mem {
				if pt.Ts >= since {
					pts = append(pts, pt)
				}
			}
		} else {
			pts = mem
		}
	}
	writeJSON(w, pts)
}

func handleAlerts(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{
		"active": alerts.Active(),
		"recent": alerts.List(),
	})
}

func handleSampling(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, sampling.Get())
}

func handleTalkers(w http.ResponseWriter, r *http.Request) {
	n := 20
	if v := r.URL.Query().Get("limit"); v != "" {
		if x, err := strconv.Atoi(v); err == nil && x > 0 && x <= 100 {
			n = x
		}
	}
	writeJSON(w, talkers.Top(n))
}

func handleExport(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	kind := r.URL.Query().Get("kind")
	if kind == "" {
		kind = "stats"
	}
	stats := store.GetStats()

	if format == "csv" {
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename=inforflow-"+kind+".csv")
		switch kind {
		case "cdn":
			fmt.Fprintln(w, "name,bytes")
			for k, v := range stats.CDNBreakdown {
				fmt.Fprintf(w, "%s,%d\n", k, v)
			}
		case "streaming":
			fmt.Fprintln(w, "name,bytes")
			for k, v := range stats.StreamingBreak {
				fmt.Fprintf(w, "%s,%d\n", k, v)
			}
		case "peers":
			fmt.Fprintln(w, "name,bytes,mbps")
			for _, p := range stats.PeerBreakdown {
				fmt.Fprintf(w, "%s,%d,%.4f\n", p.Name, p.Bytes, p.Mbps)
			}
		case "asn":
			fmt.Fprintln(w, "asn,name,bytes,flows,mbps,mbps_scaled,percentage")
			for _, a := range stats.ASNBreakdown {
				fmt.Fprintf(w, "%s,%s,%d,%d,%.4f,%.4f,%.2f\n",
					a.ASN, a.Name, a.Bytes, a.Flows, a.Mbps, a.MbpsScaled, a.Percentage)
			}
		case "talkers":
			fmt.Fprintln(w, "ip,bytes,mbps_scaled,top_category,flows")
			for _, t := range talkers.Top(50) {
				fmt.Fprintf(w, "%s,%d,%.4f,%s,%d\n", t.IP, t.Bytes, t.MbpsScaled, t.TopCat, t.Flows)
			}
		case "flows":
			fmt.Fprintln(w, "ts,src,dst,category,bytes,asn,peer_asn")
			for _, f := range store.GetRecentFlows(200) {
				fmt.Fprintf(w, "%d,%s,%s,%s,%d,%s,%s\n",
					f.Timestamp, f.SrcIP, f.DstIP, f.Category, f.Bytes, f.ASN, f.PeerASN)
			}
		default:
			fmt.Fprintln(w, "category,bytes,mbps")
			for _, c := range stats.Consumption {
				fmt.Fprintf(w, "%s,%d,%.4f\n", c.Category, c.Bytes, c.Mbps)
			}
		}
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename=inforflow-"+kind+".json")
	switch kind {
	case "cdn":
		writeJSON(w, stats.CDNBreakdown)
	case "streaming":
		writeJSON(w, stats.StreamingBreak)
	case "peers":
		writeJSON(w, stats.PeerBreakdown)
	case "asn":
		writeJSON(w, stats.ASNBreakdown)
	case "flows":
		writeJSON(w, store.GetRecentFlows(200))
	case "bgp":
		writeJSON(w, bgpStore.Get())
	default:
		writeJSON(w, stats)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	snmp := snmpStore.Get()
	bgp := bgpStore.Get()
	samp := sampling.Get()
	writeJSON(w, map[string]interface{}{
		"status":       "ok",
		"source_ip":    SourceIP,
		"service":      "inforflow-collector",
		"mode":         "netflow+snmp+bgp",
		"listen":       NetFlowPort,
		"snmp":         snmp.OK,
		"bgp":          bgp.OK,
		"bgp_peers":    bgp.Established,
		"sys_name":     snmp.SysName,
		"uptime_s":     int(time.Since(store.started).Seconds()),
		"flows":        atomic.LoadUint64(&store.seq),
		"sampling":     samp.Effective,
		"mbps_scaled":  samp.ScaledMbps,
		"alerts":       len(alerts.Active()),
		"auth_enabled": GetConfig().APIToken != "" || uiAuthEnabled(),
		"ipapi":        ipapi.Stats(),
	})
}

func main() {
	LoadConfig("")
	c := GetConfig()
	APIPort = c.APIPort
	NetFlowPort = c.NetFlowPort

	if len(c.Exporters) > 0 {
		SourceIP = c.Exporters[0]
	}
	log.Printf("Inforflow Collector — NetFlow + SNMP + BGP de %s", SourceIP)
	if len(c.Exporters) > 1 {
		log.Printf("exporters adicionais: %v", c.Exporters[1:])
	}
	log.Printf("NetFlow UDP %s | SNMP %s:%d | API %s | data %s | local %dh · S3 %dd",
		NetFlowPort, SNMPHost, SNMPPort, APIPort, c.DataDir, c.HistoryLocalH, c.HistoryS3Days)
	if c.APIToken != "" {
		log.Printf("API auth: token habilitado")
	}

	initStorage()
	loadNativeSampling()
	loadASNDaily()
	StartIPAPIResolver()
	StartSNMPPoller()
	StartBGPPoller()
	StartPrefixFeeds()
	StartAlertEvaluator()
	StartAnomalyEvaluator()
	startHistorySampler()
	StartS3Sync()
	go func() {
		for {
			time.Sleep(6 * time.Hour)
			pruneHistoryFile()
		}
	}()
	go func() {
		time.Sleep(time.Minute)
		for {
			storagePruneLocal()
			time.Sleep(time.Hour)
		}
	}()
	go StartNetFlowListener(ingestRaw)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", handleHealth)
	mux.HandleFunc("/metrics", handleMetrics)
	mux.HandleFunc("/api/stats", requireAuth(handleStats))
	mux.HandleFunc("/api/flows", requireAuth(handleFlows))
	mux.HandleFunc("/api/snmp", requireAuth(handleSNMP))
	mux.HandleFunc("/api/bgp", requireAuth(handleBGP))
	mux.HandleFunc("/api/history", requireAuth(handleHistory))
	mux.HandleFunc("/api/history/compare", requireAuth(handleHistoryCompare))
	mux.HandleFunc("/api/alerts", requireAuth(handleAlerts))
	mux.HandleFunc("/api/sampling", requireAuth(handleSampling))
	mux.HandleFunc("/api/talkers", requireAuth(handleTalkers))
	mux.HandleFunc("/api/cache", requireAuth(handleCache))
	mux.HandleFunc("/api/storage", requireAuth(handleStorage))
	mux.HandleFunc("/api/ipapi", requireAuth(handleIPAPI))
	mux.HandleFunc("/api/export", requireAuth(handleExport))
	mux.HandleFunc("/api/login", handleLogin)
	mux.HandleFunc("/api/logout", handleLogout)
	mux.HandleFunc("/api/auth/check", handleAuthCheck)

	log.Fatal(http.ListenAndServe(APIPort, mux))
}
