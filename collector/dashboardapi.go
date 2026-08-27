package main

import (
	"net/http"
	"sort"
	"time"
)

type DashboardSparkPoint struct {
	Ts         int64   `json:"ts"`
	MbpsScaled float64 `json:"mbps_scaled"`
	SNMPIn     float64 `json:"snmp_in_mbps"`
	SNMPOut    float64 `json:"snmp_out_mbps"`
}

type DashboardBlockShare struct {
	Key        string  `json:"key"`
	Label      string  `json:"label"`
	MbpsScaled float64 `json:"mbps_scaled"`
	SharePct   float64 `json:"share_pct"`
	Href       string  `json:"href"`
}

type DashboardBGPPeer struct {
	ASN         string  `json:"asn"`
	Name        string  `json:"name"`
	RemoteAddr  string  `json:"remote_addr"`
	Role        string  `json:"role"`
	Established bool    `json:"established"`
	StateName   string  `json:"state_name"`
	Mbps        float64 `json:"mbps"`
	MbpsScaled  float64 `json:"mbps_scaled"`
}

type DashboardBGPSummary struct {
	OK          bool               `json:"ok"`
	LocalAS     uint32             `json:"local_as,omitempty"`
	Established int                `json:"established"`
	Total       int                `json:"total"`
	Down        int                `json:"down"`
	Error       string             `json:"error,omitempty"`
	TopUp       []DashboardBGPPeer `json:"top_up,omitempty"`
	TopDown     []DashboardBGPPeer `json:"top_down,omitempty"`
}

type DashboardSNMPLite struct {
	OK            bool            `json:"ok"`
	Host          string          `json:"host"`
	SysName       string          `json:"sys_name"`
	UptimeHuman   string          `json:"uptime_human"`
	CPUPct        float64         `json:"cpu_pct"`
	MemPct        float64         `json:"mem_pct"`
	UpdatedAt     int64           `json:"updated_at"`
	AgeSec        int64           `json:"age_sec"`
	UplinkInMbps  float64         `json:"uplink_in_mbps"`
	UplinkOutMbps float64         `json:"uplink_out_mbps"`
	UplinkUtilPct float64         `json:"uplink_util_pct"`
	Deduped       bool            `json:"deduped"`
	Critical      []SNMPInterface `json:"critical,omitempty"`
	Error         string          `json:"error,omitempty"`
}

type DashboardPageSnapshot struct {
	Mbps             float64                `json:"mbps"`
	MbpsScaled       float64                `json:"mbps_scaled"`
	IPv4Mbps         float64                `json:"ipv4_mbps"`
	IPv6Mbps         float64                `json:"ipv6_mbps"`
	BytesPerSec      float64                `json:"bytes_per_sec"`
	PacketsPerSec    float64                `json:"packets_per_sec"`
	TotalBytes       int64                  `json:"total_bytes"`
	TotalFlows       int64                  `json:"total_flows"`
	ClassifiedPct    float64                `json:"classified_pct"`
	Exporter         string                 `json:"exporter"`
	Source           string                 `json:"source"`
	GapMbps          float64                `json:"gap_mbps"`
	GapPct           float64                `json:"gap_pct"`
	SNMPAvgMbps      float64                `json:"snmp_avg_mbps"`
	Compare24hDelta  float64                `json:"compare_24h_delta_mbps"`
	Compare24hPct    float64                `json:"compare_24h_pct"`
	BlockShares      []DashboardBlockShare  `json:"block_shares"`
	Consumption      []CategoryConsumption  `json:"consumption"`
	ByCategory       map[string]int64       `json:"by_category"`
	ByCategoryMbps   map[string]float64     `json:"by_category_mbps_scaled"`
	TopDestinations  []DestOriginCard       `json:"top_destinations"`
	TopOrigins       []DestOriginCard       `json:"top_origins"`
	TopTalkers       []TalkerStats          `json:"top_talkers"`
	DestinationCount int                    `json:"destination_count"`
	Sampling         *SamplingEstimate      `json:"sampling,omitempty"`
	Alerts           []Alert                `json:"alerts,omitempty"`
	BGP              DashboardBGPSummary    `json:"bgp"`
	SNMP             DashboardSNMPLite      `json:"snmp"`
	Sparkline        []DashboardSparkPoint  `json:"sparkline"`
	Flows            []FlowRecord           `json:"flows"`
	WindowHint       string                 `json:"window_hint"`
}

