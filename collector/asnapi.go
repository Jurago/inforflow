package main

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

type ASNPageSnapshot struct {
	Destinations []ASNTraffic       `json:"destinations"`
	Peers        []ASNTraffic       `json:"peers"`
	Sampling     *SamplingEstimate  `json:"sampling,omitempty"`
	SNMP         *SNMPSnapshot      `json:"snmp,omitempty"`
	MbpsScaled   float64            `json:"mbps_scaled"`
	IPv4Mbps     float64            `json:"ipv4_mbps"`
	IPv6Mbps     float64            `json:"ipv6_mbps"`
	Exporter     string             `json:"exporter"`
	Source       string             `json:"source"`
	WindowHint   string             `json:"window_hint"`
	BytesHint    string             `json:"bytes_hint"`
	Names        map[string]string  `json:"names"` // AS123 → nome (legendas)
	Daily        *ASNDailySnapshot  `json:"daily,omitempty"`
}

type ASNDailyEntry struct {
	ASN        string  `json:"asn"`
	Name       string  `json:"name"`
	Bytes      int64   `json:"bytes"`
	Flows      int64   `json:"flows"`
	Percentage float64 `json:"percentage"`
}

type ASNDailySnapshot struct {
	Day     string          `json:"day"`
	Entries []ASNDailyEntry `json:"entries"`
	Total   int64           `json:"total_bytes"`
}

type ASNDetailHistoryPoint struct {
	Ts         int64   `json:"ts"`
	MbpsScaled float64 `json:"mbps_scaled"`
	PeerScaled float64 `json:"peer_mbps_scaled,omitempty"`
}

type ASNDetailSnapshot struct {
	ASN      string                  `json:"asn"`
	Live     *ASNTraffic             `json:"live,omitempty"`
	PeerLive *ASNTraffic             `json:"peer_live,omitempty"`
	Daily    *ASNDailyEntry          `json:"daily,omitempty"`
	Flows    []FlowRecord            `json:"flows"`
	History  []ASNDetailHistoryPoint `json:"history"`
	Sampling *SamplingEstimate       `json:"sampling,omitempty"`
	SNMP     *SNMPSnapshot           `json:"snmp,omitempty"`
}

func csvEscape(s string) string {
	if strings.ContainsAny(s, ",\"\n") {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}

func buildASNNames(dest, peers []ASNTraffic) map[string]string {
	out := make(map[string]string, len(dest)+len(peers))
	for _, a := range dest {
		out[a.ASN] = a.Name
	}
	for _, a := range peers {
		if _, ok := out[a.ASN]; !ok {
			out[a.ASN] = a.Name
		}
	}
	return out
}

func handleASN(w http.ResponseWriter, r *http.Request) {
	stats := store.GetStatsCached()
	daily := getASNDailySnapshot()
	writeJSON(w, ASNPageSnapshot{
		Destinations: stats.ASNBreakdown,
		Peers:        stats.PeerASNBreakdown,
		Sampling:     stats.Sampling,
		SNMP:         stats.SNMP,
		MbpsScaled:   stats.MbpsScaled,
		IPv4Mbps:     stats.IPv4Mbps,
		IPv6Mbps:     stats.IPv6Mbps,
		Exporter:     stats.Exporter,
		Source:       stats.Source,
		WindowHint:   "Mbps = janela ~10s × amostragem SNMP",
		BytesHint:    "Bytes / % = acumulado do dia",
		Names:        buildASNNames(stats.ASNBreakdown, stats.PeerASNBreakdown),
		Daily:        daily,
	})
}

func getASNDailySnapshot() *ASNDailySnapshot {
	store.mu.RLock()
	defer store.mu.RUnlock()
	type kv struct {
		as    uint32
		bytes int64
		flows int64
	}
	ranked := make([]kv, 0, len(store.dstASNBytes))
	var total int64
	for as, b := range store.dstASNBytes {
		if as == 0 || b <= 0 {
			continue
		}
		ranked = append(ranked, kv{as, b, store.dstASNFlows[as]})
		total += b
	}
	sort.Slice(ranked, func(i, j int) bool { return ranked[i].bytes > ranked[j].bytes })
	if len(ranked) > 50 {
		ranked = ranked[:50]
	}
	if total <= 0 {
		total = 1
	}
	entries := make([]ASNDailyEntry, 0, len(ranked))
	for _, r := range ranked {
		name := asnDisplayName(r.as)
		if c := classifyByASN(r.as); c.Name != "" {
			name = c.Name
		} else if n := ipapi.NameForASN(r.as); n != "" {
			name = n
		} else if p, ok := bgpStore.LookupAS(r.as); ok {
			name = p.Name
		}
		entries = append(entries, ASNDailyEntry{
			ASN:        formatASN(r.as),
			Name:       name,
			Bytes:      r.bytes,
			Flows:      r.flows,
			Percentage: float64(r.bytes) / float64(total) * 100,
		})
	}
	return &ASNDailySnapshot{
		Day:     time.Now().Format("2006-01-02"),
		Entries: entries,
		Total:   total,
	}
}

func handleASNDaily(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, getASNDailySnapshot())
}

