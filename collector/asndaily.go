package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Persistência leve dos top ASNs do dia — sobrevive a restart do coletor.

type asnDailyFile struct {
	Day   string           `json:"day"`
	Bytes map[string]int64 `json:"bytes"`
	Flows map[string]int64 `json:"flows"`
}

var (
	asnDailyMu   sync.Mutex
	asnDailyLast time.Time
)

func asnDailyPath() string {
	return filepath.Join(GetConfig().DataDir, "asn_daily.json")
}

func loadASNDaily() {
	b, err := os.ReadFile(asnDailyPath())
	if err != nil {
		return
	}
	var f asnDailyFile
	if json.Unmarshal(b, &f) != nil || f.Day == "" {
		return
	}
	today := time.Now().Format("2006-01-02")
	if f.Day != today {
		log.Printf("asn_daily: arquivo de %s ignorado (hoje %s)", f.Day, today)
		return
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	n := 0
	for k, bytes := range f.Bytes {
		as := parseASNNum(k)
		if as == 0 || bytes <= 0 {
			continue
		}
		store.dstASNBytes[as] += bytes
		if fl, ok := f.Flows[k]; ok {
			store.dstASNFlows[as] += fl
		}
		n++
	}
	log.Printf("asn_daily: restaurados %d ASNs do dia (%s)", n, f.Day)
}

func persistASNDaily() {
	asnDailyMu.Lock()
	defer asnDailyMu.Unlock()
	if time.Since(asnDailyLast) < 30*time.Second {
		return
	}
	asnDailyLast = time.Now()

	today := time.Now().Format("2006-01-02")
	type kv struct {
		as    uint32
		bytes int64
		flows int64
	}

	store.mu.RLock()
	ranked := make([]kv, 0, len(store.dstASNBytes))
	for as, b := range store.dstASNBytes {
		if as > 0 && b > 0 {
			ranked = append(ranked, kv{as, b, store.dstASNFlows[as]})
		}
	}
	store.mu.RUnlock()

	sort.Slice(ranked, func(i, j int) bool { return ranked[i].bytes > ranked[j].bytes })
	if len(ranked) > 50 {
		ranked = ranked[:50]
	}

	f := asnDailyFile{
		Day:   today,
		Bytes: make(map[string]int64, len(ranked)),
		Flows: make(map[string]int64, len(ranked)),
	}
	for _, r := range ranked {
		key := formatASN(r.as)
		f.Bytes[key] = r.bytes
		f.Flows[key] = r.flows
	}

	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return
	}
	tmp := asnDailyPath() + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return
	}
	_ = os.Rename(tmp, asnDailyPath())
}

func formatASN(as uint32) string {
	return "AS" + itoa(int(as))
}
