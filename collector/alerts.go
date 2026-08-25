package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type AlertSeverity string

const (
	AlertInfo     AlertSeverity = "info"
	AlertWarning  AlertSeverity = "warning"
	AlertCritical AlertSeverity = "critical"
)

type Alert struct {
	ID        string        `json:"id"`
	Ts        int64         `json:"ts"`
	Severity  AlertSeverity `json:"severity"`
	Code      string        `json:"code"`
	Title     string        `json:"title"`
	Detail    string        `json:"detail"`
	Active    bool          `json:"active"`
}

type alertStore struct {
	mu     sync.RWMutex
	active map[string]Alert
	recent []Alert
}

var alerts = &alertStore{
	active: make(map[string]Alert),
	recent: make([]Alert, 0, 100),
}

func (s *alertStore) Raise(code, title, detail string, sev AlertSeverity) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a := Alert{
		ID: code, Ts: time.Now().Unix(), Severity: sev,
		Code: code, Title: title, Detail: detail, Active: true,
	}
	if prev, ok := s.active[code]; ok && prev.Active {
		a.Ts = prev.Ts // keep first seen
		s.active[code] = a
		return
	}
	s.active[code] = a
	s.recent = append([]Alert{a}, s.recent...)
	if len(s.recent) > 100 {
		s.recent = s.recent[:100]
	}
	go notifyAlert(a)
}

func (s *alertStore) Clear(code string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if a, ok := s.active[code]; ok && a.Active {
		a.Active = false
		a.Ts = time.Now().Unix()
		s.active[code] = a
		s.recent = append([]Alert{a}, s.recent...)
		if len(s.recent) > 100 {
			s.recent = s.recent[:100]
		}
	}
	delete(s.active, code)
}

func (s *alertStore) List() []Alert {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Alert, 0, len(s.active)+8)
	for _, a := range s.active {
		out = append(out, a)
	}
	// also last cleared events
	for _, a := range s.recent {
		if !a.Active {
			out = append(out, a)
			if len(out) > 40 {
				break
			}
		}
	}
	return out
}

func (s *alertStore) Active() []Alert {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Alert, 0, len(s.active))
	for _, a := range s.active {
		out = append(out, a)
	}
	return out
}

func evaluateAlerts() {
	c := GetConfig()
	snmp := snmpStore.Get()
	bgp := bgpStore.Get()
	stats := store.GetStatsLite()

	if snmp.OK && snmp.UplinkUtilPct >= c.AlertUtilPct {
		alerts.Raise("util_high", "Utilização de uplink alta",
			fmt.Sprintf("%.1f%% (limite %.0f%%)", snmp.UplinkUtilPct, c.AlertUtilPct), AlertWarning)
	} else {
		alerts.Clear("util_high")
	}

	if !snmp.OK {
		alerts.Raise("snmp_down", "SNMP indisponível", snmp.Error, AlertCritical)
	} else {
		alerts.Clear("snmp_down")
	}

	if bgp.OK {
		down := bgp.Total - bgp.Established
		if down > 0 && bgp.Established > 0 {
			alerts.Raise("bgp_partial", "Sessões BGP fora de established",
				fmt.Sprintf("%d/%d down ou não-established", down, bgp.Total), AlertWarning)
		} else {
			alerts.Clear("bgp_partial")
		}
		if bgp.Established == 0 && bgp.Total > 0 {
			alerts.Raise("bgp_all_down", "Nenhuma sessão BGP established", "", AlertCritical)
		} else {
			alerts.Clear("bgp_all_down")
		}
	}

	// pico de categoria: other > 90% com tráfego significativo
	if stats.Mbps > 1 && stats.ByCategoryMbps != nil {
		other := stats.ByCategoryMbps["other"]
		if other/stats.Mbps > 0.9 {
			alerts.Raise("class_low", "Baixa classificação NetFlow",
				fmt.Sprintf("%.0f%% other", other/stats.Mbps*100), AlertInfo)
		} else {
			alerts.Clear("class_low")
		}
	}

	evaluateASNAlerts(c, snmp)
}