func handleASNDetail(w http.ResponseWriter, r *http.Request) {
	asnQ := normalizeASNQuery(r.URL.Query().Get("asn"))
	if asnQ == "" {
		http.Error(w, `{"error":"asn required"}`, http.StatusBadRequest)
		return
	}
	asNum := parseASNNum(asnQ)
	stats := store.GetStatsCached()
	var live, peerLive *ASNTraffic
	for i := range stats.ASNBreakdown {
		if stats.ASNBreakdown[i].ASN == asnQ {
			live = &stats.ASNBreakdown[i]
			break
		}
	}
	for i := range stats.PeerASNBreakdown {
		if stats.PeerASNBreakdown[i].ASN == asnQ {
			peerLive = &stats.PeerASNBreakdown[i]
			break
		}
	}
	var dailyEntry *ASNDailyEntry
	if d := getASNDailySnapshot(); d != nil {
		for i := range d.Entries {
			if d.Entries[i].ASN == asnQ {
				dailyEntry = &d.Entries[i]
				break
			}
		}
	}

	flows := store.GetASNRecentFlows(asNum, 80)
	if len(flows) == 0 {
		flows = store.GetRecentFlowsFiltered(80, "", "", "", asnQ)
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
	type hp = ASNDetailHistoryPoint
	hist := make([]hp, 0, len(histPts))
	for _, p := range histPts {
		if p.Ts < since {
			continue
		}
		v := 0.0
		if p.ByASNScaled != nil {
			v = p.ByASNScaled[asnQ]
		} else if p.ByASN != nil {
			v = p.ByASN[asnQ] * p.SamplingFactor
		}
		pv := 0.0
		if p.ByPeerASNScaled != nil {
			pv = p.ByPeerASNScaled[asnQ]
		}
		hist = append(hist, hp{Ts: p.Ts, MbpsScaled: v, PeerScaled: pv})
	}

	writeJSON(w, ASNDetailSnapshot{
		ASN:      asnQ,
		Live:     live,
		PeerLive: peerLive,
		Daily:    dailyEntry,
		Flows:    flows,
		History:  hist,
		Sampling: stats.Sampling,
		SNMP:     stats.SNMP,
	})
}

func parseIntSafe(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

// StartASNDigest — envia resumo diário dos top ASNs via webhook/telegram.
func StartASNDigest() {
	go func() {
		var lastDay string
		for {
			time.Sleep(time.Minute)
			c := GetConfig()
			if c.ASNDigestHour < 0 {
				continue
			}
			now := time.Now()
			if now.Hour() != c.ASNDigestHour || now.Minute() > 2 {
				continue
			}
			day := now.Format("2006-01-02")
			if day == lastDay {
				continue
			}
			lastDay = day
			sendASNDigest()
		}
	}()
}

func sendASNDigest() {
	d := getASNDailySnapshot()
	if d == nil || len(d.Entries) == 0 {
		return
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Inforflow ASN digest %s\nTotal: %s\n\n", d.Day, formatBytesHuman(d.Total))
	limit := 15
	if len(d.Entries) < limit {
		limit = len(d.Entries)
	}
	for i := 0; i < limit; i++ {
		e := d.Entries[i]
		fmt.Fprintf(&b, "%d. %s (%s) — %s (%.1f%%) · %d flows\n",
			i+1, e.Name, e.ASN, formatBytesHuman(e.Bytes), e.Percentage, e.Flows)
	}
	detail := b.String()
	alerts.Raise("asn_digest_"+d.Day, "Digest ASN diário", detail, AlertInfo)
	// notifyAlert já é chamado por Raise; limpa para não ficar “ativo”
	alerts.Clear("asn_digest_" + d.Day)
}

func formatBytesHuman(n int64) string {
	const k = 1024.0
	f := float64(n)
	switch {
	case f >= k*k*k*k:
		return fmt.Sprintf("%.2f TB", f/(k*k*k*k))
	case f >= k*k*k:
		return fmt.Sprintf("%.2f GB", f/(k*k*k))
	case f >= k*k:
		return fmt.Sprintf("%.2f MB", f/(k*k))
	case f >= k:
		return fmt.Sprintf("%.2f KB", f/k)
	default:
		return fmt.Sprintf("%d B", n)
	}
}
