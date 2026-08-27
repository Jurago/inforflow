package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ASN canônico por CDN (para links /asn/detail).
var cdnCanonicalASN = map[string]string{
	"Cloudflare":      "AS13335",
	"Akamai":          "AS20940",
	"Fastly":          "AS54113",
	"AWS CloudFront":  "AS16509",
	"Google Cache":    "AS15169",
	"CDN77":           "AS60068",
	"BunnyCDN":        "AS262254",
	"Edgecast":        "AS15133",
	"Limelight":       "AS22822",
	"Imperva":         "AS19551",
	"Cachefly":        "AS30081",
	"G-Core":          "AS199524",
	"QUIC.cloud":      "AS13335",
	"Azure CDN":       "AS8075",
}

type CDNRate struct {
	ServiceRate
	ASN string `json:"asn,omitempty"`
}

type FeedSourceStatus struct {
	Name       string `json:"name"`
	File       string `json:"file,omitempty"`
	Prefixes   int    `json:"prefixes"`
	AgeSec     int64  `json:"age_sec,omitempty"`
	LastOK     bool   `json:"last_ok"`
	FromCache  bool   `json:"from_cache,omitempty"`
	UpdatedAt  int64  `json:"updated_at,omitempty"`
}

type FeedsStatus struct {
	TotalRules int                `json:"total_rules"`
	UpdatedAt  int64              `json:"updated_at,omitempty"`
	Sources    []FeedSourceStatus `json:"sources,omitempty"`
}

type CDNPageSnapshot struct {
	CDNBreakdown         map[string]int64   `json:"cdn_breakdown"`
	CDNRates             []CDNRate          `json:"cdn_rates"`
	ByCategoryMbps       map[string]float64 `json:"by_category_mbps"`
	ByCategoryMbpsScaled map[string]float64 `json:"by_category_mbps_scaled"`
	Sampling             *SamplingEstimate  `json:"sampling,omitempty"`
	SNMP                 *SNMPSnapshot      `json:"snmp,omitempty"`
	CacheIfaces          []SNMPInterface    `json:"cache_ifaces,omitempty"`
	CacheSNMPInMbps      float64            `json:"cache_snmp_in_mbps"`
	CacheSNMPOutMbps     float64            `json:"cache_snmp_out_mbps"`
	CacheHitPct          float64            `json:"cache_hit_pct"`
	TotalMbpsScaled      float64            `json:"total_mbps_scaled"`
	IPv4Mbps             float64            `json:"ipv4_mbps"`
	IPv6Mbps             float64            `json:"ipv6_mbps"`
	UplinkSharePct       float64            `json:"uplink_share_pct"`
	DivergenceWarn       string             `json:"divergence_warn,omitempty"`
	Flows                []FlowRecord       `json:"flows"`
	Exporter             string             `json:"exporter"`
	WindowHint           string             `json:"window_hint"`
	BytesHint            string             `json:"bytes_hint"`
	OverlapNote          string             `json:"overlap_note,omitempty"`
	Feeds                *FeedsStatus       `json:"feeds,omitempty"`
	CompareHint          string             `json:"compare_hint,omitempty"`
}

type CDNHistoryPoint struct {
	Ts         int64   `json:"ts"`
	MbpsScaled float64 `json:"mbps_scaled"`
}

type CDNDetailSnapshot struct {
	Name           string            `json:"name"`
	ASN            string            `json:"asn,omitempty"`
	Live           *CDNRate          `json:"live,omitempty"`
	Flows          []FlowRecord      `json:"flows"`
	History        []CDNHistoryPoint `json:"history"`
	CacheIfaces    []SNMPInterface   `json:"cache_ifaces,omitempty"`
	SNMPMatchMbps  float64           `json:"snmp_match_mbps"`
	DivergenceWarn string            `json:"divergence_warn,omitempty"`
	Sampling       *SamplingEstimate `json:"sampling,omitempty"`
	SNMP           *SNMPSnapshot     `json:"snmp,omitempty"`
	OverlapNote    string            `json:"overlap_note,omitempty"`
}

func cdnASNFor(name string) string {
	if a, ok := cdnCanonicalASN[name]; ok {
		return a
	}
	for k, a := range cdnCanonicalASN {
		if strings.EqualFold(k, name) {
			return a
		}
	}
	return ""
}

func enrichCDNRates(rates []ServiceRate) []CDNRate {
	out := make([]CDNRate, 0, len(rates))
	for _, r := range rates {
		out = append(out, CDNRate{ServiceRate: r, ASN: cdnASNFor(r.Name)})
	}
	return out
}

