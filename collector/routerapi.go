package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type RouterRoleSum struct {
	Role    string  `json:"role"`
	InMbps  float64 `json:"in_mbps"`
	OutMbps float64 `json:"out_mbps"`
	Count   int     `json:"count"`
	UtilMax float64 `json:"util_max_pct"`
}

type RouterPageSnapshot struct {
	OK              bool               `json:"ok"`
	Host            string             `json:"host"`
	Port            int                `json:"port"`
	AuthOK          bool               `json:"auth_ok"`
	SysName         string             `json:"sys_name"`
	SysDescr        string             `json:"sys_descr,omitempty"`
	UptimeHuman     string             `json:"uptime_human"`
	CPUPct          float64            `json:"cpu_pct"`
	MemPct          float64            `json:"mem_pct"`
	UpdatedAt       int64              `json:"updated_at"`
	AgeSec          int64              `json:"age_sec"`
	PollMs          int64              `json:"poll_ms"`
	UplinkInMbps    float64            `json:"uplink_in_mbps"`
	UplinkOutMbps   float64            `json:"uplink_out_mbps"`
	UplinkUtilPct   float64            `json:"uplink_util_pct"`
	IfacesUp        int                `json:"ifaces_up"`
	IfacesDown      int                `json:"ifaces_down"`
	CacheMbps       float64            `json:"cache_mbps"`
	IXMbps          float64            `json:"ix_mbps"`
	BrasMbps        float64            `json:"bras_mbps"`
	CGNATMbps       float64            `json:"cgnat_mbps"`
	RoleSums        []RouterRoleSum    `json:"role_sums"`
	Interfaces      []SNMPInterface    `json:"interfaces"`
	TopIn           []SNMPInterface    `json:"top_in"`
	TopOut          []SNMPInterface    `json:"top_out"`
	HighUtil        []SNMPInterface    `json:"high_util,omitempty"`
	ByIfaceRoleNF   map[string]float64 `json:"by_iface_role_nf_mbps,omitempty"`
	Sampling        *SamplingEstimate  `json:"sampling,omitempty"`
	Alerts          []Alert            `json:"alerts,omitempty"`
	DivergenceWarn  string             `json:"divergence_warn,omitempty"`
	Error           string             `json:"error,omitempty"`
	WindowHint      string             `json:"window_hint"`
}

type RouterDetailSnapshot struct {
	Iface          *SNMPInterface `json:"iface,omitempty"`
	History        []struct {
		Ts      int64   `json:"ts"`
		InMbps  float64 `json:"in_mbps"`
		OutMbps float64 `json:"out_mbps"`
	} `json:"history"`
	Flows          []FlowRecord      `json:"flows"`
	NFMbps         float64           `json:"nf_mbps_scaled"`
	DivergenceWarn string            `json:"divergence_warn,omitempty"`
	Sampling       *SamplingEstimate `json:"sampling,omitempty"`
	Host           string            `json:"host"`
}

