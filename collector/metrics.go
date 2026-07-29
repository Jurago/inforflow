package main

import (
	"fmt"
	"net/http"
	"os"
	"sync/atomic"
	"time"
)

var (
	metricFlowsTotal uint64
	metricHTTPReqs   uint64
)

func incHTTPReq() { atomic.AddUint64(&metricHTTPReqs, 1) }

func handleMetrics(w http.ResponseWriter, r *http.Request) {
	incHTTPReq()
	snmp := snmpStore.Get()
	bgp := bgpStore.Get()
	samp := sampling.Get()
	stats := store.GetStatsLite()
	c := GetConfig()

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	fmt.Fprintf(w, "# HELP inforflow_uptime_seconds Uptime do coletor\n")
	fmt.Fprintf(w, "inforflow_uptime_seconds %d\n", int(timeSinceStart()))
	fmt.Fprintf(w, "inforflow_flows_total %d\n", atomic.LoadUint64(&store.seq))
	fmt.Fprintf(w, "inforflow_http_requests_total %d\n", atomic.LoadUint64(&metricHTTPReqs))
	fmt.Fprintf(w, "inforflow_netflow_mbps %.4f\n", stats.Mbps)
	fmt.Fprintf(w, "inforflow_netflow_mbps_scaled %.4f\n", samp.ScaledMbps)
	fmt.Fprintf(w, "inforflow_snmp_uplink_in_mbps %.4f\n", snmp.UplinkInMbps)
	fmt.Fprintf(w, "inforflow_snmp_uplink_out_mbps %.4f\n", snmp.UplinkOutMbps)
	fmt.Fprintf(w, "inforflow_snmp_uplink_util_pct %.2f\n", snmp.UplinkUtilPct)
	fmt.Fprintf(w, "inforflow_snmp_ok %d\n", boolMetric(snmp.OK))
	fmt.Fprintf(w, "inforflow_bgp_peers_established %d\n", bgp.Established)
	fmt.Fprintf(w, "inforflow_bgp_peers_total %d\n", bgp.Total)
	fmt.Fprintf(w, "inforflow_sampling_effective %.2f\n", samp.Effective)
	fmt.Fprintf(w, "inforflow_alerts_active %d\n", len(alerts.Active()))
	fmt.Fprintf(w, "inforflow_netflow_silent_seconds %d\n", netFlowSilentSec())
	fmt.Fprintf(w, "inforflow_s3_enabled %d\n", boolMetric(s3Enabled()))
	pts, dbBytes := storageLocalStats()
	fmt.Fprintf(w, "inforflow_history_points %d\n", pts)
	fmt.Fprintf(w, "inforflow_db_bytes %d\n", dbBytes)
	if fi, err := os.Stat(GetConfig().DataDir); err == nil {
		fmt.Fprintf(w, "inforflow_data_dir_exists 1\n")
		_ = fi
	}
	if stats.ByCategoryMbps != nil {
		for cat, mbps := range stats.ByCategoryMbps {
			fmt.Fprintf(w, "inforflow_category_mbps{category=%q} %.4f\n", cat, mbps)
			fmt.Fprintf(w, "inforflow_category_mbps_scaled{category=%q} %.4f\n", cat, mbps*samp.Effective)
		}
	}
	_ = c
}

func boolMetric(b bool) int {
	if b {
		return 1
	}
	return 0
}

func timeSinceStart() int64 {
	return int64(time.Since(store.started).Seconds())
}
