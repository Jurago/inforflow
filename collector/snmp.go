package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"sort"
	"sync"
	"time"
)

var (
	SNMPHost      = "170.245.127.191"
	SNMPPort      = 15161
	SNMPCommunity = "infornetV2"
	SNMPInterval  = 5 * time.Second
)

type SNMPInterface struct {
	Index       int     `json:"index"`
	Name        string  `json:"name"`
	Alias       string  `json:"alias"`
	OperStatus  int     `json:"oper_status"`
	AdminStatus int     `json:"admin_status"`
	SpeedMbps   uint64  `json:"speed_mbps"`
	InOctets    uint64  `json:"in_octets"`
	OutOctets   uint64  `json:"out_octets"`
	InMbps      float64 `json:"in_mbps"`
	OutMbps     float64 `json:"out_mbps"`
	InUtilPct   float64 `json:"in_util_pct"`
	OutUtilPct  float64 `json:"out_util_pct"`
	Role        string  `json:"role"`
}

type SNMPSnapshot struct {
	OK            bool            `json:"ok"`
	Host          string          `json:"host"`
	Port          int             `json:"port"`
	SysName       string          `json:"sys_name"`
	SysDescr      string          `json:"sys_descr"`
	UptimeTicks   uint64          `json:"uptime_ticks"`
	UptimeHuman   string          `json:"uptime_human"`
	CPUPct        float64         `json:"cpu_pct"`
	MemPct        float64         `json:"mem_pct"`
	UpdatedAt     int64           `json:"updated_at"`
	PollMs        int64           `json:"poll_ms"`
	TotalInMbps   float64         `json:"total_in_mbps"`
	TotalOutMbps  float64         `json:"total_out_mbps"`
	UplinkInMbps  float64         `json:"uplink_in_mbps"`
	UplinkOutMbps float64         `json:"uplink_out_mbps"`
	UplinkUtilPct float64         `json:"uplink_util_pct"`
	Deduped       bool            `json:"deduped"`
	Interfaces    []SNMPInterface `json:"interfaces"`
	TopIn         []SNMPInterface `json:"top_in"`
	TopOut        []SNMPInterface `json:"top_out"`
	Error         string          `json:"error,omitempty"`
}

type snmpPrevCounters struct {
	in, out uint64
	at      time.Time
}

type SNMPStore struct {
	mu      sync.RWMutex
	snap    SNMPSnapshot
	prev    map[int]snmpPrevCounters
	ifaceH  []IfaceHistoryPoint // ring ~2h @ 5s ≈ 1440
}

type IfaceSample struct {
	InMbps  float64 `json:"in_mbps"`
	OutMbps float64 `json:"out_mbps"`
}

type IfaceHistoryPoint struct {
	Ts      int64                  `json:"ts"`
	ByIndex map[int]IfaceSample    `json:"by_index,omitempty"`
	ByRole  map[string]float64     `json:"by_role,omitempty"`
}

func (s *SNMPStore) pushIfaceHistory(snap SNMPSnapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()
	pt := IfaceHistoryPoint{
		Ts:      time.Now().Unix(),
		ByIndex: map[int]IfaceSample{},
		ByRole:  map[string]float64{},
	}
	kept := 0
	for _, iface := range snap.Interfaces {
		if iface.OperStatus != 1 {
			continue
		}
		total := iface.InMbps + iface.OutMbps
		if iface.Role != "" {
			pt.ByRole[iface.Role] += total
		}
		// guarda ifaces relevantes (role crítica ou >1 Mbps)
		if iface.Role == "uplink" || iface.Role == "ix" || iface.Role == "cache" ||
			iface.Role == "bras" || iface.Role == "cgnat" || total > 1 {
			pt.ByIndex[iface.Index] = IfaceSample{InMbps: iface.InMbps, OutMbps: iface.OutMbps}
			kept++
			if kept >= 40 {
				// continua roles, para índices
			}
		}
	}
	// limitar ByIndex a 40 maiores
	if len(pt.ByIndex) > 40 {
		type kv struct {
			idx int
			v   float64
		}
		arr := make([]kv, 0, len(pt.ByIndex))
		for i, sm := range pt.ByIndex {
			arr = append(arr, kv{i, sm.InMbps + sm.OutMbps})
		}
		sort.Slice(arr, func(a, b int) bool { return arr[a].v > arr[b].v })
		keep := map[int]IfaceSample{}
		for i := 0; i < 40 && i < len(arr); i++ {
			keep[arr[i].idx] = pt.ByIndex[arr[i].idx]
		}
		pt.ByIndex = keep
	}
	s.ifaceH = append(s.ifaceH, pt)
	const maxH = 1440 // ~2h @ 5s
	if len(s.ifaceH) > maxH {
		s.ifaceH = s.ifaceH[len(s.ifaceH)-maxH:]
	}
}

