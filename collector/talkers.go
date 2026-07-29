package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// --- Top talkers (CGNAT / clientes) ---

type TalkerStats struct {
	IP         string             `json:"ip"`
	Bytes      int64              `json:"bytes"`
	Mbps       float64            `json:"mbps"`
	MbpsScaled float64            `json:"mbps_scaled"`
	Flows      int64              `json:"flows"`
	TopCat     string             `json:"top_category"`
	ByCategory map[string]int64   `json:"by_category"`
	ByCatMbps  map[string]float64 `json:"by_category_mbps,omitempty"`
}

type talkerSample struct {
	at   time.Time
	ip   string
	cat  string
	bytes int64
}

type talkerStore struct {
	mu      sync.Mutex
	totals  map[string]*TalkerStats
	window  []talkerSample
}

var talkers = &talkerStore{
	totals: make(map[string]*TalkerStats),
}

func isCGNATClient(ip net.IP) bool {
	ip4 := ip.To4()
	if ip4 == nil {
		return false
	}
	// 100.64.0.0/10
	return ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127
}

func (t *talkerStore) Add(ip string, cat string, bytes int64) {
	if ip == "" || bytes <= 0 {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	ts := t.totals[ip]
	if ts == nil {
		ts = &TalkerStats{IP: ip, ByCategory: make(map[string]int64)}
		t.totals[ip] = ts
	}
	ts.Bytes += bytes
	ts.Flows++
	ts.ByCategory[cat] += bytes
	now := time.Now()
	t.window = append(t.window, talkerSample{now, ip, cat, bytes})
	cutoff := now.Add(-10 * time.Second)
	i := 0
	for i < len(t.window) && t.window[i].at.Before(cutoff) {
		i++
	}
	if i > 0 {
		t.window = t.window[i:]
	}
}

func (t *talkerStore) Top(n int) []TalkerStats {
	t.mu.Lock()
	defer t.mu.Unlock()

	winBytes := map[string]int64{}
	winCat := map[string]map[string]int64{}
	for _, s := range t.window {
		winBytes[s.ip] += s.bytes
		if winCat[s.ip] == nil {
			winCat[s.ip] = map[string]int64{}
		}
		winCat[s.ip][s.cat] += s.bytes
	}

	out := make([]TalkerStats, 0, len(t.totals))
	for _, ts := range t.totals {
		cp := *ts
		cp.ByCategory = copyMap(ts.ByCategory)
		topCat, topV := "other", int64(0)
		for c, v := range cp.ByCategory {
			if v > topV {
				topCat, topV = c, v
			}
		}
		cp.TopCat = topCat
		cp.Mbps = float64(winBytes[ts.IP]) * 8 / 10.0 / 1e6
		cp.MbpsScaled = scaleMbps(cp.Mbps)
		cp.ByCatMbps = map[string]float64{}
		for c, v := range winCat[ts.IP] {
			cp.ByCatMbps[c] = float64(v) * 8 / 10.0 / 1e6
		}
		out = append(out, cp)
	}
	// sort by window mbps then total bytes
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].Mbps > out[i].Mbps || (out[j].Mbps == out[i].Mbps && out[j].Bytes > out[i].Bytes) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	if n > 0 && len(out) > n {
		out = out[:n]
	}
	return out
}

// --- Notificações externas ---

var (
	notifyMu   sync.Mutex
	lastNotify = map[string]time.Time{}
)

func notifyAlert(a Alert) {
	c := GetConfig()
	if c.WebhookURL == "" && c.TelegramToken == "" {
		return
	}
	notifyMu.Lock()
	if t, ok := lastNotify[a.Code]; ok && time.Since(t) < 5*time.Minute {
		notifyMu.Unlock()
		return
	}
	lastNotify[a.Code] = time.Now()
	notifyMu.Unlock()

	msg := fmt.Sprintf("[Inforflow %s] %s\n%s", strings.ToUpper(string(a.Severity)), a.Title, a.Detail)
	go sendWebhook(c.WebhookURL, a, msg)
	go sendTelegram(c.TelegramToken, c.TelegramChat, msg)
}

func sendWebhook(hook string, a Alert, text string) {
	if hook == "" {
		return
	}
	body, _ := json.Marshal(map[string]interface{}{
		"text": text, "alert": a, "source": "inforflow",
	})
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Post(hook, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("webhook: %v", err)
		return
	}
	resp.Body.Close()
}

func sendTelegram(token, chat, text string) {
	if token == "" || chat == "" {
		return
	}
	u := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	form := url.Values{}
	form.Set("chat_id", chat)
	form.Set("text", text)
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.PostForm(u, form)
	if err != nil {
		log.Printf("telegram: %v", err)
		return
	}
	resp.Body.Close()
}
