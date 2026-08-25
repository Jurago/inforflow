package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

type StreamingPageSnapshot struct {
	StreamingBreakdown   map[string]int64      `json:"streaming_breakdown"`
	StreamingRates       []ServiceRate         `json:"streaming_rates"`
	RelatedSocial        []ServiceRate         `json:"related_social,omitempty"`
	ByCategoryMbps       map[string]float64    `json:"by_category_mbps"`
	ByCategoryMbpsScaled map[string]float64    `json:"by_category_mbps_scaled"`
	ByCategoryInMbps     map[string]float64    `json:"by_category_in_mbps,omitempty"`
	ByCategoryOutMbps    map[string]float64    `json:"by_category_out_mbps,omitempty"`
	Sampling             *SamplingEstimate     `json:"sampling,omitempty"`
	SNMP                 *SNMPSnapshot         `json:"snmp,omitempty"`
	CacheIfaces          []SNMPInterface       `json:"cache_ifaces,omitempty"`
	CacheSNMPInMbps      float64               `json:"cache_snmp_in_mbps"`
	CacheSNMPOutMbps     float64               `json:"cache_snmp_out_mbps"`
	CacheHitPct          float64               `json:"cache_hit_pct"`
	TotalMbpsScaled      float64               `json:"total_mbps_scaled"`
	IPv4Mbps             float64               `json:"ipv4_mbps"`
	IPv6Mbps             float64               `json:"ipv6_mbps"`
	UplinkSharePct       float64               `json:"uplink_share_pct"`
	DivergenceWarn       string                `json:"divergence_warn,omitempty"`
	Flows                []FlowRecord          `json:"flows"`
	Exporter             string                `json:"exporter"`
	WindowHint           string                `json:"window_hint"`
	BytesHint            string                `json:"bytes_hint"`
	IncludeNote          string                `json:"include_note"`
}

type StreamingHistoryPoint struct {
	Ts         int64   `json:"ts"`
	MbpsScaled float64 `json:"mbps_scaled"`
}

type StreamingDetailSnapshot struct {
	Name           string                  `json:"name"`
	Live           *ServiceRate            `json:"live,omitempty"`
	Flows          []FlowRecord            `json:"flows"`
	History        []StreamingHistoryPoint `json:"history"`
	CacheIfaces    []SNMPInterface         `json:"cache_ifaces,omitempty"`
	SNMPMatchMbps  float64                 `json:"snmp_match_mbps"`
	DivergenceWarn string                  `json:"divergence_warn,omitempty"`
	Sampling       *SamplingEstimate       `json:"sampling,omitempty"`
	SNMP           *SNMPSnapshot           `json:"snmp,omitempty"`
}

func isStreamingCacheIface(iface SNMPInterface) bool {
	t := strings.ToUpper(iface.Alias + " " + iface.Name)
	return strings.Contains(t, "CACHE") || strings.Contains(t, "NETFLIX") ||
		strings.Contains(t, "GGC") || strings.Contains(t, "GOOGLE") ||
		strings.Contains(t, "GLOBO") || strings.Contains(t, "YOUTUBE") ||
		strings.Contains(t, "STREAM") || iface.Role == "cache"
}

func ifaceMatchesService(iface SNMPInterface, service string) bool {
	t := strings.ToUpper(iface.Alias + " " + iface.Name)
	svc := strings.ToUpper(service)
	keys := []string{}
	switch {
	case strings.Contains(svc, "NETFLIX"):
		keys = []string{"NETFLIX"}
	case strings.Contains(svc, "GLOBO"):
		keys = []string{"GLOBO"}
	case strings.Contains(svc, "YOUTUBE") || strings.Contains(svc, "GOOGLE"):
		keys = []string{"GOOGLE", "GGC", "YOUTUBE"}
	case strings.Contains(svc, "APPLE"):
		keys = []string{"APPLE"}
	default:
		keys = []string{"CACHE", "STREAM"}
	}
	for _, k := range keys {
		if strings.Contains(t, k) {
			return true
		}
	}
	return false
}

