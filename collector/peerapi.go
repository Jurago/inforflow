package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

type PeersPageSnapshot struct {
	BGP              *BGPSnapshot      `json:"bgp"`
	PeerBreakdown    []DestOriginCard  `json:"peer_breakdown"`
	PeerASNBreakdown []ASNTraffic      `json:"peer_asn_breakdown"`
	Sampling         *SamplingEstimate `json:"sampling,omitempty"`
	SNMP             *SNMPSnapshot     `json:"snmp,omitempty"`
	SNMPPeerIfaces   []SNMPInterface   `json:"snmp_peer_ifaces,omitempty"`
	SNMPPeerMbpsIn   float64           `json:"snmp_peer_mbps_in"`
	SNMPPeerMbpsOut  float64           `json:"snmp_peer_mbps_out"`
	PeerMbpsScaled   float64           `json:"peer_mbps_scaled"`
	IXMbpsScaled     float64           `json:"ix_mbps_scaled"`
	IXASN            uint32            `json:"ix_asn"`
	IXName           string            `json:"ix_name"`
	DivergenceWarn   string            `json:"divergence_warn,omitempty"`
	DownPeers        []BGPPeer         `json:"down_peers"`
	Flows            []FlowRecord      `json:"flows"`
	MbpsScaled       float64           `json:"mbps_scaled"`
	Exporter         string            `json:"exporter"`
	WindowHint       string            `json:"window_hint"`
	BytesHint        string            `json:"bytes_hint"`
}

type PeersDetailSnapshot struct {
	ASN      string       `json:"asn"`
	Sessions []BGPPeer    `json:"sessions"`
	Live     *ASNTraffic  `json:"live,omitempty"`
	Card     *DestOriginCard `json:"card,omitempty"`
	Flows    []FlowRecord `json:"flows"`
	History  []ASNDetailHistoryPoint `json:"history"`
	Sampling *SamplingEstimate `json:"sampling,omitempty"`
	SNMP     *SNMPSnapshot     `json:"snmp,omitempty"`
	SNMPIfaces []SNMPInterface `json:"snmp_ifaces,omitempty"`
}

func peerSNMPIfaces(snmp SNMPSnapshot) (ifaces []SNMPInterface, inSum, outSum float64) {
	for _, iface := range snmp.Interfaces {
		role := strings.ToLower(iface.Role)
		if role == "ix" || role == "transit" || role == "peer" {
			ifaces = append(ifaces, iface)
			if iface.OperStatus == 1 {
				inSum += iface.InMbps
				outSum += iface.OutMbps
			}
		}
	}
	return
}