func isCDNCacheIface(iface SNMPInterface) bool {
	t := strings.ToUpper(iface.Alias + " " + iface.Name)
	return strings.Contains(t, "CACHE") || strings.Contains(t, "CDN") ||
		strings.Contains(t, "GGC") || strings.Contains(t, "GOOGLE") ||
		strings.Contains(t, "CLOUDFLARE") || strings.Contains(t, "AKAMAI") ||
		strings.Contains(t, "FASTLY") || strings.Contains(t, "CLOUDFRONT") ||
		iface.Role == "cache" || iface.Role == "cdn"
}

func ifaceMatchesCDN(iface SNMPInterface, cdn string) bool {
	t := strings.ToUpper(iface.Alias + " " + iface.Name)
	svc := strings.ToUpper(cdn)
	keys := []string{}
	switch {
	case strings.Contains(svc, "CLOUDFLARE") || strings.Contains(svc, "QUIC"):
		keys = []string{"CLOUDFLARE", "CF "}
	case strings.Contains(svc, "AKAMAI"):
		keys = []string{"AKAMAI"}
	case strings.Contains(svc, "FASTLY"):
		keys = []string{"FASTLY"}
	case strings.Contains(svc, "CLOUDFRONT") || strings.Contains(svc, "AWS"):
		keys = []string{"CLOUDFRONT", "AWS", "AMAZON"}
	case strings.Contains(svc, "GOOGLE") || strings.Contains(svc, "GGC"):
		keys = []string{"GOOGLE", "GGC", "YOUTUBE"}
	case strings.Contains(svc, "AZURE"):
		keys = []string{"AZURE", "MICROSOFT"}
	case strings.Contains(svc, "BUNNY"):
		keys = []string{"BUNNY"}
	case strings.Contains(svc, "CDN77"):
		keys = []string{"CDN77", "77"}
	default:
		keys = []string{"CACHE", "CDN"}
	}
	for _, k := range keys {
		if strings.Contains(t, k) {
			return true
		}
	}
	return false
}

func handleCDNPage(w http.ResponseWriter, r *http.Request) {
	stats := store.GetStatsCached()
	cache := computeCacheHit()

	scaled := stats.ByCategoryMbpsScaled
	if scaled == nil {
		scaled = map[string]float64{}
	}
	total := scaled["cdn"]

	rates := enrichCDNRates(stats.CDNRates)
	var ipv4, ipv6 float64
	for _, r := range rates {
		ipv4 += r.IPv4Mbps
		ipv6 += r.IPv6Mbps
	}

	uplink := 0.0
	if stats.SNMP != nil && stats.SNMP.OK {
		uplink = (stats.SNMP.UplinkInMbps + stats.SNMP.UplinkOutMbps) / 2
	}
	share := 0.0
	if uplink > 1 {
		share = total / uplink * 100
	}

	warn := ""
	cacheTotal := cache.CacheSNMPInMbps + cache.CacheSNMPOutMbps
	if cache.OK && cacheTotal > 50 && total > 0 {
		ratio := cacheTotal / total
		if ratio < 0.15 || ratio > 3 {
			warn = fmt.Sprintf("Divergência cache SNMP (%.0f Mbps) vs CDN classificado (%.0f Mbps)", cacheTotal, total)
		}
	}

	flows := store.GetRecentFlowsFiltered(40, "", "cdn", "", "")
	if len(flows) > 25 {
		flows = flows[:25]
	}

	exporter := stats.Exporter
	if exporter == "" {
		exporter = SourceIP
	}

	overlap := ""
	ggc := 0.0
	for _, r := range rates {
		if strings.Contains(strings.ToLower(r.Name), "google") {
			ggc = r.MbpsScaled
			break
		}
	}
	yt := 0.0
	for _, r := range stats.StreamingRates {
		if strings.EqualFold(r.Name, "YouTube") {
			yt = r.MbpsScaled
			break
		}
	}
	if ggc > 0 || yt > 0 {
		overlap = fmt.Sprintf("Google Cache (CDN) e YouTube (Streaming) podem overlapar: GGC ~%.0f Mbps · YouTube ~%.0f Mbps — não some os dois.", ggc, yt)
	}

	writeJSON(w, CDNPageSnapshot{
		CDNBreakdown:         stats.CDNBreakdown,
		CDNRates:             rates,
		ByCategoryMbps:       stats.ByCategoryMbps,
		ByCategoryMbpsScaled: scaled,
		Sampling:             stats.Sampling,
		SNMP:                 stats.SNMP,
		CacheIfaces:          filterCDNIfaces(cache.Interfaces),
		CacheSNMPInMbps:      cache.CacheSNMPInMbps,
		CacheSNMPOutMbps:     cache.CacheSNMPOutMbps,
		CacheHitPct:          cache.EstimatedHitPct,
		TotalMbpsScaled:      total,
		IPv4Mbps:             ipv4,
		IPv6Mbps:             ipv6,
		UplinkSharePct:       share,
		DivergenceWarn:       warn,
		Flows:                flows,
		Exporter:             exporter,
		WindowHint:           "Mbps = janela ~10s × amostragem SNMP",
		BytesHint:            "Bytes / % = acumulado do dia",
		OverlapNote:          overlap,
		Feeds:                getFeedsStatus(),
	})
}