func handleStreamingPage(w http.ResponseWriter, r *http.Request) {
	stats := store.GetStats()
	cache := computeCacheHit()

	scaled := stats.ByCategoryMbpsScaled
	if scaled == nil {
		scaled = map[string]float64{}
	}
	total := scaled["streaming"] + scaled["netflix"] + scaled["globo"] + scaled["apple"]

	var ipv4, ipv6 float64
	for _, r := range stats.StreamingRates {
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
			warn = fmt.Sprintf("Divergência cache SNMP (%.0f Mbps) vs streaming classificado (%.0f Mbps)", cacheTotal, total)
		}
	}

	// Social video relacionado (TikTok etc.) — opcional na UI
	related := []ServiceRate{}
	for _, d := range stats.TopDestinations {
		if d.Category == "social" && (strings.Contains(strings.ToLower(d.Name), "tiktok") ||
			strings.Contains(strings.ToLower(d.Name), "meta") ||
			strings.Contains(strings.ToLower(d.Name), "whatsapp") ||
			strings.Contains(strings.ToLower(d.Name), "twitter") ||
			strings.Contains(strings.ToLower(d.Name), "instagram")) {
			related = append(related, ServiceRate{
				Name: d.Name, Bytes: d.Bytes, Mbps: d.Mbps, MbpsScaled: d.MbpsScaled,
				Category: "social", Percentage: d.Percentage,
			})
		}
	}

	flows := store.GetRecentFlowsFiltered(40, "", "", "", "")
	streamFlows := make([]FlowRecord, 0, 25)
	streamCats := map[string]bool{"streaming": true, "netflix": true, "globo": true, "apple": true}
	for _, f := range flows {
		if streamCats[string(f.Category)] {
			streamFlows = append(streamFlows, f)
			if len(streamFlows) >= 25 {
				break
			}
		}
	}

	exporter := stats.Exporter
	if exporter == "" {
		exporter = SourceIP
	}

	writeJSON(w, StreamingPageSnapshot{
		StreamingBreakdown:   stats.StreamingBreak,
		StreamingRates:       stats.StreamingRates,
		RelatedSocial:        related,
		ByCategoryMbps:       stats.ByCategoryMbps,
		ByCategoryMbpsScaled: scaled,
		ByCategoryInMbps:     stats.ByCategoryInMbps,
		ByCategoryOutMbps:    stats.ByCategoryOutMbps,
		Sampling:             stats.Sampling,
		SNMP:                 stats.SNMP,
		CacheIfaces:          cache.Interfaces,
		CacheSNMPInMbps:      cache.CacheSNMPInMbps,
		CacheSNMPOutMbps:     cache.CacheSNMPOutMbps,
		CacheHitPct:          cache.EstimatedHitPct,
		TotalMbpsScaled:      total,
		IPv4Mbps:             ipv4,
		IPv6Mbps:             ipv6,
		UplinkSharePct:       share,
		DivergenceWarn:       warn,
		Flows:                streamFlows,
		Exporter:             exporter,
		WindowHint:           "Mbps = janela ~10s × amostragem SNMP",
		BytesHint:            "Bytes / % = acumulado do dia",
		IncludeNote:          "Total inclui streaming + netflix + globo + apple (Apple TV+). TikTok/Meta ficam em social (chip opcional).",
	})
}

func handleStreamingDetail(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		http.Error(w, `{"error":"name required"}`, http.StatusBadRequest)
		return
	}
	stats := store.GetStats()
	var live *ServiceRate
	for i := range stats.StreamingRates {
		if strings.EqualFold(stats.StreamingRates[i].Name, name) {
			live = &stats.StreamingRates[i]
			name = stats.StreamingRates[i].Name
			break
		}
	}
	if live == nil {
		// fallback from breakdown
		for k, b := range stats.StreamingBreak {
			if strings.EqualFold(k, name) {
				live = &ServiceRate{Name: k, Bytes: b, Category: "streaming"}
				name = k
				break
			}
		}
	}

	flowsAll := store.GetRecentFlowsFiltered(100, "", "", "", "")
	flows := make([]FlowRecord, 0, 40)
	for _, f := range flowsAll {
		hay := f.Destination + " " + f.Origin + " " + string(f.Category)
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
	type hp = StreamingHistoryPoint
	hist := make([]hp, 0, len(histPts))
	for _, p := range histPts {
		if p.Ts < since {
			continue
		}
		v := 0.0
		if p.ByStreamingScaled != nil {
			if x, ok := p.ByStreamingScaled[name]; ok {
				v = x
			} else {
				for k, x := range p.ByStreamingScaled {
					if strings.EqualFold(k, name) {
						v = x
						break
					}
				}
			}
		}
		hist = append(hist, hp{Ts: p.Ts, MbpsScaled: v})
	}

	snmp := snmpStore.Get()
	var matched []SNMPInterface
	var snmpMbps float64
	for _, iface := range snmp.Interfaces {
		if isStreamingCacheIface(iface) && ifaceMatchesService(iface, name) {
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

	writeJSON(w, StreamingDetailSnapshot{
		Name:           name,
		Live:           live,
		Flows:          flows,
		History:        hist,
		CacheIfaces:    matched,
		SNMPMatchMbps:  snmpMbps,
		DivergenceWarn: warn,
		Sampling:       stats.Sampling,
		SNMP:           &snmp,
	})
}
