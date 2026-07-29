package main

import (
	"bytes"
	"encoding/json"
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

// Resolução ASN/nome via ip-api.com (gratuita, sem API key).
// Docs: http://ip-api.com/docs/api:batch
// Limite free: ~15 batch/min · até 100 IPs por batch.

type IPAPIInfo struct {
	IP     string `json:"ip"`
	ASN    uint32 `json:"asn"`
	ASNStr string `json:"asn_str"`
	ASName string `json:"asname"`
	Org    string `json:"org"`
	ISP    string `json:"isp"`
	Name   string `json:"name"` // nome amigável para UI
}

type ipAPIResp struct {
	Status string `json:"status"`
	Message string `json:"message,omitempty"`
	AS     string `json:"as"`
	ASName string `json:"asname"`
	ISP    string `json:"isp"`
	Org    string `json:"org"`
	Query  string `json:"query"`
}

type ipAPIStore struct {
	mu       sync.RWMutex
	byIP     map[string]*IPAPIInfo
	byASN    map[uint32]*IPAPIInfo
	queue    []string
	queued   map[string]bool
	inflight map[string]bool
	client   *http.Client
	lastSave time.Time
	lookups  int64
	hits     int64
	enabled  bool
}

var ipapi = &ipAPIStore{
	byIP:     make(map[string]*IPAPIInfo),
	byASN:    make(map[uint32]*IPAPIInfo),
	queued:   make(map[string]bool),
	inflight: make(map[string]bool),
	client:   &http.Client{Timeout: 8 * time.Second},
	enabled:  true,
}

func ipAPICachePath() string {
	return filepath.Join(GetConfig().DataDir, "ipapi_asn.json")
}

func loadIPAPICache() {
	b, err := os.ReadFile(ipAPICachePath())
	if err != nil {
		return
	}
	var list []IPAPIInfo
	if json.Unmarshal(b, &list) != nil {
		return
	}
	ipapi.mu.Lock()
	defer ipapi.mu.Unlock()
	for i := range list {
		info := list[i]
		if info.IP != "" {
			cp := info
			ipapi.byIP[info.IP] = &cp
		}
		if info.ASN > 0 {
			cp := info
			ipapi.byASN[info.ASN] = &cp
		}
	}
	log.Printf("ip-api: cache restaurado (%d IPs · %d ASNs)", len(ipapi.byIP), len(ipapi.byASN))
}

func persistIPAPICache() {
	ipapi.mu.RLock()
	list := make([]IPAPIInfo, 0, len(ipapi.byIP))
	seen := map[uint32]bool{}
	for _, info := range ipapi.byIP {
		if info == nil {
			continue
		}
		list = append(list, *info)
		if info.ASN > 0 {
			seen[info.ASN] = true
		}
	}
	for as, info := range ipapi.byASN {
		if seen[as] || info == nil {
			continue
		}
		list = append(list, *info)
	}
	ipapi.mu.RUnlock()
	if len(list) > 5000 {
		list = list[len(list)-5000:]
	}
	b, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return
	}
	tmp := ipAPICachePath() + ".tmp"
	if err := os.WriteFile(tmp, b, 0644); err != nil {
		return
	}
	_ = os.Rename(tmp, ipAPICachePath())
}

func shortOrgName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// Remove prefixo "AS12345 "
	if strings.HasPrefix(strings.ToUpper(s), "AS") {
		parts := strings.SplitN(s, " ", 2)
		if len(parts) == 2 {
			s = parts[1]
		}
	}
	s = strings.TrimSpace(s)
	// Encurta nomes longos de razão social
	lower := strings.ToLower(s)
	for _, cut := range []string{" ltda", " ltda.", " s.a.", " s/a", " sa", " me", " eireli", " inc.", " inc", " llc", " ltd."} {
		if idx := strings.Index(lower, cut); idx > 3 {
			s = strings.TrimSpace(s[:idx])
			break
		}
	}
	if len(s) > 40 {
		s = s[:40]
	}
	return s
}

func parseASField(asField string) (uint32, string) {
	asField = strings.TrimSpace(asField)
	if asField == "" {
		return 0, ""
	}
	// "AS15169 Google LLC"
	parts := strings.SplitN(asField, " ", 2)
	num := parseASNNum(parts[0])
	name := ""
	if len(parts) == 2 {
		name = shortOrgName(parts[1])
	}
	return num, name
}

func (s *ipAPIStore) LookupIP(ip string) *IPAPIInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if info, ok := s.byIP[ip]; ok {
		s.hits++
		return info
	}
	return nil
}

