package main

import (
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

var (
	lastNetFlowAt   int64
	lastNetFlowMu   sync.RWMutex
	historyBaseline = map[string]float64{}
	baselineMu      sync.RWMutex
	udpQueueHighFor int64 // consecutive evaluations with high queue
)

func markNetFlowReceived() {
	lastNetFlowMu.Lock()
	lastNetFlowAt = time.Now().Unix()
	lastNetFlowMu.Unlock()
}

func netFlowSilentSec() int64 {
	lastNetFlowMu.RLock()
	defer lastNetFlowMu.RUnlock()
	if lastNetFlowAt == 0 {
		return 0
	}
	return time.Now().Unix() - lastNetFlowAt
}

func gapPct(nfScaled, snmpAvg float64) float64 {
	if snmpAvg <= 0 {
		return 0
	}
	return math.Abs(nfScaled-snmpAvg) / snmpAvg * 100
}

func evaluateAnomalies() {
	stats := store.GetStatsLite()
	snmp := snmpStore.Get()
	samp := sampling.Get()
	cfg := GetConfig()

	silentLimit := int64(cfg.AlertSilentSec)
	if silentLimit <= 0 {
		silentLimit = 90
	}

	// NetFlow parou de chegar
	silent := netFlowSilentSec()
	if silent > silentLimit && snmp.OK && snmp.UplinkInMbps > 50 {
		alerts.Raise("netflow_silent", "NetFlow sem pacotes",
			fmt.Sprintf("sem flows há %ds com uplink ativo (%.0f Mbps)", silent, snmp.UplinkInMbps), AlertCritical)
	} else if silent <= silentLimit {
		alerts.Clear("netflow_silent")
	}

	// Fila UDP do kernel crescente / alta
	q := atomic.LoadInt64(&netflowUDPQueue)
	qLim := cfg.AlertUDPQueue
	if qLim > 0 && q >= qLim {
		atomic.AddInt64(&udpQueueHighFor, 1)
		if atomic.LoadInt64(&udpQueueHighFor) >= 2 {
			alerts.Raise("udp_queue_high", "Fila UDP NetFlow alta",
				fmt.Sprintf("fila kernel ~%d bytes (limite %d) · drops=%d",
					q, qLim, atomic.LoadUint64(&netflowKernelDrops)), AlertWarning)
		}
	} else {
		atomic.StoreInt64(&udpQueueHighFor, 0)
		alerts.Clear("udp_queue_high")
	}

	// Gap NetFlow×SNMP persistente
	snmpAvg := (snmp.UplinkInMbps + snmp.UplinkOutMbps) / 2
	if snmp.OK && samp.ScaledMbps > 100 && snmpAvg > 100 && cfg.AlertGapPct > 0 {
		gp := gapPct(samp.ScaledMbps, snmpAvg)
		if gp >= cfg.AlertGapPct {
			alerts.Raise("nf_snmp_gap", "Gap NetFlow × SNMP",
				fmt.Sprintf("NF×fator=%.0f vs SNMP=%.0f · gap %.0f%% (limite %.0f%%)",
					samp.ScaledMbps, snmpAvg, gp, cfg.AlertGapPct), AlertWarning)
		} else {
			alerts.Clear("nf_snmp_gap")
		}
	}

	// Spike de categoria (>3x baseline ou >2 Gbps scaled)
	if stats.ByCategoryMbps != nil && samp.Effective > 1 {
		baselineMu.Lock()
		for cat, mbps := range stats.ByCategoryMbps {
			scaled := mbps * samp.Effective
			base := historyBaseline[cat]
			if base <= 0 {
				historyBaseline[cat] = scaled
				continue
			}
			if scaled > math.Max(base*3, 2000) && scaled > 500 {
				code := "spike_" + cat
				alerts.Raise(code, "Pico de tráfego: "+cat,
					fmt.Sprintf("%.0f Mbps (baseline ~%.0f Mbps)", scaled, base), AlertWarning)
			} else if scaled < base*1.5 {
				alerts.Clear("spike_" + cat)
			}
			historyBaseline[cat] = base*0.95 + scaled*0.05
		}
		baselineMu.Unlock()
	}

	// Divergência amostragem (ratio extremo)
	if snmp.OK && samp.ScaledMbps > 100 && samp.SNMPMbps > 100 {
		ratio := samp.ScaledMbps / samp.SNMPMbps
		if ratio < 0.5 || ratio > 2.0 {
			alerts.Raise("sampling_drift", "Divergência amostragem",
				fmt.Sprintf("NF×fator=%.0f vs SNMP=%.0f (ratio %.2f)", samp.ScaledMbps, samp.SNMPMbps, ratio), AlertInfo)
		} else {
			alerts.Clear("sampling_drift")
		}
	}
}

func StartAnomalyEvaluator() {
	go func() {
		time.Sleep(30 * time.Second)
		for {
			evaluateAnomalies()
			time.Sleep(15 * time.Second)
		}
	}()
}
