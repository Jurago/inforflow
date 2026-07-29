package main

import (
	"fmt"
	"math"
	"sync"
	"time"
)

var (
	lastNetFlowAt   int64
	lastNetFlowMu   sync.RWMutex
	historyBaseline = map[string]float64{}
	baselineMu      sync.RWMutex
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

func evaluateAnomalies() {
	stats := store.GetStatsLite()
	snmp := snmpStore.Get()
	samp := sampling.Get()

	// NetFlow parou de chegar
	silent := netFlowSilentSec()
	if silent > 120 && snmp.OK && snmp.UplinkInMbps > 100 {
		alerts.Raise("netflow_silent", "NetFlow sem pacotes",
			fmt.Sprintf("sem flows há %ds com uplink ativo (%.0f Mbps)", silent, snmp.UplinkInMbps), AlertCritical)
	} else if silent <= 120 {
		alerts.Clear("netflow_silent")
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

	// Divergência NetFlow×SNMP > 50%
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
