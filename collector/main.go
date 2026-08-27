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
	MaxHistory = 2000
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
	IPv4Mbps   float64 `json:"ipv4_mbps,omitempty"`
	IPv6Mbps   float64 `json:"ipv6_mbps,omitempty"`
	Percentage float64 `json:"percentage,omitempty"`
	Category   string  `json:"category"`
}

// ASNTraffic — tráfego agregado por ASN (destino ou peer BGP).
type ASNTraffic struct {
	ASN        string  `json:"asn"`
	Name       string  `json:"name"`
	Role       string  `json:"role"` // "destination" | "peer"
	Bytes      int64   `json:"bytes"`
	Flows      int64   `json:"flows"`
	Mbps       float64 `json:"mbps"`              // NetFlow amostrado (~10s)
	MbpsScaled float64 `json:"mbps_scaled"`       // estimado × sampling
	InMbps     float64 `json:"in_mbps,omitempty"`
	OutMbps    float64 `json:"out_mbps,omitempty"`
	IPv4Mbps   float64 `json:"ipv4_mbps,omitempty"`
	IPv6Mbps   float64 `json:"ipv6_mbps,omitempty"`
	Percentage float64 `json:"percentage"` // % dos bytes acumulados do dia
	Category   string  `json:"category"`
	Icon       string  `json:"icon"`
	Pending    bool    `json:"pending,omitempty"`
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
	ByDestination        map[string]int64      `json:"by_destination,omitempty"`
	ByOrigin             map[string]int64      `json:"by_origin,omitempty"`
	DestinationCount     int                   `json:"destination_count,omitempty"`
	OriginCount          int                   `json:"origin_count,omitempty"`
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
	PeerASNBreakdown     []ASNTraffic          `json:"peer_asn_breakdown,omitempty"`
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
	ByPeerASN        map[string]float64 `json:"by_peer_asn_mbps,omitempty"`
	ByPeerASNScaled  map[string]float64 `json:"by_peer_asn_mbps_scaled,omitempty"`
	ByStreaming      map[string]float64 `json:"by_streaming_mbps,omitempty"`
	ByStreamingScaled map[string]float64 `json:"by_streaming_mbps_scaled,omitempty"`
	ByCDN            map[string]float64 `json:"by_cdn_mbps,omitempty"`
	ByCDNScaled      map[string]float64 `json:"by_cdn_mbps_scaled,omitempty"`
	BySNMPRole       map[string]float64 `json:"by_snmp_role_mbps,omitempty"` // soma in+out por role
	IPv4Mbps         float64            `json:"ipv4_mbps,omitempty"`
	IPv6Mbps         float64            `json:"ipv6_mbps,omitempty"`
	SNMPIn           float64            `json:"snmp_in_mbps"`
	SNMPOut          float64            `json:"snmp_out_mbps"`
	SamplingFactor   float64            `json:"sampling_factor,omitempty"`
}

type FlowStore struct {
	mu             sync.RWMutex
	flowsBuf       []FlowRecord
	flowsPos       int
	flowsCount     int
	stats          AggregatedStats
	window         []rateSample
	winBytes       int64
	winPkts        int64
	winCatBytes    map[string]int64
	winCatIn       map[string]int64
	winCatOut      map[string]int64
	winRoleBytes   map[string]int64
	winSvcBytes    map[string]int64
	seq            uint64
	started        time.Time
	history        []HistoryPoint
	peerASNBytes   map[uint32]int64
	peerASNFlows   map[uint32]int64
	dstASNBytes    map[uint32]int64
	dstASNFlows    map[uint32]int64
	asnRecentFlows map[uint32][]FlowRecord // últimos flows por ASN destino
	asnRecentPos   map[uint32]int
}

