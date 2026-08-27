package main

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

type CacheHitSnapshot struct {
	OK              bool               `json:"ok"`
	CacheSNMPInMbps float64            `json:"cache_snmp_in_mbps"`
	CacheSNMPOutMbps float64           `json:"cache_snmp_out_mbps"`
	StreamingScaled float64            `json:"streaming_scaled_mbps"`
	CDNScaled       float64            `json:"cdn_scaled_mbps"`
	NetflixScaled   float64            `json:"netflix_scaled_mbps"`
	EstimatedHitPct float64            `json:"estimated_hit_pct"`
	Interfaces      []SNMPInterface    `json:"interfaces,omitempty"`
	Detail          string             `json:"detail,omitempty"`
}

func computeCacheHit() CacheHitSnapshot {
	snmp := snmpStore.Get()
	stats := store.GetStatsCached()
	samp := sampling.Get()
	eff := samp.Effective
	if eff < 1 {
		eff = 1
	}
	scaled := stats.ByCategoryMbpsScaled
	if scaled == nil {
		scaled = map[string]float64{}
		for k, v := range stats.ByCategoryMbps {
			scaled[k] = v * eff
		}
	}

	var cacheIn, cacheOut float64
	var ifaces []SNMPInterface
	for _, iface := range snmp.Interfaces {
		t := strings.ToUpper(iface.Alias + " " + iface.Name)
		if strings.Contains(t, "CACHE") || strings.Contains(t, "NETFLIX") ||
			strings.Contains(t, "GGC") || strings.Contains(t, "GOOGLE") {
			cacheIn += iface.InMbps
			cacheOut += iface.OutMbps
			ifaces = append(ifaces, iface)
		}
	}

	streaming := scaled["streaming"] + scaled["netflix"] + scaled["globo"] + scaled["apple"]
	cdn := scaled["cdn"]
	cacheTotal := cacheIn + cacheOut
	hitPct := 0.0
	if streaming > 0 && cacheTotal > 0 {
		hitPct = (cacheTotal / streaming) * 100
		if hitPct > 100 {
			hitPct = 100
		}
	}

	return CacheHitSnapshot{
		OK:               snmp.OK,
		CacheSNMPInMbps:  cacheIn,
		CacheSNMPOutMbps: cacheOut,
		StreamingScaled:  streaming,
		CDNScaled:        cdn,
		NetflixScaled:    scaled["netflix"],
		EstimatedHitPct:  hitPct,
		Interfaces:       ifaces,
		Detail:           "Cache SNMP vs streaming classificado (NetFlow×amostragem)",
	}
}

func handleCache(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, computeCacheHit())
}

func handleHistoryCompare(w http.ResponseWriter, r *http.Request) {
	hours := 24
	if v := r.URL.Query().Get("hours"); v != "" {
		if h, err := strconv.Atoi(v); err == nil && h > 0 && h <= 720 {
			hours = h
		}
	}
	now := time.Now().Unix()
	period := int64(hours) * 3600
	current := queryHistorySince(now - period)
	previous := queryHistorySince(now - period*2)
	var prevFiltered []HistoryPoint
	for _, p := range previous {
		if p.Ts >= now-period*2 && p.Ts < now-period {
			prevFiltered = append(prevFiltered, p)
		}
	}
	writeJSON(w, map[string]interface{}{
		"hours":    hours,
		"current":  current,
		"previous": prevFiltered,
	})
}

func handleStorage(w http.ResponseWriter, r *http.Request) {
	keys, _ := s3ListBackups("history/daily/")
	localPts, localSize := storageLocalStats()
	writeJSON(w, map[string]interface{}{
		"sqlite_path":       storageDBPath(),
		"s3_enabled":        s3Enabled(),
		"s3_bucket":         GetConfig().S3Bucket,
		"s3_status":         S3Status(),
		"local_hours":       GetConfig().HistoryLocalH,
		"s3_days":           GetConfig().HistoryS3Days,
		"local_points":      localPts,
		"local_db_bytes":    localSize,
		"s3_daily_archives": keys,
		"s3_backups":        func() []string { k, _ := s3ListBackups("backups/"); return k }(),
	})
}