func (s *ipAPIStore) LookupASN(as uint32) *IPAPIInfo {
	if as == 0 {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if info, ok := s.byASN[as]; ok {
		return info
	}
	return nil
}

func (s *ipAPIStore) NameForASN(as uint32) string {
	if info := s.LookupASN(as); info != nil && info.Name != "" {
		return info.Name
	}
	return ""
}

func (s *ipAPIStore) ASNForIP(ip net.IP) (uint32, string) {
	if ip == nil || isPrivateOrSpecial(ip) {
		return 0, ""
	}
	info := s.LookupIP(ip.String())
	if info == nil || info.ASN == 0 {
		s.Enqueue(ip)
		return 0, ""
	}
	return info.ASN, info.Name
}

func (s *ipAPIStore) Enqueue(ip net.IP) {
	if !s.enabled || ip == nil || isPrivateOrSpecial(ip) {
		return
	}
	key := ip.String()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.byIP[key] != nil || s.queued[key] || s.inflight[key] {
		return
	}
	if len(s.queue) >= 2000 {
		return
	}
	s.queue = append(s.queue, key)
	s.queued[key] = true
}

func (s *ipAPIStore) storeResult(r ipAPIResp) {
	if r.Status != "success" || r.Query == "" {
		return
	}
	asNum, asOrg := parseASField(r.AS)
	name := shortOrgName(r.ASName)
	if name == "" {
		name = asOrg
	}
	if name == "" {
		name = shortOrgName(r.Org)
	}
	if name == "" {
		name = shortOrgName(r.ISP)
	}
	if name == "" && asNum > 0 {
		name = fmt.Sprintf("AS%d", asNum)
	}
	info := &IPAPIInfo{
		IP:     r.Query,
		ASN:    asNum,
		ASNStr: fmt.Sprintf("AS%d", asNum),
		ASName: r.ASName,
		Org:    r.Org,
		ISP:    r.ISP,
		Name:   name,
	}
	if asNum == 0 {
		info.ASNStr = ""
	}

	s.mu.Lock()
	s.byIP[r.Query] = info
	delete(s.inflight, r.Query)
	delete(s.queued, r.Query)
	if asNum > 0 {
		s.byASN[asNum] = info
	}
	s.lookups++
	s.mu.Unlock()
}

func (s *ipAPIStore) flushBatch() {
	s.mu.Lock()
	n := len(s.queue)
	if n == 0 {
		s.mu.Unlock()
		return
	}
	if n > 80 {
		n = 80
	}
	batch := append([]string(nil), s.queue[:n]...)
	s.queue = s.queue[n:]
	for _, ip := range batch {
		delete(s.queued, ip)
		s.inflight[ip] = true
	}
	s.mu.Unlock()

	body, _ := json.Marshal(batch)
	url := "http://ip-api.com/batch?fields=status,message,as,asname,isp,org,query"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		s.requeue(batch)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := s.client.Do(req)
	if err != nil {
		log.Printf("ip-api: %v", err)
		s.requeue(batch)
		return
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode == 429 {
		log.Printf("ip-api: rate limit — aguardando")
		s.requeue(batch)
		time.Sleep(60 * time.Second)
		return
	}
	if res.StatusCode != 200 {
		log.Printf("ip-api: HTTP %d", res.StatusCode)
		s.requeue(batch)
		return
	}
	var out []ipAPIResp
	if err := json.Unmarshal(raw, &out); err != nil {
		log.Printf("ip-api: decode: %v", err)
		s.requeue(batch)
		return
	}
	for _, r := range out {
		s.storeResult(r)
	}
	// Marca IPs sem resposta como "vistos" vazios para não reenfileirar forever
	s.mu.Lock()
	for _, ip := range batch {
		if s.byIP[ip] == nil {
			s.byIP[ip] = &IPAPIInfo{IP: ip}
		}
		delete(s.inflight, ip)
	}
	save := time.Since(s.lastSave) > 60*time.Second
	if save {
		s.lastSave = time.Now()
	}
	s.mu.Unlock()
	if save {
		go persistIPAPICache()
	}
}

func (s *ipAPIStore) requeue(batch []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, ip := range batch {
		delete(s.inflight, ip)
		if s.byIP[ip] != nil || s.queued[ip] {
			continue
		}
		s.queue = append(s.queue, ip)
		s.queued[ip] = true
	}
}

func (s *ipAPIStore) Stats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return map[string]interface{}{
		"enabled":  s.enabled,
		"cached_ips": len(s.byIP),
		"cached_asns": len(s.byASN),
		"queue":    len(s.queue),
		"lookups":  s.lookups,
		"hits":     s.hits,
		"source":   "ip-api.com",
	}
}

func StartIPAPIResolver() {
	loadIPAPICache()
	go func() {
		// ~12 batch/min para ficar sob o limite free (15/min)
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			ipapi.flushBatch()
		}
	}()
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			persistIPAPICache()
		}
	}()
	log.Printf("ip-api: resolvedor ASN iniciado (batch a cada 5s)")
}

func handleIPAPI(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, ipapi.Stats())
}