func handleRouterPage(w http.ResponseWriter, r *http.Request) {
	snmp := snmpStore.Get()
	stats := store.GetStats()
	now := time.Now().Unix()
	age := int64(0)
	if snmp.UpdatedAt > 0 {
		age = now - snmp.UpdatedAt
	}

	up, down := 0, 0
	cacheM, ixM, brasM, cgnatM := 0.0, 0.0, 0.0, 0.0
	roleMap := map[string]*RouterRoleSum{}
	var high []SNMPInterface
	for _, iface := range snmp.Interfaces {
		if iface.OperStatus == 1 {
			up++
		} else {
			down++
		}
		total := iface.InMbps + iface.OutMbps
		role := iface.Role
		if role == "" {
			role = "other"
		}
		rs := roleMap[role]
		if rs == nil {
			rs = &RouterRoleSum{Role: role}
			roleMap[role] = rs
		}
		rs.InMbps += iface.InMbps
		rs.OutMbps += iface.OutMbps
		rs.Count++
		util := iface.InUtilPct
		if iface.OutUtilPct > util {
			util = iface.OutUtilPct
		}
		if util > rs.UtilMax {
			rs.UtilMax = util
		}
		switch iface.Role {
		case "cache":
			cacheM += total
		case "ix":
			ixM += total
		case "bras":
			brasM += total
		case "cgnat":
			cgnatM += total
		}
		if iface.OperStatus == 1 && util >= 80 {
			high = append(high, iface)
		}
	}
	roles := make([]RouterRoleSum, 0, len(roleMap))
	for _, rs := range roleMap {
		roles = append(roles, *rs)
	}

	warn := ""
	if snmp.OK && stats.ByIfaceRole != nil {
		nfCache := stats.ByIfaceRole["cache"]
		if cacheM > 50 && nfCache > 0 {
			ratio := cacheM / nfCache
			if ratio < 0.2 || ratio > 5 {
				warn = fmt.Sprintf("Cache SNMP %.0f Mbps vs NetFlow×role %.0f Mbps", cacheM, nfCache)
			}
		}
	}

	activeAlerts := []Alert{}
	for _, a := range alerts.Active() {
		if strings.HasPrefix(a.Code, "util_") || strings.HasPrefix(a.Code, "snmp_") {
			activeAlerts = append(activeAlerts, a)
		}
	}

	writeJSON(w, RouterPageSnapshot{
		OK:             snmp.OK,
		Host:           snmp.Host,
		Port:           snmp.Port,
		AuthOK:         snmp.OK,
		SysName:        snmp.SysName,
		SysDescr:       snmp.SysDescr,
		UptimeHuman:    snmp.UptimeHuman,
		CPUPct:         snmp.CPUPct,
		MemPct:         snmp.MemPct,
		UpdatedAt:      snmp.UpdatedAt,
		AgeSec:         age,
		PollMs:         snmp.PollMs,
		UplinkInMbps:   snmp.UplinkInMbps,
		UplinkOutMbps:  snmp.UplinkOutMbps,
		UplinkUtilPct:  snmp.UplinkUtilPct,
		IfacesUp:       up,
		IfacesDown:     down,
		CacheMbps:      cacheM,
		IXMbps:         ixM,
		BrasMbps:       brasM,
		CGNATMbps:      cgnatM,
		RoleSums:       roles,
		Interfaces:     snmp.Interfaces,
		TopIn:          snmp.TopIn,
		TopOut:         snmp.TopOut,
		HighUtil:       high,
		ByIfaceRoleNF:  stats.ByIfaceRole,
		Sampling:       stats.Sampling,
		Alerts:         activeAlerts,
		DivergenceWarn: warn,
		Error:          snmp.Error,
		WindowHint:     "Mbps SNMP = delta octets entre polls · NF role = NetFlow × amostragem",
	})
}

func handleRouterDetail(w http.ResponseWriter, r *http.Request) {
	idxStr := r.URL.Query().Get("ifindex")
	nameQ := strings.TrimSpace(r.URL.Query().Get("name"))
	if idxStr == "" && nameQ == "" {
		http.Error(w, `{"error":"ifindex or name required"}`, http.StatusBadRequest)
		return
	}
	idx := 0
	if idxStr != "" {
		if n, err := strconv.Atoi(idxStr); err == nil {
			idx = n
		}
	}
	snmp := snmpStore.Get()
	var iface *SNMPInterface
	for i := range snmp.Interfaces {
		it := &snmp.Interfaces[i]
		if idx > 0 && it.Index == idx {
			iface = it
			break
		}
		if nameQ != "" && (strings.EqualFold(it.Name, nameQ) || strings.EqualFold(it.Alias, nameQ)) {
			iface = it
			idx = it.Index
			break
		}
	}
	hours := 1
	if v := r.URL.Query().Get("hours"); v != "" {
		if n, err := parseIntSafe(v); err == nil && n > 0 && n <= 24 {
			hours = n
		}
	}
	since := time.Now().Unix() - int64(hours)*3600
	hist := snmpStore.GetIfaceHistory(since, idx)

	flowsAll := store.GetRecentFlowsFiltered(100, "", "", "", "")
	flows := make([]FlowRecord, 0, 40)
	var nfBytes int64
	for _, f := range flowsAll {
		if idx > 0 && (f.InIf == idx || f.OutIf == idx) {
			flows = append(flows, f)
			nfBytes += f.Bytes
			if len(flows) >= 40 {
				break
			}
		}
	}
	eff := 1.0
	if stats := store.GetStats(); stats.Sampling != nil && stats.Sampling.Effective > 1 {
		eff = stats.Sampling.Effective
	}
	nfMbps := float64(nfBytes) * 8 / 10 / 1e6 * eff
	warn := ""
	if iface != nil {
		snmpTotal := iface.InMbps + iface.OutMbps
		if snmpTotal > 10 && nfMbps > 0 {
			ratio := snmpTotal / nfMbps
			if ratio < 0.2 || ratio > 5 {
				warn = fmt.Sprintf("SNMP %.0f Mbps vs NetFlow×if %.0f Mbps (amostra flows buffer)", snmpTotal, nfMbps)
			}
		}
	}
	stats := store.GetStats()
	writeJSON(w, RouterDetailSnapshot{
		Iface:          iface,
		History:        hist,
		Flows:          flows,
		NFMbps:         nfMbps,
		DivergenceWarn: warn,
		Sampling:       stats.Sampling,
		Host:           snmp.Host,
	})
}