func (s *SNMPStore) GetIfaceHistory(since int64, ifIndex int) []struct {
	Ts      int64   `json:"ts"`
	InMbps  float64 `json:"in_mbps"`
	OutMbps float64 `json:"out_mbps"`
} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]struct {
		Ts      int64   `json:"ts"`
		InMbps  float64 `json:"in_mbps"`
		OutMbps float64 `json:"out_mbps"`
	}, 0, len(s.ifaceH))
	for _, p := range s.ifaceH {
		if p.Ts < since {
			continue
		}
		sm, ok := p.ByIndex[ifIndex]
		if !ok {
			out = append(out, struct {
				Ts      int64   `json:"ts"`
				InMbps  float64 `json:"in_mbps"`
				OutMbps float64 `json:"out_mbps"`
			}{Ts: p.Ts})
			continue
		}
		out = append(out, struct {
			Ts      int64   `json:"ts"`
			InMbps  float64 `json:"in_mbps"`
			OutMbps float64 `json:"out_mbps"`
		}{Ts: p.Ts, InMbps: sm.InMbps, OutMbps: sm.OutMbps})
	}
	return out
}

var snmpStore = &SNMPStore{
	prev: make(map[int]snmpPrevCounters),
	snap: SNMPSnapshot{Host: SNMPHost, Port: SNMPPort},
}

func (s *SNMPStore) Get() SNMPSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := s.snap
	out.Interfaces = append([]SNMPInterface(nil), s.snap.Interfaces...)
	out.TopIn = append([]SNMPInterface(nil), s.snap.TopIn...)
	out.TopOut = append([]SNMPInterface(nil), s.snap.TopOut...)
	return out
}

// --- minimal SNMPv2c ---

type asn1Node struct {
	tag  byte
	val  []byte
	kids []asn1Node
}

func berLen(n int) []byte {
	if n < 0x80 {
		return []byte{byte(n)}
	}
	if n < 0x100 {
		return []byte{0x81, byte(n)}
	}
	return []byte{0x82, byte(n >> 8), byte(n)}
}

func berEncode(tag byte, body []byte) []byte {
	out := []byte{tag}
	out = append(out, berLen(len(body))...)
	return append(out, body...)
}

func berInt(n int) []byte {
	var b []byte
	if n == 0 {
		b = []byte{0}
	} else {
		neg := n < 0
		x := n
		if neg {
			x = -n
		}
		for x > 0 {
			b = append([]byte{byte(x & 0xff)}, b...)
			x >>= 8
		}
		if b[0]&0x80 != 0 {
			b = append([]byte{0}, b...)
		}
		if neg {
			// two's complement rough for positive path only used
		}
	}
	return berEncode(0x02, b)
}

func berOctet(s string) []byte {
	return berEncode(0x04, []byte(s))
}

func berOID(oid string) []byte {
	parts := splitOID(oid)
	if len(parts) < 2 {
		return berEncode(0x06, nil)
	}
	body := []byte{byte(40*parts[0] + parts[1])}
	for _, p := range parts[2:] {
		if p < 128 {
			body = append(body, byte(p))
			continue
		}
		var stack []byte
		for p > 0 {
			stack = append(stack, byte(p&0x7f))
			p >>= 7
		}
		for i := len(stack) - 1; i >= 0; i-- {
			b := stack[i]
			if i > 0 {
				b |= 0x80
			}
			body = append(body, b)
		}
	}
	return berEncode(0x06, body)
}

func berSeq(tag byte, items ...[]byte) []byte {
	var body []byte
	for _, it := range items {
		body = append(body, it...)
	}
	return berEncode(tag, body)
}