var store = &FlowStore{
	flowsBuf:       make([]FlowRecord, MaxHistory),
	history:        make([]HistoryPoint, 0, 120),
	started:        time.Now(),
	winCatBytes:    make(map[string]int64),
	winCatIn:       make(map[string]int64),
	winCatOut:      make(map[string]int64),
	winRoleBytes:   make(map[string]int64),
	winSvcBytes:    make(map[string]int64),
	peerASNBytes:   make(map[uint32]int64),
	peerASNFlows:   make(map[uint32]int64),
	dstASNBytes:    make(map[uint32]int64),
	dstASNFlows:    make(map[uint32]int64),
	asnRecentFlows: make(map[uint32][]FlowRecord),
	asnRecentPos:   make(map[uint32]int),
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

func (s *FlowStore) pushFlowLocked(f FlowRecord) {
	s.flowsBuf[s.flowsPos] = f
	s.flowsPos = (s.flowsPos + 1) % MaxHistory
	if s.flowsCount < MaxHistory {
		s.flowsCount++
	}
}

func (s *FlowStore) pushASNRecentLocked(as uint32, f FlowRecord) {
	const capN = 80
	ring := s.asnRecentFlows[as]
	if ring == nil {
		ring = make([]FlowRecord, capN)
		s.asnRecentFlows[as] = ring
		s.asnRecentPos[as] = 0
	}
	pos := s.asnRecentPos[as]
	ring[pos%capN] = f
	s.asnRecentPos[as] = pos + 1
	if len(s.asnRecentFlows) > 120 {
		s.pruneASNRecentLocked()
	}
}

func (s *FlowStore) applyWindowSampleLocked(sm rateSample, sign int64) {
	s.winBytes += sign * sm.bytes
	s.winPkts += sign * sm.packets
	s.winCatBytes[sm.category] += sign * sm.bytes
	if sm.direction == "inbound" {
		s.winCatIn[sm.category] += sign * sm.bytes
	} else {
		s.winCatOut[sm.category] += sign * sm.bytes
	}
	if sm.ifaceRole != "" {
		s.winRoleBytes[sm.ifaceRole] += sign * sm.bytes
	}
	if sm.service != "" {
		s.winSvcBytes[sm.service] += sign * sm.bytes
	}
}

func (s *FlowStore) recomputeWindowRatesLocked() {
	const windowSec = 10.0
	wb, wp := s.winBytes, s.winPkts
	if wb < 0 {
		wb = 0
	}
	if wp < 0 {
		wp = 0
	}
	s.stats.BytesPerSec = float64(wb) / windowSec
	s.stats.PacketsPerSec = float64(wp) / windowSec
	s.stats.Mbps = s.stats.BytesPerSec * 8 / 1e6
	s.stats.ByCategoryMbps = make(map[string]float64, len(s.winCatBytes))
	for k, v := range s.winCatBytes {
		if v > 0 {
			s.stats.ByCategoryMbps[k] = float64(v) * 8 / windowSec / 1e6
		}
	}
	s.stats.ByCategoryInMbps = make(map[string]float64, len(s.winCatIn))
	for k, v := range s.winCatIn {
		if v > 0 {
			s.stats.ByCategoryInMbps[k] = float64(v) * 8 / windowSec / 1e6
		}
	}
	s.stats.ByCategoryOutMbps = make(map[string]float64, len(s.winCatOut))
	for k, v := range s.winCatOut {
		if v > 0 {
			s.stats.ByCategoryOutMbps[k] = float64(v) * 8 / windowSec / 1e6
		}
	}
	s.stats.ByIfaceRole = make(map[string]float64, len(s.winRoleBytes))
	for k, v := range s.winRoleBytes {
		if v > 0 {
			s.stats.ByIfaceRole[k] = float64(v) * 8 / windowSec / 1e6
		}
	}
}

func (s *FlowStore) AddFlow(f FlowRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.pushFlowLocked(f)

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
		if f.Category == CategoryStreaming || f.Category == CategoryNetflix || f.Category == CategoryGlobo || f.Category == CategoryApple {
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
		if f.Direction == "outbound" {
			dstAS = parseASNNum(f.ASN)
		}
	}
	if !isVerifiedASN(dstAS) {
		dstAS = 0
	}
	sm := rateSample{
		now, f.Bytes, f.Packets, catKey, svc, peerAS, dstAS, f.Direction, f.IfaceRole, f.IPVersion,
	}
	s.window = append(s.window, sm)
	s.applyWindowSampleLocked(sm, 1)

	if peerAS > 0 {
		s.peerASNBytes[peerAS] += f.Bytes
		s.peerASNFlows[peerAS]++
	}
	if dstAS > 0 {
		s.dstASNBytes[dstAS] += f.Bytes
		s.dstASNFlows[dstAS]++
		s.pushASNRecentLocked(dstAS, f)
	}

	cutoff := now.Add(-10 * time.Second)
	i := 0
	for i < len(s.window) && s.window[i].at.Before(cutoff) {
		s.applyWindowSampleLocked(s.window[i], -1)
		i++
	}
	if i > 0 {
		s.window = s.window[i:]
	}

	s.recomputeWindowRatesLocked()
	s.rebuildConsumptionLocked(s.winSvcBytes)
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

func pruneIPKeys(m map[string]int64) {
	for k := range m {
		if looksLikeIP(k) {
			delete(m, k)
		}
	}
}

func (s *FlowStore) refreshDerivedLocked() {
	pruneIPKeys(s.stats.ByDestination)
	pruneIPKeys(s.stats.ByOrigin)
	s.pruneASNMapsLocked(400)
	s.rebuildConsumptionLocked(s.winSvcBytes)
	s.rebuildTopCards(s.winSvcBytes)
}

func (s *FlowStore) pruneASNMapsLocked(maxKeep int) {
	if maxKeep < 50 {
		maxKeep = 50
	}
	pruneOne := func(bytes map[uint32]int64, flows map[uint32]int64) {
		if len(bytes) <= maxKeep {
			return
		}
		type kv struct {
			as uint32
			b  int64
		}
		arr := make([]kv, 0, len(bytes))
		for as, b := range bytes {
			arr = append(arr, kv{as, b})
		}
		sort.Slice(arr, func(i, j int) bool { return arr[i].b > arr[j].b })
		keep := make(map[uint32]bool, maxKeep)
		for i := 0; i < maxKeep && i < len(arr); i++ {
			keep[arr[i].as] = true
		}
		for as := range bytes {
			if !keep[as] {
				delete(bytes, as)
				delete(flows, as)
			}
		}
	}
	pruneOne(s.dstASNBytes, s.dstASNFlows)
	pruneOne(s.peerASNBytes, s.peerASNFlows)
}

func (s *FlowStore) startDerivedRefresh() {
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		for range ticker.C {
			s.mu.Lock()
			s.refreshDerivedLocked()
			s.mu.Unlock()
		}
	}()
}

func (s *FlowStore) GetStats() AggregatedStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	svcBytes := map[string]int64{}
	svcIn := map[string]int64{}
	svcOut := map[string]int64{}
	svcV4 := map[string]int64{}
	svcV6 := map[string]int64{}
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
			if sm.ipVersion == "6" {
				svcV6[sm.service] += sm.bytes
			} else {
				svcV4[sm.service] += sm.bytes
			}
		}
	}
	// rebuild feito em startDerivedRefresh(); aqui só leitura

	out := s.stats
	out.ByCategory = copyMap(s.stats.ByCategory)
	out.ByCategoryMbps = copyFloatMap(s.stats.ByCategoryMbps)
	out.ByCategoryInMbps = copyFloatMap(s.stats.ByCategoryInMbps)
	out.ByCategoryOutMbps = copyFloatMap(s.stats.ByCategoryOutMbps)
	out.ByIfaceRole = copyFloatMap(s.stats.ByIfaceRole)
	// Não serializar maps completos (podem ter centenas de milhares de IPs) —
	// o dashboard só precisa das contagens + top_*.
	out.DestinationCount = len(s.stats.ByDestination)
	out.OriginCount = len(s.stats.ByOrigin)
	out.ByDestination = nil
	out.ByOrigin = nil
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
	// Destino e peer ficam separados (sem merge).
	asnBytes := copyMapUint(s.dstASNBytes)
	asnFlows := copyMapUint(s.dstASNFlows)
	peerBytes := copyMapUint(s.peerASNBytes)
	peerFlows := copyMapUint(s.peerASNFlows)

	peerWinOnly := map[uint32]int64{}
	peerIn := map[uint32]int64{}
	peerOut := map[uint32]int64{}
	peerV4 := map[uint32]int64{}
	peerV6 := map[uint32]int64{}
	for _, sm := range s.window {
		if sm.peerASN == 0 {
			continue
		}
		peerWinOnly[sm.peerASN] += sm.bytes
		if sm.direction == "inbound" {
			peerIn[sm.peerASN] += sm.bytes
		} else {
			peerOut[sm.peerASN] += sm.bytes
		}
		if sm.ipVersion == "6" {
			peerV6[sm.peerASN] += sm.bytes
		} else {
			peerV4[sm.peerASN] += sm.bytes
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
	out.StreamingRates = buildServiceRatesList(out.StreamingBreak, svcBytes, svcIn, svcOut, svcV4, svcV6, lookupCat, eff,
		[]string{"streaming", "netflix", "globo", "apple"})
	out.CDNRates = buildServiceRatesList(out.CDNBreakdown, svcBytes, svcIn, svcOut, svcV4, svcV6, lookupCat, eff,
		[]string{"cdn"})
	out.ASNBreakdown = buildASNBreakdown(asnBytes, asnFlows, dstWin, dstIn, dstOut, dstV4, dstV6, s.stats.TotalBytes, eff, "destination")
	out.PeerASNBreakdown = buildASNBreakdown(peerBytes, peerFlows, peerWinOnly, peerIn, peerOut, peerV4, peerV6, s.stats.TotalBytes, eff, "peer")
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
	svcBytes, svcIn, svcOut, svcV4, svcV6 map[string]int64,
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
	var totalBytes int64
	for k, v := range breakdown {
		if v > 0 {
			names[k] = struct{}{}
			totalBytes += v
		}
	}
	for name := range svcBytes {
		if catSet[lookup(name)] {
			names[name] = struct{}{}
		}
	}
	if totalBytes <= 0 {
		totalBytes = 1
	}
	out := make([]ServiceRate, 0, len(names))
	for name := range names {
		mbps := float64(svcBytes[name]) * 8 / windowSec / 1e6
		inM := float64(svcIn[name]) * 8 / windowSec / 1e6
		outM := float64(svcOut[name]) * 8 / windowSec / 1e6
		v4M := float64(svcV4[name]) * 8 / windowSec / 1e6
		v6M := float64(svcV6[name]) * 8 / windowSec / 1e6
		out = append(out, ServiceRate{
			Name:       name,
			Bytes:      breakdown[name],
			Mbps:       mbps,
			MbpsScaled: mbps * eff,
			InMbps:     inM * eff,
			OutMbps:    outM * eff,
			IPv4Mbps:   v4M * eff,
			IPv6Mbps:   v6M * eff,
			Percentage: float64(breakdown[name]) / float64(totalBytes) * 100,
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
	role string,
) []ASNTraffic {
	const windowSec = 10.0
	if total <= 0 {
		total = 1
	}
	if eff < 1 {
		eff = 1
	}
	if role == "" {
		role = "destination"
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
		verified := isVerifiedASN(as)
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
		if !verified {
			name = fmt.Sprintf("AS%d (pendente)", as)
		}
		cat := "other"
		icon := "peer"
		if role == "peer" {
			cat = "peer"
		}
		if c := classifyByASN(as); c.Name != "" {
			cat = string(c.Category)
			icon = c.Icon
			if verified && strings.HasPrefix(name, "AS") {
				name = c.Name
			}
		} else if p, ok := bgpStore.LookupAS(as); ok && verified {
			name = p.Name
			if role == "peer" {
				cat = "peer"
			}
		} else if n := ipapi.NameForASN(as); n != "" && verified {
			name = n
		}
		out = append(out, ASNTraffic{
			ASN:        fmt.Sprintf("AS%d", as),
			Name:       name,
			Role:       role,
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
			Pending:    !verified,
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

func (s *FlowStore) pruneASNRecentLocked() {
	type kv struct {
		as uint32
		n  int
	}
	ranked := make([]kv, 0, len(s.asnRecentFlows))
	for as, pos := range s.asnRecentPos {
		n := pos
		if n > 80 {
			n = 80
		}
		ranked = append(ranked, kv{as, n})
	}
	sort.Slice(ranked, func(i, j int) bool { return ranked[i].n < ranked[j].n })
	for len(s.asnRecentFlows) > 100 && len(ranked) > 0 {
		as := ranked[0].as
		delete(s.asnRecentFlows, as)
		delete(s.asnRecentPos, as)
		ranked = ranked[1:]
	}
}

func (s *FlowStore) GetASNRecentFlows(as uint32, limit int) []FlowRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 || limit > 200 {
		limit = 80
	}
	ring := s.asnRecentFlows[as]
	if ring == nil {
		return nil
	}
	pos := s.asnRecentPos[as]
	n := pos
	if n > len(ring) {
		n = len(ring)
	}
	if n > limit {
		n = limit
	}
	out := make([]FlowRecord, 0, n)
	// newest first
	for i := 0; i < n; i++ {
		idx := (pos - 1 - i) % len(ring)
		if idx < 0 {
			idx += len(ring)
		}
		f := ring[idx]
		if f.ID == "" {
			continue
		}
		out = append(out, f)
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

var (
	statsCacheMu sync.RWMutex
	statsCache   AggregatedStats
	statsCacheAt time.Time
)

// GetStatsCached — snapshot completo refresca no máx. a cada ~1.5s.
func (s *FlowStore) GetStatsCached() AggregatedStats {
	statsCacheMu.RLock()
	if !statsCacheAt.IsZero() && time.Since(statsCacheAt) < 1500*time.Millisecond {
		out := statsCache
		statsCacheMu.RUnlock()
		return out
	}
	statsCacheMu.RUnlock()

	full := s.GetStats()
	statsCacheMu.Lock()
	statsCache = full
	statsCacheAt = time.Now()
	statsCacheMu.Unlock()
	return full
}

func startStatsCacheRefresh() {
	go func() {
		time.Sleep(2 * time.Second)
		ticker := time.NewTicker(1500 * time.Millisecond)
		for range ticker.C {
			full := store.GetStats()
			statsCacheMu.Lock()
			statsCache = full
			statsCacheAt = time.Now()
			statsCacheMu.Unlock()
		}
	}()
}

func buildPeerBreakdown(peers []BGPPeer, total int64) []DestOriginCard {
	if total <= 0 {
		total = 1
	}
	eff := sampling.Get().Effective
	if eff < 1 {
		eff = 1
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
		// Mbps already shared per ASN in annotate — take max to avoid multi-session multiply
		if p.Mbps > a.mbps {
			a.mbps = p.Mbps
		}
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
		out = append(out, DestOriginCard{
			Name:       a.name + " (" + a.asn + ")",
			Bytes:      a.bytes,
			Mbps:       a.mbps,
			MbpsScaled: a.mbps * eff,
			Percentage: float64(a.bytes) / float64(total) * 100,
			Category:   cat,
			Icon:       "peer",
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].MbpsScaled != out[j].MbpsScaled {
			return out[i].MbpsScaled > out[j].MbpsScaled
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
	peerWin := map[uint32]int64{}
	var v4b, v6b int64
	for _, sm := range s.window {
		if sm.dstASN > 0 {
			asnWin[sm.dstASN] += sm.bytes
		}
		if sm.peerASN > 0 {
			peerWin[sm.peerASN] += sm.bytes
		}
		if sm.ipVersion == "6" {
			v6b += sm.bytes
		} else {
			v4b += sm.bytes
		}
	}
	const ws = 10.0
	c := GetConfig()
	topN := c.ASNHistoryTop
	if topN <= 0 {
		topN = 30
	}
	watched := map[uint32]bool{}
	for _, w := range c.ASNWatched {
		if as := parseASNNum(w); as > 0 {
			watched[as] = true
		}
	}

	rankASNMap := func(win map[uint32]int64) (map[string]float64, map[string]float64) {
		type asnKV struct {
			as uint32
			v  float64
		}
		ranked := make([]asnKV, 0, len(win))
		for as, b := range win {
			ranked = append(ranked, asnKV{as, float64(b) * 8 / ws / 1e6})
		}
		sort.Slice(ranked, func(i, j int) bool { return ranked[i].v > ranked[j].v })
		keep := map[uint32]float64{}
		for i, kv := range ranked {
			if i < topN || watched[kv.as] {
				keep[kv.as] = kv.v
			}
		}
		for as := range watched {
			if _, ok := keep[as]; !ok {
				if b, ok2 := win[as]; ok2 {
					keep[as] = float64(b) * 8 / ws / 1e6
				}
			}
		}
		by := make(map[string]float64, len(keep))
		byS := make(map[string]float64, len(keep))
		for as, v := range keep {
			key := fmt.Sprintf("AS%d", as)
			by[key] = v
			byS[key] = v * eff
		}
		return by, byS
	}

	byASN, byASNScaled := rankASNMap(asnWin)
	byPeer, byPeerScaled := rankASNMap(peerWin)

	// Top serviços de streaming na janela
	streamWin := map[string]int64{}
	streamCats := map[string]bool{"streaming": true, "netflix": true, "globo": true, "apple": true}
	for _, sm := range s.window {
		if sm.service == "" || !streamCats[sm.category] {
			continue
		}
		streamWin[sm.service] += sm.bytes
	}
	type skv struct {
		k string
		v float64
	}
	sranked := make([]skv, 0, len(streamWin))
	for k, b := range streamWin {
		sranked = append(sranked, skv{k, float64(b) * 8 / ws / 1e6})
	}
	sort.Slice(sranked, func(i, j int) bool { return sranked[i].v > sranked[j].v })
	if len(sranked) > 12 {
		sranked = sranked[:12]
	}
	byStream := make(map[string]float64, len(sranked))
	byStreamScaled := make(map[string]float64, len(sranked))
	for _, kv := range sranked {
		byStream[kv.k] = kv.v
		byStreamScaled[kv.k] = kv.v * eff
	}

	// Top CDNs na janela (+ watchlist)
	cdnWin := map[string]int64{}
	for _, sm := range s.window {
		if sm.service == "" || sm.category != "cdn" {
			continue
		}
		cdnWin[sm.service] += sm.bytes
	}
	cranked := make([]skv, 0, len(cdnWin))
	for k, b := range cdnWin {
		cranked = append(cranked, skv{k, float64(b) * 8 / ws / 1e6})
	}
	sort.Slice(cranked, func(i, j int) bool { return cranked[i].v > cranked[j].v })
	cdnTopN := 12
	watchedCDN := map[string]bool{}
	for _, w := range c.CDNWatched {
		watchedCDN[strings.TrimSpace(w)] = true
	}
	byCDN := make(map[string]float64)
	byCDNScaled := make(map[string]float64)
	for i, kv := range cranked {
		if i < cdnTopN || watchedCDN[kv.k] {
			byCDN[kv.k] = kv.v
			byCDNScaled[kv.k] = kv.v * eff
		}
	}
	for name := range watchedCDN {
		if _, ok := byCDN[name]; ok {
			continue
		}
		if b, ok := cdnWin[name]; ok {
			v := float64(b) * 8 / ws / 1e6
			byCDN[name] = v
			byCDNScaled[name] = v * eff
		}
	}

	byRole := map[string]float64{}
	for _, iface := range snmp.Interfaces {
		if iface.OperStatus != 1 || iface.Role == "" {
			continue
		}
		byRole[iface.Role] += iface.InMbps + iface.OutMbps
	}
	v4Mbps := float64(v4b) * 8 / ws / 1e6 * eff
	v6Mbps := float64(v6b) * 8 / ws / 1e6 * eff

	pt := HistoryPoint{
		Ts:                time.Now().Unix(),
		Mbps:              s.stats.Mbps,
		MbpsScaled:        s.stats.Mbps * eff,
		ByCategory:        cats,
		ByCategoryScaled:  scaledCats,
		ByASN:             byASN,
		ByASNScaled:       byASNScaled,
		ByPeerASN:         byPeer,
		ByPeerASNScaled:   byPeerScaled,
		ByStreaming:       byStream,
		ByStreamingScaled: byStreamScaled,
		ByCDN:             byCDN,
		ByCDNScaled:       byCDNScaled,
		BySNMPRole:        byRole,
		IPv4Mbps:          v4Mbps,
		IPv6Mbps:          v6Mbps,
		SNMPIn:            snmp.UplinkInMbps,
		SNMPOut:           snmp.UplinkOutMbps,
		SamplingFactor:    eff,
	}
	s.history = append(s.history, pt)
	if len(s.history) > 120 {
		s.history = s.history[len(s.history)-120:]
	}
	go storageEnqueue(pt, v4Mbps, v6Mbps)
	go persistASNDaily()
	go snmpStore.pushIfaceHistory(snmp)
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
	if limit <= 0 {
		limit = 50
	}
	ip = strings.ToLower(strings.TrimSpace(ip))
	category = strings.ToLower(strings.TrimSpace(category))
	q = strings.ToLower(strings.TrimSpace(q))
	asn = normalizeASNQuery(asn)
	var result []FlowRecord
	n := s.flowsCount
	for i := 0; i < n; i++ {
		idx := s.flowsPos - 1 - i
		for idx < 0 {
			idx += MaxHistory
		}
		f := s.flowsBuf[idx%MaxHistory]
		if f.ID == "" {
			continue
		}
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

func classifyRaw(raw RawFlow) (FlowRecord, bool) {
	src := raw.SrcIP
	dst := raw.DstIP
	if src == nil || dst == nil {
		return FlowRecord{}, false
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

	return FlowRecord{
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
	}, true
}

func ingestRaw(raw RawFlow) {
	if f, ok := classifyRaw(raw); ok {
		store.AddFlow(f)
	}
}

func lookupIfaceMeta(inIf, outIf uint32, direction string) (name, role string) {
	prefer := outIf
	if direction == "inbound" {
		prefer = inIf
	}
	if n, r, ok := snmpStore.LookupIface(prefer); ok {
		return n, r
	}
	if n, r, ok := snmpStore.LookupIface(inIf); ok {
		return n, r
	}
	if n, r, ok := snmpStore.LookupIface(outIf); ok {
		return n, r
	}
	return "", ""
}

func lookupCacheIface(inIf, outIf uint32) string {
	return snmpStore.LookupCacheIface(inIf, outIf)
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
		incHTTPReq()
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
	if from := r.URL.Query().Get("from"); from != "" {
		if n, err := strconv.ParseInt(from, 10, 64); err == nil && n > 0 {
			since = n
			hasSince = true
		}
	}
	until := int64(0)
	if to := r.URL.Query().Get("to"); to != "" {
		if n, err := strconv.ParseInt(to, 10, 64); err == nil && n > 0 {
			until = n
		}
	}
	maxPoints := 0
	if v := r.URL.Query().Get("max_points"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxPoints = n
			if maxPoints > 5000 {
				maxPoints = 5000
			}
		}
	}
	// Ao vivo (sem filtro): só memória (~10 min) — evita carregar 30d do SQLite/S3.
	var pts []HistoryPoint
	if !hasSince && !hasHours {
		pts = store.GetHistory()
	} else {
		pts = queryHistorySince(since)
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
	}
	if until > 0 {
		filtered := pts[:0]
		for _, pt := range pts {
			if pt.Ts <= until {
				filtered = append(filtered, pt)
			}
		}
		pts = filtered
	}
	if maxPoints > 0 {
		pts = downsampleHistoryPoints(pts, maxPoints)
	}
	writeJSON(w, pts)
}

func downsampleHistoryPoints(pts []HistoryPoint, max int) []HistoryPoint {
	if max <= 0 || len(pts) <= max {
		return pts
	}
	out := make([]HistoryPoint, 0, max)
	n := len(pts)
	for i := 0; i < max; i++ {
		idx := i * (n - 1) / (max - 1)
		out = append(out, pts[idx])
	}
	if out[len(out)-1].Ts != pts[n-1].Ts {
		out[len(out)-1] = pts[n-1]
	}
	return out
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
			fmt.Fprintln(w, "name,asn,category,bytes,mbps,mbps_scaled,in_mbps,out_mbps,ipv4_mbps,ipv6_mbps,percentage")
			for _, r := range enrichCDNRates(stats.CDNRates) {
				fmt.Fprintf(w, "%s,%s,%s,%d,%.4f,%.4f,%.4f,%.4f,%.4f,%.4f,%.2f\n",
					csvEscape(r.Name), r.ASN, r.Category, r.Bytes, r.Mbps, r.MbpsScaled,
					r.InMbps, r.OutMbps, r.IPv4Mbps, r.IPv6Mbps, r.Percentage)
			}
		case "router":
			fmt.Fprintln(w, "index,name,alias,role,oper_status,speed_mbps,in_mbps,out_mbps,in_util_pct,out_util_pct")
			snmp := snmpStore.Get()
			for _, i := range snmp.Interfaces {
				fmt.Fprintf(w, "%d,%s,%s,%s,%d,%d,%.4f,%.4f,%.2f,%.2f\n",
					i.Index, csvEscape(i.Name), csvEscape(i.Alias), i.Role, i.OperStatus, i.SpeedMbps,
					i.InMbps, i.OutMbps, i.InUtilPct, i.OutUtilPct)
			}
		case "history":
			fmt.Fprintln(w, "ts,mbps,mbps_scaled,snmp_in,snmp_out,ipv4_mbps,ipv6_mbps,sampling_factor")
			hours := 6
			if v := r.URL.Query().Get("hours"); v != "" {
				if h, err := strconv.Atoi(v); err == nil && h > 0 {
					hours = h
				}
			}
			since := time.Now().Unix() - int64(hours)*3600
			pts := queryHistorySince(since)
			if len(pts) == 0 {
				pts = store.GetHistory()
			}
			pts = downsampleHistoryPoints(pts, 2000)
			for _, p := range pts {
				fmt.Fprintf(w, "%d,%.4f,%.4f,%.4f,%.4f,%.4f,%.4f,%.2f\n",
					p.Ts, p.Mbps, p.MbpsScaled, p.SNMPIn, p.SNMPOut, p.IPv4Mbps, p.IPv6Mbps, p.SamplingFactor)
			}
		case "streaming":
			fmt.Fprintln(w, "name,category,bytes,mbps,mbps_scaled,in_mbps,out_mbps,ipv4_mbps,ipv6_mbps,percentage")
			for _, r := range stats.StreamingRates {
				fmt.Fprintf(w, "%s,%s,%d,%.4f,%.4f,%.4f,%.4f,%.4f,%.4f,%.2f\n",
					csvEscape(r.Name), r.Category, r.Bytes, r.Mbps, r.MbpsScaled,
					r.InMbps, r.OutMbps, r.IPv4Mbps, r.IPv6Mbps, r.Percentage)
			}
		case "peers":
			fmt.Fprintln(w, "remote_addr,asn,name,state,role,established,mbps,mbps_scaled,bytes,flows,in_updates,out_updates,uptime_sec,flap_count,last_flap_at")
			bgp := bgpStore.Get()
			store.mu.RLock()
			peerWin := map[uint32]int64{}
			for _, sm := range store.window {
				if sm.peerASN > 0 {
					peerWin[sm.peerASN] += sm.bytes
				}
			}
			peers := annotatePeersWithTraffic(bgp.Peers, store.peerASNBytes, peerWin, store.peerASNFlows)
			store.mu.RUnlock()
			for _, p := range peers {
				est := 0
				if p.Established {
					est = 1
				}
				fmt.Fprintf(w, "%s,%s,%s,%s,%s,%d,%.4f,%.4f,%d,%d,%d,%d,%d,%d,%d\n",
					p.RemoteAddr, p.ASN, csvEscape(p.Name), p.StateName, p.Role, est,
					p.Mbps, p.MbpsScaled, p.Bytes, p.Flows, p.InUpdates, p.OutUpdates,
					p.UptimeSec, p.FlapCount, p.LastFlapAt)
			}
		case "asn":
			fmt.Fprintln(w, "asn,name,role,bytes,flows,mbps,mbps_scaled,in_mbps,out_mbps,ipv4_mbps,ipv6_mbps,percentage,pending")
			all := append([]ASNTraffic{}, stats.ASNBreakdown...)
			all = append(all, stats.PeerASNBreakdown...)
			for _, a := range all {
				pending := 0
				if a.Pending {
					pending = 1
				}
				fmt.Fprintf(w, "%s,%s,%s,%d,%d,%.4f,%.4f,%.4f,%.4f,%.4f,%.4f,%.2f,%d\n",
					a.ASN, csvEscape(a.Name), a.Role, a.Bytes, a.Flows, a.Mbps, a.MbpsScaled,
					a.InMbps, a.OutMbps, a.IPv4Mbps, a.IPv6Mbps, a.Percentage, pending)
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
		case "dashboard":
			fallthrough
		case "stats":
			fmt.Fprintln(w, "section,key,value")
			fmt.Fprintf(w, "kpi,mbps,%.4f\n", stats.Mbps)
			fmt.Fprintf(w, "kpi,mbps_scaled,%.4f\n", stats.MbpsScaled)
			fmt.Fprintf(w, "kpi,ipv4_mbps,%.4f\n", stats.IPv4Mbps)
			fmt.Fprintf(w, "kpi,ipv6_mbps,%.4f\n", stats.IPv6Mbps)
			if stats.SNMP != nil {
				fmt.Fprintf(w, "kpi,snmp_in,%.4f\n", stats.SNMP.UplinkInMbps)
				fmt.Fprintf(w, "kpi,snmp_out,%.4f\n", stats.SNMP.UplinkOutMbps)
			}
			fmt.Fprintln(w, "")
			fmt.Fprintln(w, "category,bytes,mbps,mbps_scaled,percentage")
			for _, c := range stats.Consumption {
				fmt.Fprintf(w, "%s,%d,%.4f,%.4f,%.2f\n", c.Category, c.Bytes, c.Mbps, c.MbpsScaled, c.Percentage)
			}
			fmt.Fprintln(w, "")
			fmt.Fprintln(w, "talker_ip,bytes,mbps_scaled,top_category,flows")
			for _, t := range talkers.Top(30) {
				fmt.Fprintf(w, "%s,%d,%.4f,%s,%d\n", t.IP, t.Bytes, t.MbpsScaled, t.TopCat, t.Flows)
			}
			if stats.BGP != nil {
				fmt.Fprintln(w, "")
				fmt.Fprintln(w, "bgp_asn,name,state,role,mbps_scaled")
				for _, p := range stats.BGP.Peers {
					fmt.Fprintf(w, "%s,%s,%s,%s,%.4f\n", p.ASN, csvEscape(p.Name), p.StateName, p.Role, p.MbpsScaled)
				}
			}
		}
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename=inforflow-"+kind+".json")
	switch kind {
	case "cdn":
		writeJSON(w, stats.CDNBreakdown)
	case "streaming":
		writeJSON(w, stats.StreamingRates)
	case "peers":
		writeJSON(w, stats.PeerBreakdown)
	case "asn":
		writeJSON(w, map[string]interface{}{
			"destinations": stats.ASNBreakdown,
			"peers":        stats.PeerASNBreakdown,
		})
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
	silent := netFlowSilentSec()
	pts, dbBytes := storageLocalStats()
	snmpAvg := (snmp.UplinkInMbps + snmp.UplinkOutMbps) / 2
	nf := NetFlowStats()
	writeJSON(w, map[string]interface{}{
		"status":           "ok",
		"source_ip":        SourceIP,
		"service":          "inforflow-collector",
		"mode":             "netflow+snmp+bgp",
		"listen":           NetFlowPort,
		"snmp":             snmp.OK,
		"bgp":              bgp.OK,
		"bgp_peers":        bgp.Established,
		"bgp_total":        bgp.Total,
		"sys_name":         snmp.SysName,
		"uptime_s":         int(time.Since(store.started).Seconds()),
		"flows":            atomic.LoadUint64(&store.seq),
		"sampling":         samp.Effective,
		"sampling_mode":    samp.Mode,
		"mbps_scaled":      samp.ScaledMbps,
		"snmp_mbps":        snmpAvg,
		"gap_pct":          gapPct(samp.ScaledMbps, snmpAvg),
		"netflow_silent_s": silent,
		"netflow":          nf,
		"udp_queue":        nf["udp_queue"],
		"udp_rcvbuf":       nf["udp_rcvbuf"],
		"udp_kernel_drops": nf["kernel_drops"],
		"ingest_queue":     ingestQueueLen(),
		"ingest_workers":   ingestWorkerCount(),
		"alerts":           len(alerts.Active()),
		"auth_enabled":     GetConfig().APIToken != "" || uiAuthEnabled(),
		"ipapi":            ipapi.Stats(),
		"s3":               S3Status(),
		"storage": map[string]interface{}{
			"history_points": pts,
			"db_bytes":       dbBytes,
		},
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
	startStorageWriter()
	loadNativeSampling()
	loadASNDaily()
	loadPeersConfig()
	startSessionJanitor()
	store.startDerivedRefresh()
	startStatsCacheRefresh()
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
	startIngestPipeline()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", handleHealth)
	mux.HandleFunc("/metrics", handleMetrics)
	mux.HandleFunc("/api/stats", requireAuth(handleStats))
	mux.HandleFunc("/api/dashboard", requireAuth(handleDashboardPage))
	mux.HandleFunc("/api/flows", requireAuth(handleFlows))
	mux.HandleFunc("/api/snmp", requireAuth(handleSNMP))
	mux.HandleFunc("/api/router", requireAuth(handleRouterPage))
	mux.HandleFunc("/api/router/detail", requireAuth(handleRouterDetail))
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
	mux.HandleFunc("/api/asn", requireAuth(handleASN))
	mux.HandleFunc("/api/asn/daily", requireAuth(handleASNDaily))
	mux.HandleFunc("/api/asn/detail", requireAuth(handleASNDetail))
	mux.HandleFunc("/api/cdn", requireAuth(handleCDNPage))
	mux.HandleFunc("/api/cdn/detail", requireAuth(handleCDNDetail))
	mux.HandleFunc("/api/streaming", requireAuth(handleStreamingPage))
	mux.HandleFunc("/api/streaming/detail", requireAuth(handleStreamingDetail))
	mux.HandleFunc("/api/peers", requireAuth(handlePeersPage))
	mux.HandleFunc("/api/peers/detail", requireAuth(handlePeersDetail))
	mux.HandleFunc("/api/login", handleLogin)
	mux.HandleFunc("/api/logout", handleLogout)
	mux.HandleFunc("/api/auth/check", handleAuthCheck)

	StartASNDigest()
	log.Fatal(http.ListenAndServe(APIPort, mux))
}
