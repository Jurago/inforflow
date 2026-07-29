package main

import (
	"fmt"
	"log"
	"sort"
	"sync"
	"time"
)

// BGP4-MIB (RFC 4273)
const (
	oidBgpPeerState      = "1.3.6.1.2.1.15.3.1.2"
	oidBgpPeerRemoteAddr = "1.3.6.1.2.1.15.3.1.7"
	oidBgpPeerRemoteAs   = "1.3.6.1.2.1.15.3.1.9"
	oidBgpPeerInUpdates  = "1.3.6.1.2.1.15.3.1.10"
	oidBgpPeerOutUpdates = "1.3.6.1.2.1.15.3.1.11"
	oidBgpPeerIdentifier = "1.3.6.1.2.1.15.3.1.1"
	oidBgpLocalAs        = "1.3.6.1.2.1.15.2.0"
	BGPInterval          = 15 * time.Second
)

var bgpStateNames = map[int]string{
	1: "idle", 2: "connect", 3: "active",
	4: "opensent", 5: "openconfirm", 6: "established",
}

// Known ASNs seen on this edge (IX.br SP + peers).
var knownASNNames = map[uint32]string{
	26162:  "IX.br",
	15169:  "Google",
	32934:  "Meta",
	6939:   "Hurricane Electric",
	266022: "Infornet",
	272694: "AS272694",
	263433: "AS263433",
	267311: "AS267311",
	268843: "AS268843",
	269096: "N&K Tecnologia",
	11344:  "AS11344",
	20121:  "AS20121",
	65332:  "iBGP / privado",
	174:    "Cogent",
	1299:   "Telia",
	13335:  "Cloudflare",
	20940:  "Akamai",
	2906:   "Netflix",
	16509:  "Amazon",
	8075:   "Microsoft",
	714:    "Apple",
}

type BGPPeer struct {
	RemoteAddr  string  `json:"remote_addr"`
	RemoteAS    uint32  `json:"remote_as"`
	ASN         string  `json:"asn"`
	Name        string  `json:"name"`
	State       int     `json:"state"`
	StateName   string  `json:"state_name"`
	Established bool    `json:"established"`
	PeerID      string  `json:"peer_id"`
	InUpdates   uint64  `json:"in_updates"`
	OutUpdates  uint64  `json:"out_updates"`
	Role        string  `json:"role"` // ix | content | transit | regional | local | private
	Bytes       int64   `json:"bytes"`
	Mbps        float64 `json:"mbps"`
	Flows       int64   `json:"flows"`
}

type BGPSnapshot struct {
	OK           bool      `json:"ok"`
	LocalAS      uint32    `json:"local_as"`
	LocalASN     string    `json:"local_asn"`
	Total        int       `json:"total"`
	Established  int       `json:"established"`
	UpdatedAt    int64     `json:"updated_at"`
	PollMs       int64     `json:"poll_ms"`
	Peers        []BGPPeer `json:"peers"`
	Error        string    `json:"error,omitempty"`
}

type BGPStore struct {
	mu       sync.RWMutex
	snap     BGPSnapshot
	byIP     map[string]*BGPPeer // remote addr → peer
	byAS     map[uint32][]*BGPPeer
	asNames  map[uint32]string
}

var bgpStore = &BGPStore{
	byIP:    make(map[string]*BGPPeer),
	byAS:    make(map[uint32][]*BGPPeer),
	asNames: make(map[uint32]string),
	snap:    BGPSnapshot{},
}

func asnDisplayName(as uint32) string {
	if as == 0 {
		return ""
	}
	if n := ipapi.NameForASN(as); n != "" {
		return n
	}
	if n, ok := knownASNNames[as]; ok {
		return n
	}
	bgpStore.mu.RLock()
	defer bgpStore.mu.RUnlock()
	if n, ok := bgpStore.asNames[as]; ok {
		return n
	}
	return fmt.Sprintf("AS%d", as)
}

func peerRoleForAS(as uint32) string {
	if r, ok := peerRoles[as]; ok && r != "" {
		return r
	}
	switch as {
	case 26162:
		return "ix"
	case 15169, 32934, 13335, 20940, 2906, 16509, 714:
		return "content"
	case 6939, 174, 1299:
		return "transit"
	case 65332:
		return "private"
	case 266022, 11344, 269096, 263433, 267311, 268843, 272694:
		return "regional"
	default:
		if as >= 64512 && as <= 65534 {
			return "private"
		}
		return "regional"
	}
}

func (s *BGPStore) Get() BGPSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := s.snap
	out.Peers = append([]BGPPeer(nil), s.snap.Peers...)
	return out
}

func (s *BGPStore) LookupIP(ip string) (BGPPeer, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.byIP[ip]
	if !ok || p == nil {
		return BGPPeer{}, false
	}
	return *p, true
}