func splitOID(oid string) []int {
	oid = trimDot(oid)
	var parts []int
	cur := 0
	have := false
	for i := 0; i < len(oid); i++ {
		c := oid[i]
		if c == '.' {
			parts = append(parts, cur)
			cur = 0
			have = false
			continue
		}
		cur = cur*10 + int(c-'0')
		have = true
	}
	if have {
		parts = append(parts, cur)
	}
	return parts
}

func trimDot(s string) string {
	for len(s) > 0 && s[0] == '.' {
		s = s[1:]
	}
	return s
}

func parseBER(data []byte, i int) (tag byte, val []byte, next int, err error) {
	if i >= len(data) {
		return 0, nil, i, fmt.Errorf("eof")
	}
	tag = data[i]
	i++
	if i >= len(data) {
		return 0, nil, i, fmt.Errorf("eof len")
	}
	first := data[i]
	i++
	var ln int
	if first < 0x80 {
		ln = int(first)
	} else {
		n := int(first & 0x7f)
		if i+n > len(data) {
			return 0, nil, i, fmt.Errorf("bad len")
		}
		for j := 0; j < n; j++ {
			ln = (ln << 8) | int(data[i])
			i++
		}
	}
	if i+ln > len(data) {
		return 0, nil, i, fmt.Errorf("truncated")
	}
	return tag, data[i : i+ln], i + ln, nil
}

func parseOID(val []byte) string {
	if len(val) == 0 {
		return ""
	}
	oid := []int{int(val[0] / 40), int(val[0] % 40)}
	j := 1
	for j < len(val) {
		n := 0
		for {
			b := val[j]
			j++
			n = (n << 7) | int(b&0x7f)
			if b&0x80 == 0 {
				break
			}
			if j >= len(val) {
				break
			}
		}
		oid = append(oid, n)
	}
	s := ""
	for i, p := range oid {
		if i > 0 {
			s += "."
		}
		s += fmt.Sprintf("%d", p)
	}
	return s
}

func decodeSNMPValue(tag byte, val []byte) interface{} {
	switch tag {
	case 0x02: // INTEGER
		n := int64(0)
		for _, b := range val {
			n = (n << 8) | int64(b)
		}
		if len(val) > 0 && val[0]&0x80 != 0 {
			// negative
			shift := uint(8 * (8 - len(val)))
			n = int64(int64(n) << shift >> shift)
		}
		return n
	case 0x04:
		return string(val)
	case 0x06:
		return parseOID(val)
	case 0x40: // IpAddress
		if len(val) == 4 {
			return fmt.Sprintf("%d.%d.%d.%d", val[0], val[1], val[2], val[3])
		}
		return fmt.Sprintf("%x", val)
	case 0x41, 0x42, 0x43: // Counter32, Gauge32, TimeTicks
		var n uint64
		for _, b := range val {
			n = (n << 8) | uint64(b)
		}
		return n
	case 0x46: // Counter64
		if len(val) <= 8 {
			var n uint64
			for _, b := range val {
				n = (n << 8) | uint64(b)
			}
			return n
		}
		return binary.BigEndian.Uint64(val[len(val)-8:])
	case 0x05:
		return nil
	default:
		return val
	}
}