func evaluateASNAlerts(c AppConfig, snmp SNMPSnapshot) {
	pctLim := c.AlertASNPct
	mbpsLim := c.AlertASNMbps
	if pctLim <= 0 && mbpsLim <= 0 {
		return
	}
	ignore := map[string]bool{}
	for _, a := range c.AlertASNIgnore {
		ignore[normalizeASNQuery(a)] = true
	}
	asnMbps := store.asnWindowScaledMbps()
	uplink := (snmp.UplinkInMbps + snmp.UplinkOutMbps) / 2
	if !snmp.OK {
		uplink = 0
	}
	active := map[string]bool{}
	for asn, mbps := range asnMbps {
		if ignore[normalizeASNQuery(asn)] {
			continue
		}
		hit := false
		detail := ""
		if mbpsLim > 0 && mbps >= mbpsLim {
			hit = true
			detail = fmt.Sprintf("%.0f Mbps (limite %.0f Mbps)", mbps, mbpsLim)
		}
		if pctLim > 0 && uplink > 50 {
			share := mbps / uplink * 100
			if share >= pctLim {
				hit = true
				detail = fmt.Sprintf("%.0f Mbps · %.1f%% do uplink (limite %.0f%%)", mbps, share, pctLim)
			}
		}
		code := "asn_high_" + asn
		if hit {
			active[code] = true
			name := asn
			if n := parseASNNum(asn); n > 0 {
				if dn := asnDisplayName(n); dn != "" {
					name = dn + " (" + asn + ")"
				}
			}
			alerts.Raise(code, "ASN alto: "+name, detail, AlertWarning)
		}
	}
	// Limpa alertas ASN que não estão mais ativos
	for _, a := range alerts.Active() {
		if strings.HasPrefix(a.Code, "asn_high_") && !active[a.Code] {
			alerts.Clear(a.Code)
		}
	}
}

func StartAlertEvaluator() {
	go func() {
		time.Sleep(15 * time.Second)
		for {
			evaluateAlerts()
			time.Sleep(10 * time.Second)
		}
	}()
}

// --- histórico persistente (JSONL) ---

func historyPath() string {
	return filepath.Join(GetConfig().DataDir, "history.jsonl")
}

func appendHistoryFile(pt HistoryPoint) {
	path := historyPath()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	b, _ := json.Marshal(pt)
	_, _ = f.Write(append(b, '\n'))
}

func loadHistoryFile(sinceTs int64) []HistoryPoint {
	b, err := os.ReadFile(historyPath())
	if err != nil {
		return nil
	}
	var out []HistoryPoint
	start := 0
	for i := 0; i <= len(b); i++ {
		if i == len(b) || b[i] == '\n' {
			line := b[start:i]
			start = i + 1
			if len(line) == 0 {
				continue
			}
			var pt HistoryPoint
			if json.Unmarshal(line, &pt) == nil {
				if sinceTs == 0 || pt.Ts >= sinceTs {
					out = append(out, pt)
				}
			}
		}
	}
	// keep last 2000
	if len(out) > 2000 {
		out = out[len(out)-2000:]
	}
	return out
}

func pruneHistoryFile() {
	c := GetConfig()
	retain := int64(c.HistoryLocalH) * 3600
	if retain <= 0 {
		retain = int64(c.HistoryRetainH) * 3600
	}
	if retain <= 0 {
		retain = 72 * 3600
	}
	cutoff := time.Now().Unix() - retain
	pts := loadHistoryFile(cutoff)
	path := historyPath()
	f, err := os.Create(path + ".tmp")
	if err != nil {
		return
	}
	for _, pt := range pts {
		b, _ := json.Marshal(pt)
		_, _ = f.Write(append(b, '\n'))
	}
	_ = f.Close()
	_ = os.Rename(path+".tmp", path)
}
