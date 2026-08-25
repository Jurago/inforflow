package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Feeds dinâmicos de prefixos CDN — complementam classify.go.
type feedRule struct {
	net      *net.IPNet
	name     string
	category FlowCategory
	icon     string
	asn      string
}

var (
	feedMu    sync.RWMutex
	feedRules []feedRule
	feedStat  FeedsStatus
)

func getFeedsStatus() *FeedsStatus {
	feedMu.RLock()
	defer feedMu.RUnlock()
	cp := feedStat
	cp.Sources = append([]FeedSourceStatus(nil), feedStat.Sources...)
	if cp.UpdatedAt > 0 {
		now := time.Now().Unix()
		for i := range cp.Sources {
			if cp.Sources[i].UpdatedAt > 0 {
				cp.Sources[i].AgeSec = now - cp.Sources[i].UpdatedAt
			}
		}
	}
	return &cp
}

func feedClassify(ip net.IP) Classified {
	if ip == nil {
		return Classified{}
	}
	feedMu.RLock()
	defer feedMu.RUnlock()
	check := ip
	if ip4 := ip.To4(); ip4 != nil {
		check = ip4
	}
	for _, r := range feedRules {
		if r.net.Contains(check) {
			return Classified{Name: r.name, Category: r.category, Icon: r.icon, ASN: r.asn}
		}
	}
	return Classified{}
}

func StartPrefixFeeds() {
	go func() {
		refreshFeeds()
		for {
			mins := GetConfig().FeedIntervalM
			if mins < 30 {
				mins = 30
			}
			time.Sleep(time.Duration(mins) * time.Minute)
			refreshFeeds()
		}
	}()
}

func refreshFeeds() {
	dir := filepath.Join(GetConfig().DataDir, "feeds")
	_ = os.MkdirAll(dir, 0755)

	type src struct {
		url, name, icon, asn string
		cat                  FlowCategory
		file                 string
	}
	sources := []src{
		{"https://www.cloudflare.com/ips-v4", "Cloudflare", "cloudflare", "AS13335", CategoryCDN, "cloudflare-v4.txt"},
		{"https://www.cloudflare.com/ips-v6", "Cloudflare", "cloudflare", "AS13335", CategoryCDN, "cloudflare-v6.txt"},
	}

	var rules []feedRule
	client := &http.Client{Timeout: 20 * time.Second}
	sourcesStatus := []FeedSourceStatus{}
	now := time.Now().Unix()

	for _, s := range sources {
		path := filepath.Join(dir, s.file)
		fromCache := false
		ok := true
		body, err := fetchURL(client, s.url)
		if err != nil {
			// usa cache em disco
			if b, e := os.ReadFile(path); e == nil {
				body = b
				fromCache = true
				log.Printf("feed %s: usando cache local (%v)", s.name, err)
			} else {
				log.Printf("feed %s: falhou (%v)", s.name, err)
				sourcesStatus = append(sourcesStatus, FeedSourceStatus{
					Name: s.name, File: s.file, LastOK: false,
				})
				continue
			}
		} else {
			_ = os.WriteFile(path, body, 0644)
		}
		n := 0
		sc := bufio.NewScanner(strings.NewReader(string(body)))
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			_, network, err := net.ParseCIDR(line)
			if err != nil {
				// às vezes vem IP solo
				if ip := net.ParseIP(line); ip != nil {
					if ip.To4() != nil {
						_, network, err = net.ParseCIDR(line + "/32")
					} else {
						_, network, err = net.ParseCIDR(line + "/128")
					}
				}
			}
			if err != nil || network == nil {
				continue
			}
			rules = append(rules, feedRule{network, s.name, s.cat, s.icon, s.asn})
			n++
		}
		log.Printf("feed %s: %d prefixos", s.name, n)
		sourcesStatus = append(sourcesStatus, FeedSourceStatus{
			Name: s.name, File: s.file, Prefixes: n, LastOK: ok,
			FromCache: fromCache, UpdatedAt: now,
		})
	}

	// Prefixos locais extras (BR / caches) em data/feeds/extra.txt
	extra := filepath.Join(dir, "extra.txt")
	extraN := 0
	if b, err := os.ReadFile(extra); err == nil {
		sc := bufio.NewScanner(strings.NewReader(string(b)))
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			// formato: CIDR|Nome|categoria|icon|ASN
			parts := strings.Split(line, "|")
			if len(parts) < 2 {
				continue
			}
			_, network, err := net.ParseCIDR(strings.TrimSpace(parts[0]))
			if err != nil {
				continue
			}
			name := strings.TrimSpace(parts[1])
			cat := CategoryCDN
			icon := "cdn"
			asn := ""
			if len(parts) > 2 {
				cat = FlowCategory(strings.TrimSpace(parts[2]))
			}
			if len(parts) > 3 {
				icon = strings.TrimSpace(parts[3])
			}
			if len(parts) > 4 {
				asn = strings.TrimSpace(parts[4])
			}
			rules = append(rules, feedRule{network, name, cat, icon, asn})
			extraN++
		}
	}
	if extraN >= 0 {
		sourcesStatus = append(sourcesStatus, FeedSourceStatus{
			Name: "extra.txt", File: "extra.txt", Prefixes: extraN,
			LastOK: true, UpdatedAt: now,
		})
	}

	feedMu.Lock()
	feedRules = rules
	feedStat = FeedsStatus{
		TotalRules: len(rules),
		UpdatedAt:  now,
		Sources:    sourcesStatus,
	}
	feedMu.Unlock()
	log.Printf("feeds: %d regras ativas", len(rules))
}

func fetchURL(client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "inforflow-collector/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 2<<20))
}