func snmpRequest(pduType byte, oid string, reqID int) (string, interface{}, error) {
	varbind := berSeq(0x30, berOID(oid), berEncode(0x05, nil))
	pdu := berSeq(pduType, berInt(reqID), berInt(0), berInt(0), berSeq(0x30, varbind))
	msg := berSeq(0x30, berInt(1), berOctet(SNMPCommunity), pdu)

	addr := &net.UDPAddr{IP: net.ParseIP(SNMPHost), Port: SNMPPort}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return "", nil, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	if _, err := conn.Write(msg); err != nil {
		return "", nil, err
	}

	buf := make([]byte, 65535)
	// Accept matching request-id; ignore stale UDP replies
	for attempt := 0; attempt < 5; attempt++ {
		n, err := conn.Read(buf)
		if err != nil {
			return "", nil, err
		}
		data := buf[:n]
		_, body, _, err := parseBER(data, 0)
		if err != nil {
			continue
		}
		i := 0
		for i < len(body) {
			tag, val, next, err := parseBER(body, i)
			if err != nil {
				break
			}
			i = next
			if tag != 0xa2 && tag != pduType {
				continue
			}
			k := 0
			tReq, vReq, k, err := parseBER(val, k)
			if err != nil || tReq != 0x02 {
				break
			}
			gotID := int(asUint64(decodeSNMPValue(0x02, vReq)))
			// signed int decode via asUint64 path — re-decode properly
			gotID = berIntValue(vReq)
			if gotID != reqID {
				continue // stale
			}
			_, _, k, err = parseBER(val, k) // error-status
			if err != nil {
				break
			}
			_, _, k, err = parseBER(val, k) // error-index
			if err != nil {
				break
			}
			_, vbl, _, err := parseBER(val, k)
			if err != nil {
				break
			}
			_, vb, _, err := parseBER(vbl, 0)
			if err != nil {
				break
			}
			m := 0
			tOID, vOID, m, err := parseBER(vb, m)
			if err != nil || tOID != 0x06 {
				break
			}
			tVal, vVal, _, err := parseBER(vb, m)
			if err != nil {
				break
			}
			return parseOID(vOID), decodeSNMPValue(tVal, vVal), nil
		}
	}
	return "", nil, fmt.Errorf("no matching snmp response")
}

func berIntValue(val []byte) int {
	n := 0
	for _, b := range val {
		n = (n << 8) | int(b)
	}
	return n
}

func snmpGet(oid string) (interface{}, error) {
	_, v, err := snmpRequest(0xa0, oid, 1)
	return v, err
}

func snmpWalk(base string, limit int) map[string]interface{} {
	base = trimDot(base)
	out := make(map[string]interface{})
	oid := base
	for i := 0; i < limit; i++ {
		noid, val, err := snmpRequest(0xa1, oid, i+1)
		if err != nil || noid == "" {
			break
		}
		noid = trimDot(noid)
		if noid == oid || len(noid) < len(base) || noid[:len(base)] != base {
			break
		}
		// ensure next char is '.' or end
		if len(noid) > len(base) && noid[len(base)] != '.' {
			break
		}
		out[noid] = val
		oid = noid
	}
	return out
}

func asUint64(v interface{}) uint64 {
	switch t := v.(type) {
	case uint64:
		return t
	case int64:
		if t < 0 {
			return 0
		}
		return uint64(t)
	case int:
		if t < 0 {
			return 0
		}
		return uint64(t)
	default:
		return 0
	}
}

func asString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	default:
		return fmt.Sprintf("%v", t)
	}
}

func ifaceRole(alias, name string) string {
	a := alias + " " + name
	upper := toUpper(a)
	switch {
	case containsAny(upper, "BRAS", "ENLACE BRAS"):
		return "bras"
	case containsAny(upper, "PTT", "IX"):
		return "ix"
	case containsAny(upper, "CGNAT"):
		return "cgnat"
	case containsAny(upper, "GOOGLE", "CACHE", "CDN"):
		return "cache"
	case containsAny(upper, "NORTH", "ONEX", "UPLINK", "ENLACE"):
		return "uplink"
	case containsAny(upper, "VPN"):
		return "vpn"
	default:
		if containsAny(name, "100GE", "Eth-Trunk") {
			return "core"
		}
		return "access"
	}
}

func toUpper(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'a' && c <= 'z' {
			c -= 32
		}
		b[i] = c
	}
	return string(b)
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if len(sub) == 0 {
			continue
		}
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
	}
	return false
}

func counterDelta(prev, cur uint64) uint64 {
	if cur >= prev {
		return cur - prev
	}
	// wrap (unlikely for Counter64 in our poll window)
	return cur
}