func handlePeersPage(w http.ResponseWriter, r *http.Request) {
	stats := store.GetStatsCached()
	bgp := stats.BGP
	if bgp == nil {
		snap := bgpStore.Get()
		bgp = &snap
	}
	ixASN := GetConfig().IXASN
	if ixASN == 0 {
		ixASN = 26162
	}
	if bgp.IXASN > 0 {
		ixASN = bgp.IXASN
	}

	var peerMbps, ixMbps float64
	seen := map[uint32]bool{}
	for _, p := range bgp.Peers {
		if seen[p.RemoteAS] {
			continue
		}
		seen[p.RemoteAS] = true
		peerMbps += p.MbpsScaled
		if p.RemoteAS == ixASN {
			ixMbps = p.MbpsScaled
		}
	}
	// Prefer peer_asn_breakdown if richer
	if len(stats.PeerASNBreakdown) > 0 {
		peerMbps = 0
		for _, a := range stats.PeerASNBreakdown {
			peerMbps += a.MbpsScaled
			if parseASNNum(a.ASN) == ixASN {
				ixMbps = a.MbpsScaled
			}
		}
	}

	snmp := snmpStore.Get()
	ifaces, snmpIn, snmpOut := peerSNMPIfaces(snmp)
	warn := ""
	snmpAvg := (snmpIn + snmpOut) / 2
	if snmp.OK && snmpAvg > 50 && peerMbps > 0 {
		ratio := peerMbps / snmpAvg
		if ratio < 0.4 || ratio > 2.5 {
			warn = fmt.Sprintf("Divergência NetFlow×sampling (%.0f Mbps) vs SNMP IX/transit (%.0f Mbps)", peerMbps, snmpAvg)
		}
	}

	down := make([]BGPPeer, 0)
	for _, p := range bgp.Peers {
		if !p.Established {
			down = append(down, p)
		}
	}

	flows := store.GetRecentFlowsFiltered(40, "", "", "", "")
	peerFlows := make([]FlowRecord, 0, 25)
	for _, f := range flows {
		if f.PeerASN != "" || f.Category == CategoryPeer {
			peerFlows = append(peerFlows, f)
			if len(peerFlows) >= 25 {
				break
			}
		}
	}

	writeJSON(w, PeersPageSnapshot{
		BGP:              bgp,
		PeerBreakdown:    stats.PeerBreakdown,
		PeerASNBreakdown: stats.PeerASNBreakdown,
		Sampling:         stats.Sampling,
		SNMP:             &snmp,
		SNMPPeerIfaces:   ifaces,
		SNMPPeerMbpsIn:   snmpIn,
		SNMPPeerMbpsOut:  snmpOut,
		PeerMbpsScaled:   peerMbps,
		IXMbpsScaled:     ixMbps,
		IXASN:            ixASN,
		IXName:           asnDisplayName(ixASN),
		DivergenceWarn:   warn,
		DownPeers:        down,
		Flows:            peerFlows,
		MbpsScaled:       stats.MbpsScaled,
		Exporter:         stats.Exporter,
		WindowHint:       "Mbps = janela ~10s × amostragem SNMP",
		BytesHint:        "Bytes = acumulado do dia / desde o start",
	})
}

func handlePeersDetail(w http.ResponseWriter, r *http.Request) {
	asnQ := normalizeASNQuery(r.URL.Query().Get("asn"))
	if asnQ == "" {
		http.Error(w, `{"error":"asn required"}`, http.StatusBadRequest)
		return
	}
	asNum := parseASNNum(asnQ)
	stats := store.GetStatsCached()
	bgp := bgpStore.Get()
	var sessions []BGPPeer
	for _, p := range bgp.Peers {
		if p.RemoteAS == asNum || p.ASN == asnQ {
			sessions = append(sessions, p)
		}
	}
	var live *ASNTraffic
	for i := range stats.PeerASNBreakdown {
		if stats.PeerASNBreakdown[i].ASN == asnQ {
			live = &stats.PeerASNBreakdown[i]
			break
		}
	}
	var card *DestOriginCard
	for i := range stats.PeerBreakdown {
		if strings.Contains(stats.PeerBreakdown[i].Name, asnQ) {
			card = &stats.PeerBreakdown[i]
			break
		}
	}

	flows := store.GetRecentFlowsFiltered(80, "", "", "", asnQ)
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
	hist := make([]ASNDetailHistoryPoint, 0, len(histPts))
	for _, p := range histPts {
		if p.Ts < since {
			continue
		}
		v := 0.0
		if p.ByPeerASNScaled != nil {
			v = p.ByPeerASNScaled[asnQ]
		} else if p.ByPeerASN != nil {
			v = p.ByPeerASN[asnQ] * p.SamplingFactor
		}
		hist = append(hist, ASNDetailHistoryPoint{Ts: p.Ts, PeerScaled: v, MbpsScaled: v})
	}

	snmp := snmpStore.Get()
	ifaces, _, _ := peerSNMPIfaces(snmp)

	writeJSON(w, PeersDetailSnapshot{
		ASN:        asnQ,
		Sessions:   sessions,
		Live:       live,
		Card:       card,
		Flows:      flows,
		History:    hist,
		Sampling:   stats.Sampling,
		SNMP:       &snmp,
		SNMPIfaces: ifaces,
	})
}