func (s *BGPStore) LookupAS(as uint32) (BGPPeer, bool) {
	if as == 0 {
		return BGPPeer{}, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := s.byAS[as]
	if len(list) == 0 {
		return BGPPeer{}, false
	}
	// Prefer established session
	for _, p := range list {
		if p.Established {
			return *p, true
		}
	}
	return *list[0], true
}

func (s *BGPStore) HasAS(as uint32) bool {
	_, ok := s.LookupAS(as)
	return ok
}

func walkIndexed(base string, limit int) map[string]interface{} {
	raw := snmpWalk(base, limit)
	out := make(map[string]interface{}, len(raw))
	b := trimDot(base)
	for oid, v := range raw {
		oid = trimDot(oid)
		if len(oid) > len(b)+1 && oid[:len(b)] == b && oid[len(b)] == '.' {
			out[oid[len(b)+1:]] = v
		}
	}
	return out
}

func pollBGPOnce() {
	start := time.Now()
	snap := BGPSnapshot{}

	if v, err := snmpGet(oidBgpLocalAs); err == nil {
		snap.LocalAS = uint32(asUint64(v))
		snap.LocalASN = fmt.Sprintf("AS%d", snap.LocalAS)
	}

	addrs := walkIndexed(oidBgpPeerRemoteAddr, 80)
	if len(addrs) == 0 {
		snap.OK = false
		snap.Error = "sem peers BGP via SNMP"
		bgpStore.mu.Lock()
		bgpStore.snap = snap
		bgpStore.mu.Unlock()
		return
	}

	asns := walkIndexed(oidBgpPeerRemoteAs, 80)
	states := walkIndexed(oidBgpPeerState, 80)
	inUpd := walkIndexed(oidBgpPeerInUpdates, 80)
	outUpd := walkIndexed(oidBgpPeerOutUpdates, 80)
	idents := walkIndexed(oidBgpPeerIdentifier, 80)

	peers := make([]BGPPeer, 0, len(addrs))
	asNames := make(map[uint32]string)

	for idx := range addrs {
		ip := asString(addrs[idx])
		if ip == "" || ip == "<nil>" {
			// IpAddress may already be string; also try index itself (x.x.x.x)
			ip = idx
		}
		// Prefer OID index as remote IP (always dotted quad)
		if looksLikeIPv4Index(idx) {
			ip = idx
		}
		as := uint32(asUint64(asns[idx]))
		st := int(asUint64(states[idx]))
		name := asnDisplayName(as)
		if name == fmt.Sprintf("AS%d", as) && as > 0 {
			name = fmt.Sprintf("AS%d", as)
		}
		p := BGPPeer{
			RemoteAddr:  ip,
			RemoteAS:    as,
			ASN:         fmt.Sprintf("AS%d", as),
			Name:        name,
			State:       st,
			StateName:   bgpStateNames[st],
			Established: st == 6,
			PeerID:      asString(idents[idx]),
			InUpdates:   asUint64(inUpd[idx]),
			OutUpdates:  asUint64(outUpd[idx]),
			Role:        peerRoleForAS(as),
		}
		if p.StateName == "" {
			p.StateName = fmt.Sprintf("state-%d", st)
		}
		peers = append(peers, p)
		if as > 0 {
			asNames[as] = name
		}
		if p.Established {
			snap.Established++
		}
	}

	sort.Slice(peers, func(i, j int) bool {
		if peers[i].Established != peers[j].Established {
			return peers[i].Established
		}
		if peers[i].RemoteAS != peers[j].RemoteAS {
			return peers[i].RemoteAS < peers[j].RemoteAS
		}
		return peers[i].RemoteAddr < peers[j].RemoteAddr
	})

	snap.Peers = peers
	snap.Total = len(peers)
	snap.OK = true
	snap.UpdatedAt = time.Now().Unix()
	snap.PollMs = time.Since(start).Milliseconds()

	bgpStore.mu.Lock()
	bgpStore.snap = snap
	bgpStore.asNames = asNames
	bgpStore.byIP = make(map[string]*BGPPeer, len(peers))
	bgpStore.byAS = make(map[uint32][]*BGPPeer)
	for i := range bgpStore.snap.Peers {
		p := &bgpStore.snap.Peers[i]
		bgpStore.byIP[p.RemoteAddr] = p
		if p.RemoteAS > 0 {
			bgpStore.byAS[p.RemoteAS] = append(bgpStore.byAS[p.RemoteAS], p)
		}
	}
	bgpStore.mu.Unlock()
}

func looksLikeIPv4Index(s string) bool {
	dots := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '.' {
			dots++
			continue
		}
		if c < '0' || c > '9' {
			return false
		}
	}
	return dots == 3
}

// annotatePeersWithTraffic fills Bytes/Mbps/Flows on a copy of peers from flow aggregates.
func annotatePeersWithTraffic(peers []BGPPeer, byASNBytes, byASNWindow map[uint32]int64, byASNFlows map[uint32]int64) []BGPPeer {
	const windowSec = 10.0
	out := make([]BGPPeer, len(peers))
	copy(out, peers)
	seen := map[uint32]bool{}
	for i := range out {
		as := out[i].RemoteAS
		out[i].Bytes = byASNBytes[as]
		out[i].Flows = byASNFlows[as]
		out[i].Mbps = float64(byASNWindow[as]) * 8 / windowSec / 1e6
		seen[as] = true
	}
	return out
}

func StartBGPPoller() {
	log.Printf("BGP SNMP poller → BGP4-MIB @ %s:%d", SNMPHost, SNMPPort)
	go func() {
		pollBGPOnce()
		for {
			time.Sleep(BGPInterval)
			pollBGPOnce()
		}
	}()
}