func handleDashboardPage(w http.ResponseWriter, r *http.Request) {
	stats := store.GetStatsCached()
	snmpAvg := 0.0
	if stats.SNMP != nil && stats.SNMP.OK {
		snmpAvg = (stats.SNMP.UplinkInMbps + stats.SNMP.UplinkOutMbps) / 2
	}
	gap := absFloat(stats.MbpsScaled - snmpAvg)
	gapPct := 0.0
	if snmpAvg > 1 {
		gapPct = gap / snmpAvg * 100
	}

	scaled := stats.ByCategoryMbpsScaled
	if scaled == nil {
		scaled = map[string]float64{}
	}
	cdn := scaled["cdn"]
	stream := scaled["streaming"] + scaled["netflix"] + scaled["globo"] + scaled["apple"]
	peer := scaled["peer"]
	other := stats.MbpsScaled - cdn - stream - peer
	if other < 0 {
		other = 0
	}
	uplink := snmpAvg
	if uplink < 1 {
		uplink = stats.MbpsScaled
		if uplink < 1 {
			uplink = 1
		}
	}
	blocks := []DashboardBlockShare{
		{Key: "cdn", Label: "CDN", MbpsScaled: cdn, SharePct: cdn / uplink * 100, Href: "/cdns"},
		{Key: "streaming", Label: "Streaming", MbpsScaled: stream, SharePct: stream / uplink * 100, Href: "/streaming"},
		{Key: "peer", Label: "Peers", MbpsScaled: peer, SharePct: peer / uplink * 100, Href: "/peers"},
		{Key: "other", Label: "Outros", MbpsScaled: other, SharePct: other / uplink * 100, Href: "/asn"},
	}

	bgpSum := DashboardBGPSummary{}
	if stats.BGP != nil {
		bgpSum.OK = stats.BGP.OK
		bgpSum.LocalAS = stats.BGP.LocalAS
		bgpSum.Established = stats.BGP.Established
		bgpSum.Total = stats.BGP.Total
		bgpSum.Down = stats.BGP.Down
		bgpSum.Error = stats.BGP.Error
		peers := append([]BGPPeer(nil), stats.BGP.Peers...)
		sort.Slice(peers, func(i, j int) bool {
			return peers[i].MbpsScaled > peers[j].MbpsScaled
		})
		for _, p := range peers {
			item := DashboardBGPPeer{
				ASN: p.ASN, Name: p.Name, RemoteAddr: p.RemoteAddr, Role: p.Role,
				Established: p.Established, StateName: p.StateName,
				Mbps: p.Mbps, MbpsScaled: p.MbpsScaled,
			}
			if p.Established && len(bgpSum.TopUp) < 8 {
				bgpSum.TopUp = append(bgpSum.TopUp, item)
			}
			if !p.Established && len(bgpSum.TopDown) < 4 {
				bgpSum.TopDown = append(bgpSum.TopDown, item)
			}
		}
	}

	snmpLite := DashboardSNMPLite{}
	if stats.SNMP != nil {
		s := stats.SNMP
		snmpLite.OK = s.OK
		snmpLite.Host = s.Host
		snmpLite.SysName = s.SysName
		snmpLite.UptimeHuman = s.UptimeHuman
		snmpLite.CPUPct = s.CPUPct
		snmpLite.MemPct = s.MemPct
		snmpLite.UpdatedAt = s.UpdatedAt
		if s.UpdatedAt > 0 {
			snmpLite.AgeSec = time.Now().Unix() - s.UpdatedAt
		}
		snmpLite.UplinkInMbps = s.UplinkInMbps
		snmpLite.UplinkOutMbps = s.UplinkOutMbps
		snmpLite.UplinkUtilPct = s.UplinkUtilPct
		snmpLite.Deduped = s.Deduped
		snmpLite.Error = s.Error
		roles := map[string]bool{"bras": true, "uplink": true, "ix": true, "cgnat": true, "cache": true}
		crit := make([]SNMPInterface, 0, 8)
		for _, iface := range s.Interfaces {
			if iface.OperStatus == 1 && roles[iface.Role] {
				crit = append(crit, iface)
			}
		}
		sort.Slice(crit, func(i, j int) bool {
			return crit[i].InMbps+crit[i].OutMbps > crit[j].InMbps+crit[j].OutMbps
		})
		if len(crit) > 8 {
			crit = crit[:8]
		}
		snmpLite.Critical = crit
	}

	// sparkline 1h + compare 24h — uma leitura leve (sem JSON ASN/CDN)
	now := time.Now().Unix()
	histAll := queryHistoryLite(now - 86400 - 3600)
	hist1h := make([]HistoryPoint, 0, 128)
	for _, p := range histAll {
		if p.Ts >= now-3600 {
			hist1h = append(hist1h, p)
		}
	}
	if len(hist1h) == 0 {
		mem := store.GetHistory()
		for _, p := range mem {
			if p.Ts >= now-3600 {
				hist1h = append(hist1h, p)
			}
		}
	}
	hist1h = downsampleHistoryPoints(hist1h, 120)
	spark := make([]DashboardSparkPoint, 0, len(hist1h))
	for _, p := range hist1h {
		spark = append(spark, DashboardSparkPoint{
			Ts: p.Ts, MbpsScaled: p.MbpsScaled, SNMPIn: p.SNMPIn, SNMPOut: p.SNMPOut,
		})
	}

	cmpDelta, cmpPct := 0.0, 0.0
	avg := func(pts []HistoryPoint, minTs, maxTs int64) float64 {
		var s float64
		n := 0
		for _, p := range pts {
			if p.Ts < minTs || p.Ts > maxTs {
				continue
			}
			s += p.MbpsScaled
			n++
		}
		if n == 0 {
			return 0
		}
		return s / float64(n)
	}
	a := avg(histAll, now-3600, now)
	b := avg(histAll, now-86400-3600, now-86400)
	if a == 0 && len(spark) > 0 {
		var s float64
		for _, p := range spark {
			s += p.MbpsScaled
		}
		a = s / float64(len(spark))
	}
	cmpDelta = a - b
	if b > 1 {
		cmpPct = cmpDelta / b * 100
	}

	topDest := stats.TopDestinations
	if len(topDest) > 8 {
		topDest = topDest[:8]
	}
	topOrig := stats.TopOrigins
	if len(topOrig) > 8 {
		topOrig = topOrig[:8]
	}
	talkers := stats.TopTalkers
	if len(talkers) > 10 {
		talkers = talkers[:10]
	}

	flows := store.GetRecentFlowsFiltered(15, "", "", "", "")

	exporter := stats.Exporter
	if exporter == "" {
		exporter = stats.Source
	}
	if exporter == "" {
		exporter = SourceIP
	}

	writeJSON(w, DashboardPageSnapshot{
		Mbps:             stats.Mbps,
		MbpsScaled:       stats.MbpsScaled,
		IPv4Mbps:         stats.IPv4Mbps,
		IPv6Mbps:         stats.IPv6Mbps,
		BytesPerSec:      stats.BytesPerSec,
		PacketsPerSec:    stats.PacketsPerSec,
		TotalBytes:       stats.TotalBytes,
		TotalFlows:       stats.TotalFlows,
		ClassifiedPct:    stats.ClassifiedPct,
		Exporter:         exporter,
		Source:           stats.Source,
		GapMbps:          gap,
		GapPct:           gapPct,
		SNMPAvgMbps:      snmpAvg,
		Compare24hDelta:  cmpDelta,
		Compare24hPct:    cmpPct,
		BlockShares:      blocks,
		Consumption:      stats.Consumption,
		ByCategory:       stats.ByCategory,
		ByCategoryMbps:   scaled,
		TopDestinations:  topDest,
		TopOrigins:       topOrig,
		TopTalkers:       talkers,
		DestinationCount: stats.DestinationCount,
		Sampling:         stats.Sampling,
		Alerts:           stats.Alerts,
		BGP:              bgpSum,
		SNMP:             snmpLite,
		Sparkline:        spark,
		Flows:            flows,
		WindowHint:       "Mbps estimado = NetFlow × amostragem SNMP · SNMP = taxa real de uplink",
	})
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