func filterCDNIfaces(ifaces []SNMPInterface) []SNMPInterface {
	out := make([]SNMPInterface, 0)
	for _, iface := range ifaces {
		if isCDNCacheIface(iface) {
			out = append(out, iface)
		}
	}
	return out
}

func handleCDNDetail(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		http.Error(w, `{"error":"name required"}`, http.StatusBadRequest)
		return
	}
	stats := store.GetStatsCached()
	rates := enrichCDNRates(stats.CDNRates)
	var live *CDNRate
	for i := range rates {
		if strings.EqualFold(rates[i].Name, name) {
			live = &rates[i]
			name = rates[i].Name
			break
		}
	}
	if live == nil {
		for k, b := range stats.CDNBreakdown {
			if strings.EqualFold(k, name) {
				cr := CDNRate{
					ServiceRate: ServiceRate{Name: k, Bytes: b, Category: "cdn"},
					ASN:         cdnASNFor(k),
				}
				live = &cr
				name = k
				break
			}
		}
	}

	flowsAll := store.GetRecentFlowsFiltered(80, "", "cdn", "", "")
	flows := make([]FlowRecord, 0, 40)
	for _, f := range flowsAll {
		hay := f.Destination + " " + f.Origin
		if strings.EqualFold(f.Destination, name) || strings.EqualFold(f.Origin, name) ||
			strings.Contains(strings.ToLower(hay), strings.ToLower(name)) {
			flows = append(flows, f)
			if len(flows) >= 40 {
				break
			}
		}
	}

	hours := 6
	if v := r.URL.Query().Get("hours"); v != "" {
		if n, err := parseIntSafe(v); err == nil && n > 0 && n <= 72 {
			hours = n
		}
	}
	since := time.Now().Unix() - int64(hours)*3600
	histPts := queryHistorySince(since)
	if len(histPts) == 0 {
		histPts = store.GetHistory()
	}
	hist := make([]CDNHistoryPoint, 0, len(histPts))
	for _, p := range histPts {
		if p.Ts < since {
			continue
		}
		v := 0.0
		if p.ByCDNScaled != nil {
			if x, ok := p.ByCDNScaled[name]; ok {
				v = x
			} else {
				for k, x := range p.ByCDNScaled {
					if strings.EqualFold(k, name) {
						v = x
						break
					}
				}
			}
		}
		hist = append(hist, CDNHistoryPoint{Ts: p.Ts, MbpsScaled: v})
	}

	snmp := snmpStore.Get()
	var matched []SNMPInterface
	var snmpMbps float64
	for _, iface := range snmp.Interfaces {
		if isCDNCacheIface(iface) && ifaceMatchesCDN(iface, name) {
			matched = append(matched, iface)
			if iface.OperStatus == 1 {
				snmpMbps += iface.InMbps + iface.OutMbps
			}
		}
	}
	warn := ""
	if live != nil && snmpMbps > 10 && live.MbpsScaled > 0 {
		ratio := snmpMbps / live.MbpsScaled
		if ratio < 0.3 || ratio > 4 {
			warn = fmt.Sprintf("SNMP iface ~%.0f Mbps vs NetFlow×sampling %.0f Mbps", snmpMbps, live.MbpsScaled)
		}
	}

	overlap := ""
	if strings.Contains(strings.ToLower(name), "google") {
		overlap = "Google Cache (CDN) pode overlapar com YouTube na página Streaming — categorias distintas."
	}

	asn := ""
	if live != nil {
		asn = live.ASN
	}
	if asn == "" {
		asn = cdnASNFor(name)
	}

	writeJSON(w, CDNDetailSnapshot{
		Name:           name,
		ASN:            asn,
		Live:           live,
		Flows:          flows,
		History:        hist,
		CacheIfaces:    matched,
		SNMPMatchMbps:  snmpMbps,
		DivergenceWarn: warn,
		Sampling:       stats.Sampling,
		SNMP:           &snmp,
		OverlapNote:    overlap,
	})
}