func pollSNMPOnce() {
	start := time.Now()
	snap := SNMPSnapshot{
		Host: SNMPHost,
		Port: SNMPPort,
	}

	if v, err := snmpGet("1.3.6.1.2.1.1.5.0"); err == nil {
		snap.SysName = asString(v)
	} else {
		snap.Error = err.Error()
		snap.OK = false
		snmpStore.mu.Lock()
		snmpStore.snap = snap
		snmpStore.mu.Unlock()
		return
	}
	if v, err := snmpGet("1.3.6.1.2.1.1.1.0"); err == nil {
		snap.SysDescr = asString(v)
	}
	if v, err := snmpGet("1.3.6.1.2.1.1.3.0"); err == nil {
		snap.UptimeTicks = asUint64(v)
		secs := snap.UptimeTicks / 100
		d := secs / 86400
		h := (secs % 86400) / 3600
		m := (secs % 3600) / 60
		snap.UptimeHuman = fmt.Sprintf("%dd %dh %dm", d, h, m)
	}

	// CPU / Mem Huawei
	cpuWalk := snmpWalk("1.3.6.1.4.1.2011.5.25.31.1.1.1.1.5", 20)
	var cpuSum float64
	var cpuN int
	for _, v := range cpuWalk {
		n := float64(asUint64(v))
		if n > 0 {
			cpuSum += n
			cpuN++
		}
	}
	if cpuN > 0 {
		snap.CPUPct = cpuSum / float64(cpuN)
	}
	memWalk := snmpWalk("1.3.6.1.4.1.2011.5.25.31.1.1.1.1.7", 20)
	var memSum float64
	var memN int
	for _, v := range memWalk {
		n := float64(asUint64(v))
		if n > 0 {
			memSum += n
			memN++
		}
	}
	if memN > 0 {
		snap.MemPct = memSum / float64(memN)
	}

	descr := snmpWalk("1.3.6.1.2.1.2.2.1.2", 300)
	alias := snmpWalk("1.3.6.1.2.1.31.1.1.1.18", 300)
	oper := snmpWalk("1.3.6.1.2.1.2.2.1.8", 300)
	admin := snmpWalk("1.3.6.1.2.1.2.2.1.7", 300)
	hspeed := snmpWalk("1.3.6.1.2.1.31.1.1.1.15", 300)

	type meta struct {
		idx, op, ad int
		name, al    string
		spd         uint64
		wantCounter bool
	}
	metas := make([]meta, 0, len(descr))
	for oid, v := range descr {
		idx := oidIndex(oid)
		if idx <= 0 {
			continue
		}
		name := asString(v)
		al := asString(alias[fmt.Sprintf("1.3.6.1.2.1.31.1.1.1.18.%d", idx)])
		op := int(asUint64(oper[fmt.Sprintf("1.3.6.1.2.1.2.2.1.8.%d", idx)]))
		ad := int(asUint64(admin[fmt.Sprintf("1.3.6.1.2.1.2.2.1.7.%d", idx)]))
		spd := asUint64(hspeed[fmt.Sprintf("1.3.6.1.2.1.31.1.1.1.15.%d", idx)])
		role := ifaceRole(al, name)
		want := op == 1 && (al != "" || role == "bras" || role == "uplink" || role == "ix" ||
			role == "cache" || role == "cgnat" || role == "core" || containsAny(name, "100GE", "Eth-Trunk", "Gigabit"))
		metas = append(metas, meta{idx, op, ad, name, al, spd, want})
	}

	req := int(time.Now().UnixNano() & 0x0fffffff)
	counters := make(map[int][2]uint64, len(metas))
	for _, m := range metas {
		if !m.wantCounter {
			continue
		}
		req++
		var inO, outO uint64
		if _, iv, err := snmpRequest(0xa0, fmt.Sprintf("1.3.6.1.2.1.31.1.1.1.6.%d", m.idx), req); err == nil {
			inO = asUint64(iv)
		}
		req++
		if _, ov, err := snmpRequest(0xa0, fmt.Sprintf("1.3.6.1.2.1.31.1.1.1.10.%d", m.idx), req); err == nil {
			outO = asUint64(ov)
		}
		counters[m.idx] = [2]uint64{inO, outO}
	}

	now := time.Now() // after counter collection
	ifaces := make([]SNMPInterface, 0, len(metas))

	snmpStore.mu.Lock()
	defer snmpStore.mu.Unlock()

	prevCopy := make(map[int]snmpPrevCounters, len(snmpStore.prev))
	for k, v := range snmpStore.prev {
		prevCopy[k] = v
	}

	var totalIn, totalOut, upIn, upOut, upSpeed float64

	for _, m := range metas {
		inO, outO := counters[m.idx][0], counters[m.idx][1]
		iface := SNMPInterface{
			Index: m.idx, Name: m.name, Alias: m.al,
			OperStatus: m.op, AdminStatus: m.ad, SpeedMbps: m.spd,
			InOctets: inO, OutOctets: outO,
			Role: ifaceRole(m.al, m.name),
		}

		if prev, ok := prevCopy[m.idx]; ok && !prev.at.IsZero() {
			dt := now.Sub(prev.at).Seconds()
			if dt >= 2.0 && (inO > 0 || outO > 0) && (prev.in > 0 || prev.out > 0) {
				dIn := counterDelta(prev.in, inO)
				dOut := counterDelta(prev.out, outO)
				iface.InMbps = (float64(dIn) / dt) * 8.0 / 1000000.0
				iface.OutMbps = (float64(dOut) / dt) * 8.0 / 1000000.0
				if m.spd > 0 {
					iface.InUtilPct = iface.InMbps / float64(m.spd) * 100
					iface.OutUtilPct = iface.OutMbps / float64(m.spd) * 100
				}
			}
		}
		if inO > 0 || outO > 0 {
			snmpStore.prev[m.idx] = snmpPrevCounters{in: inO, out: outO, at: now}
		}

		if m.op == 1 && m.spd > 0 && !containsAny(m.name, "NULL", "InLoop", "LoopBack", "Virtual") {
			if !isTrunkMemberPhys(m.name) {
				totalIn += iface.InMbps
				totalOut += iface.OutMbps
				if iface.Role == "bras" || iface.Role == "uplink" || iface.Role == "ix" || iface.Role == "cache" || iface.Role == "cgnat" {
					upIn += iface.InMbps
					upOut += iface.OutMbps
					upSpeed += float64(m.spd)
				}
			}
		}

		if m.op == 1 || m.al != "" || iface.InMbps > 0.1 || iface.OutMbps > 0.1 {
			ifaces = append(ifaces, iface)
		}
	}

	sort.Slice(ifaces, func(i, j int) bool {
		ai := ifaces[i].InMbps + ifaces[i].OutMbps
		aj := ifaces[j].InMbps + ifaces[j].OutMbps
		if ai == aj {
			return ifaces[i].Index < ifaces[j].Index
		}
		return ai > aj
	})

	snap.Interfaces = ifaces
	snap.TotalInMbps = totalIn
	snap.TotalOutMbps = totalOut
	snap.UplinkInMbps = upIn
	snap.UplinkOutMbps = upOut
	snap.Deduped = true
	if upSpeed > 0 {
		util := ((upIn + upOut) / 2) / upSpeed * 100
		snap.UplinkUtilPct = util
	}

	topN := 8
	if len(ifaces) < topN {
		topN = len(ifaces)
	}
	byIn := append([]SNMPInterface(nil), ifaces...)
	sort.Slice(byIn, func(i, j int) bool { return byIn[i].InMbps > byIn[j].InMbps })
	byOut := append([]SNMPInterface(nil), ifaces...)
	sort.Slice(byOut, func(i, j int) bool { return byOut[i].OutMbps > byOut[j].OutMbps })
	snap.TopIn = byIn[:min(topN, len(byIn))]
	snap.TopOut = byOut[:min(topN, len(byOut))]

	snap.OK = true
	snap.UpdatedAt = now.Unix()
	snap.PollMs = time.Since(start).Milliseconds()
	snmpStore.snap = snap
}

func isTrunkMemberPhys(name string) bool {
	// Portas físicas tipicamente agregadas em Eth-Trunk — excluir dos totais
	if containsAny(name, "Eth-Trunk", "NULL", "InLoop", "LoopBack", "Virtual", "Vlanif") {
		return false
	}
	return containsAny(name, "100GE", "40GE", "25GE", "10GE", "GigabitEthernet", "XGigabitEthernet", "GE")
}

func oidIndex(oid string) int {
	parts := splitOID(oid)
	if len(parts) == 0 {
		return 0
	}
	return parts[len(parts)-1]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func StartSNMPPoller() {
	log.Printf("SNMP poller → %s:%d community=%s", SNMPHost, SNMPPort, SNMPCommunity)
	go func() {
		for {
			pollSNMPOnce()
			time.Sleep(SNMPInterval)
		}
	}()
}
