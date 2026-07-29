package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// SamplingEstimate compara NetFlow amostrado com SNMP (autoridade).
type SamplingEstimate struct {
	Configured  float64 `json:"configured"`
	Native      float64 `json:"native"`
	Estimated   float64 `json:"estimated"`
	Effective   float64 `json:"effective"`
	NetFlowMbps float64 `json:"netflow_mbps"`
	SNMPMbps    float64 `json:"snmp_mbps"`
	ScaledMbps  float64 `json:"scaled_mbps"`
	UpdatedAt   int64   `json:"updated_at"`
	Mode        string  `json:"mode"` // native|auto|fixed
	Source      string  `json:"source"`
}

type samplingStore struct {
	mu     sync.RWMutex
	snap   SamplingEstimate
	native float64
}

var sampling = &samplingStore{
	snap: SamplingEstimate{Mode: "auto", Effective: 1, Source: "default"},
}

func samplingNativePath() string {
	return filepath.Join(GetConfig().DataDir, "sampling_native.json")
}

func loadNativeSampling() {
	b, err := os.ReadFile(samplingNativePath())
	if err != nil {
		return
	}
	var f struct {
		Rate float64 `json:"rate"`
	}
	if json.Unmarshal(b, &f) != nil || f.Rate < 2 {
		return
	}
	sampling.mu.Lock()
	sampling.native = f.Rate
	sampling.mu.Unlock()
	log.Printf("sampling nativo restaurado: 1:%.0f", f.Rate)
}

func persistNativeSampling(rate float64) {
	_ = os.MkdirAll(GetConfig().DataDir, 0755)
	b, _ := json.Marshal(map[string]float64{"rate": rate})
	_ = os.WriteFile(samplingNativePath(), b, 0644)
}

func (s *samplingStore) SetNative(rate float64) {
	if rate < 2 || rate > 1e6 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	changed := s.native != rate
	if changed {
		log.Printf("sampling nativo NetFlow: 1:%.0f", rate)
	}
	s.native = rate
	if changed {
		go persistNativeSampling(rate)
	}
}

func (s *samplingStore) Get() SamplingEstimate {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snap
}

func (s *samplingStore) Update(nfMbps, snmpUplinkAvg float64) {
	c := GetConfig()
	s.mu.Lock()
	defer s.mu.Unlock()

	out := SamplingEstimate{
		Configured:  c.SamplingRate,
		Native:      s.native,
		NetFlowMbps: nfMbps,
		SNMPMbps:    snmpUplinkAvg,
		UpdatedAt:   time.Now().Unix(),
	}

	// Prioridade: config fixa > amostragem nativa do template NetFlow > ratio SNMP
	switch {
	case c.SamplingRate > 1:
		out.Mode = "fixed"
		out.Source = "config"
		out.Effective = c.SamplingRate
		out.Estimated = c.SamplingRate
	case s.native > 1:
		out.Mode = "native"
		out.Source = "netflow_options"
		out.Effective = s.native
		if nfMbps > 0.5 && snmpUplinkAvg > nfMbps*2 {
			est := snmpUplinkAvg / nfMbps
			prev := s.snap.Estimated
			if prev > 1 {
				est = prev*0.7 + est*0.3
			}
			out.Estimated = est
		} else {
			out.Estimated = s.native
		}
	default:
		out.Mode = "auto"
		out.Source = "snmp_ratio"
		if nfMbps > 0.5 && snmpUplinkAvg > nfMbps*2 {
			est := snmpUplinkAvg / nfMbps
			prev := s.snap.Estimated
			if prev > 1 {
				est = prev*0.7 + est*0.3
			}
			if est < 1 {
				est = 1
			}
			if est > 100000 {
				est = 100000
			}
			out.Estimated = est
			out.Effective = est
		} else if s.snap.Effective > 1 {
			out.Estimated = s.snap.Estimated
			out.Effective = s.snap.Effective
		} else {
			out.Estimated = 1
			out.Effective = 1
		}
	}
	out.ScaledMbps = nfMbps * out.Effective
	s.snap = out
}

func scaleMbps(mbps float64) float64 {
	e := sampling.Get().Effective
	if e <= 1 {
		return mbps
	}
	return mbps * e
}
