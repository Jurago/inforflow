package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	NetFlowPort = ":2055"
	UDPBufSize  = 65535

	netflowPktsTotal   uint64
	netflowFlowsTotal  uint64
	netflowUDPRcvBuf   int64
	netflowUDPQueue    int64 // bytes pending in kernel recv queue (approx)
	netflowKernelDrops uint64
	netflowConn        *net.UDPConn
)

// NetFlow v9 / IPFIX field type IDs (common subset)
const (
	fieldIN_BYTES      = 1
	fieldIN_PKTS       = 2
	fieldPROTOCOL      = 4
	fieldTOS           = 5
	fieldTCP_FLAGS     = 6
	fieldL4_SRC_PORT   = 7
	fieldIPV4_SRC_ADDR = 8
	fieldSRC_MASK      = 9
	fieldINPUT_SNMP    = 10
	fieldL4_DST_PORT   = 11
	fieldIPV4_DST_ADDR = 12
	fieldDST_MASK      = 13
	fieldOUTPUT_SNMP   = 14
	fieldIPV4_NEXT_HOP = 15
	fieldSRC_AS        = 16
	fieldDST_AS        = 17
	fieldBGP_NEXT_HOP  = 18
	fieldLAST_SWITCHED = 21
	fieldFIRST_SWITCHED = 22
	fieldIPV6_SRC_ADDR = 27
	fieldIPV6_DST_ADDR = 28
	fieldIPV6_NEXT_HOP = 62
	// Sampling / options
	fieldSAMPLING_INTERVAL        = 34
	fieldSAMPLING_ALGORITHM       = 35
	fieldFLOW_SAMPLER_ID          = 48
	fieldFLOW_SAMPLER_MODE        = 49
	fieldFLOW_SAMPLER_RANDOM_INT  = 50
	fieldSAMPLER_NAME             = 84
	fieldSAMPLING_PACKET_INTERVAL = 305
	fieldSAMPLER_ID               = 302
)

type TemplateField struct {
	Type   uint16
	Length uint16
}

type Template struct {
	ID     uint16
	Fields []TemplateField
}

type TemplateCache struct {
	mu        sync.RWMutex
	templates map[string]map[uint16]*Template
	options   map[string]map[uint16]*OptionsTemplate
}

type OptionsTemplate struct {
	ID           uint16
	ScopeFields  []TemplateField
	OptionFields []TemplateField
}

func NewTemplateCache() *TemplateCache {
	return &TemplateCache{
		templates: make(map[string]map[uint16]*Template),
		options:   make(map[string]map[uint16]*OptionsTemplate),
	}
}

func (c *TemplateCache) Set(exporter string, t *Template) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.templates[exporter] == nil {
		c.templates[exporter] = make(map[uint16]*Template)
	}
	c.templates[exporter][t.ID] = t
}

func (c *TemplateCache) Get(exporter string, id uint16) *Template {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if m := c.templates[exporter]; m != nil {
		return m[id]
	}
	return nil
}

func (c *TemplateCache) SetOptions(exporter string, t *OptionsTemplate) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.options[exporter] == nil {
		c.options[exporter] = make(map[uint16]*OptionsTemplate)
	}
	c.options[exporter][t.ID] = t
}

func (c *TemplateCache) GetOptions(exporter string, id uint16) *OptionsTemplate {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if m := c.options[exporter]; m != nil {
		return m[id]
	}
	return nil
}

type RawFlow struct {
	SrcIP   net.IP
	DstIP   net.IP
	NextHop net.IP
	SrcPort uint16
	DstPort uint16
	Proto   uint8
	Bytes   uint64
	Packets uint64
	SrcAS   uint32
	DstAS   uint32
	InIf    uint32
	OutIf   uint32
}

func NetFlowStats() map[string]interface{} {
	return map[string]interface{}{
		"packets_total":  atomic.LoadUint64(&netflowPktsTotal),
		"flows_decoded":  atomic.LoadUint64(&netflowFlowsTotal),
		"udp_rcvbuf":     atomic.LoadInt64(&netflowUDPRcvBuf),
		"udp_queue":      atomic.LoadInt64(&netflowUDPQueue),
		"kernel_drops":   atomic.LoadUint64(&netflowKernelDrops),
		"silent_s":       netFlowSilentSec(),
	}
}

func udpRcvBufBytes() int {
	mb := GetConfig().UDPRcvBufMB
	if mb <= 0 {
		mb = 32
	}
	if mb > 256 {
		mb = 256
	}
	return mb * 1024 * 1024
}

// pollUDPKernelStats lê fila/drops aproximados via /proc/net/udp e /proc/net/snmp.
func pollUDPKernelStats() {
	port := NetFlowPort
	if strings.HasPrefix(port, ":") {
		port = port[1:]
	}
	pnum, err := strconv.Atoi(port)
	if err != nil || pnum <= 0 {
		return
	}
	want := strings.ToUpper(fmt.Sprintf("%04X", pnum))

	if b, err := os.ReadFile("/proc/net/udp"); err == nil {
		lines := strings.Split(string(b), "\n")
		for _, line := range lines[1:] {
			fields := strings.Fields(line)
			if len(fields) < 5 {
				continue
			}
			// local_address is ip:port hex
			parts := strings.Split(fields[1], ":")
			if len(parts) != 2 || !strings.EqualFold(parts[1], want) {
				continue
			}
			// tx:rx queue hex
			qr := strings.Split(fields[4], ":")
			if len(qr) == 2 {
				if rx, e := strconv.ParseInt(qr[1], 16, 64); e == nil {
					atomic.StoreInt64(&netflowUDPQueue, rx)
				}
			}
			break
		}
	}

	if b, err := os.ReadFile("/proc/net/snmp"); err == nil {
		lines := strings.Split(string(b), "\n")
		for i := 0; i+1 < len(lines); i++ {
			if !strings.HasPrefix(lines[i], "Udp:") || !strings.HasPrefix(lines[i+1], "Udp:") {
				continue
			}
			hdr := strings.Fields(lines[i])
			val := strings.Fields(lines[i+1])
			if len(hdr) != len(val) {
				break
			}
			for j, h := range hdr {
				if h == "RcvbufErrors" {
					if n, e := strconv.ParseUint(val[j], 10, 64); e == nil {
						atomic.StoreUint64(&netflowKernelDrops, n)
					}
					break
				}
			}
			break
		}
	}
}

func StartNetFlowListener(handler func(RawFlow)) {
	addr, err := net.ResolveUDPAddr("udp", NetFlowPort)
	if err != nil {
		log.Fatalf("resolve UDP: %v", err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatalf("listen UDP %s: %v", NetFlowPort, err)
	}
	netflowConn = conn
	wantBuf := udpRcvBufBytes()
	if err := conn.SetReadBuffer(wantBuf); err != nil {
		log.Printf("aviso: buffer UDP %d: %v", wantBuf, err)
	}
	atomic.StoreInt64(&netflowUDPRcvBuf, int64(wantBuf))
	if f, err := conn.File(); err == nil {
		// Report effective SO_RCVBUF (kernel often doubles the request)
		var actual int
		if n, err := syscallGetSockOptRcvBuf(int(f.Fd())); err == nil {
			actual = n
			atomic.StoreInt64(&netflowUDPRcvBuf, int64(actual))
		}
		_ = f.Close()
		if actual > 0 {
			log.Printf("NetFlow SO_RCVBUF efetivo=%d (pedido=%d)", actual, wantBuf)
		}
	}

	cache := NewTemplateCache()
	log.Printf("NetFlow listener ativo em UDP %s (aguardando %s)", NetFlowPort, SourceIP)

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		for range ticker.C {
			pollUDPKernelStats()
		}
	}()

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		for range ticker.C {
			log.Printf("NetFlow: %d pacotes recebidos, %d flows decodificados, fila_udp=%d drops=%d",
				atomic.LoadUint64(&netflowPktsTotal),
				atomic.LoadUint64(&netflowFlowsTotal),
				atomic.LoadInt64(&netflowUDPQueue),
				atomic.LoadUint64(&netflowKernelDrops))
		}
	}()

	buf := make([]byte, UDPBufSize)
	for {
		n, remote, err := conn.ReadFromUDP(buf)
		if n == 0 || err != nil {
			continue
		}
		atomic.AddUint64(&netflowPktsTotal, 1)
		exporter := remote.IP.String()
		if !exporterAllowed(exporter) {
			continue
		}
		markNetFlowReceived()
		flows := parsePacket(buf[:n], cache, exporter)
		for _, f := range flows {
			atomic.AddUint64(&netflowFlowsTotal, 1)
			handler(f)
		}
	}
}

func parsePacket(data []byte, cache *TemplateCache, exporter string) []RawFlow {
	if len(data) < 4 {
		return nil
	}
	version := binary.BigEndian.Uint16(data[0:2])
	switch version {
	case 5:
		return parseV5(data)
	case 9:
		return parseV9(data, cache, exporter)
	case 10:
		return parseIPFIX(data, cache, exporter)
	default:
		return nil
	}
}

func parseV5(data []byte) []RawFlow {
	if len(data) < 24 {
		return nil
	}
	count := int(binary.BigEndian.Uint16(data[2:4]))
	const recordSize = 48
	offset := 24
	var flows []RawFlow
	for i := 0; i < count; i++ {
		if offset+recordSize > len(data) {
			break
		}
		rec := data[offset : offset+recordSize]
		f := RawFlow{
			SrcIP:   append(net.IP(nil), rec[0:4]...),
			DstIP:   append(net.IP(nil), rec[4:8]...),
			Packets: uint64(binary.BigEndian.Uint32(rec[16:20])),
			Bytes:   uint64(binary.BigEndian.Uint32(rec[20:24])),
			SrcPort: binary.BigEndian.Uint16(rec[32:34]),
			DstPort: binary.BigEndian.Uint16(rec[34:36]),
			Proto:   rec[38],
			SrcAS:   uint32(binary.BigEndian.Uint16(rec[40:42])),
			DstAS:   uint32(binary.BigEndian.Uint16(rec[42:44])),
		}
		flows = append(flows, f)
		offset += recordSize
	}
	return flows
}

func parseV9(data []byte, cache *TemplateCache, exporter string) []RawFlow {
	if len(data) < 20 {
		return nil
	}
	count := int(binary.BigEndian.Uint16(data[2:4]))
	offset := 20
	var flows []RawFlow
	setsParsed := 0

	for offset+4 <= len(data) && setsParsed < count+50 {
		flowSetID := binary.BigEndian.Uint16(data[offset : offset+2])
		length := int(binary.BigEndian.Uint16(data[offset+2 : offset+4]))
		if length < 4 || offset+length > len(data) {
			break
		}
		payload := data[offset+4 : offset+length]

		switch {
		case flowSetID == 0: // Template FlowSet
			parseTemplateSet(payload, cache, exporter)
		case flowSetID == 1: // Options Template — store similarly if needed
			parseOptionsTemplateSet(payload, cache, exporter)
		case flowSetID >= 256: // Data FlowSet (flows ou options data)
			if ot := cache.GetOptions(exporter, flowSetID); ot != nil {
				decodeOptionsData(payload, ot)
			} else if tmpl := cache.Get(exporter, flowSetID); tmpl != nil {
				flows = append(flows, decodeDataSet(payload, cache, exporter, flowSetID)...)
			} else if decoded := decodeKnownTemplate(payload, flowSetID); decoded != nil {
				flows = append(flows, decoded...)
			}
		}
		offset += length
		// Pad to 4-byte boundary is included in length for v9
		setsParsed++
	}
	return flows
}

func parseIPFIX(data []byte, cache *TemplateCache, exporter string) []RawFlow {
	if len(data) < 16 {
		return nil
	}
	msgLen := int(binary.BigEndian.Uint16(data[2:4]))
	if msgLen > len(data) {
		msgLen = len(data)
	}
	offset := 16
	var flows []RawFlow

	for offset+4 <= msgLen {
		setID := binary.BigEndian.Uint16(data[offset : offset+2])
		length := int(binary.BigEndian.Uint16(data[offset+2 : offset+4]))
		if length < 4 || offset+length > msgLen {
			break
		}
		payload := data[offset+4 : offset+length]

		switch {
		case setID == 2: // Template Set
			parseTemplateSet(payload, cache, exporter)
		case setID == 3: // Options Template
			parseOptionsTemplateSet(payload, cache, exporter)
		case setID >= 256:
			if ot := cache.GetOptions(exporter, setID); ot != nil {
				decodeOptionsData(payload, ot)
			} else if tmpl := cache.Get(exporter, setID); tmpl != nil {
				flows = append(flows, decodeDataSet(payload, cache, exporter, setID)...)
			} else if decoded := decodeKnownTemplate(payload, setID); decoded != nil {
				flows = append(flows, decoded...)
			}
		}
		offset += length
	}
	return flows
}

func parseTemplateSet(payload []byte, cache *TemplateCache, exporter string) {
	offset := 0
	for offset+4 <= len(payload) {
		templateID := binary.BigEndian.Uint16(payload[offset : offset+2])
		fieldCount := int(binary.BigEndian.Uint16(payload[offset+2 : offset+4]))
		offset += 4
		fields := make([]TemplateField, 0, fieldCount)
		for i := 0; i < fieldCount; i++ {
			if offset+4 > len(payload) {
				return
			}
			ftype := binary.BigEndian.Uint16(payload[offset : offset+2])
			flen := binary.BigEndian.Uint16(payload[offset+2 : offset+4])
			offset += 4
			// Enterprise bit (IPFIX)
			if ftype&0x8000 != 0 {
				if offset+4 > len(payload) {
					return
				}
				offset += 4 // skip enterprise number
				ftype = ftype & 0x7FFF
			}
			fields = append(fields, TemplateField{Type: ftype, Length: flen})
		}
		if templateID >= 256 {
			cache.Set(exporter, &Template{ID: templateID, Fields: fields})
			log.Printf("Template %d registrado (%d campos) de %s", templateID, fieldCount, exporter)
		}
	}
}

func parseOptionsTemplateSet(payload []byte, cache *TemplateCache, exporter string) {
	// NetFlow v9: TemplateID(2) + ScopeLen(2) + OptionLen(2) then scope fields then option fields
	// Also try ScopeCount/OptionCount form used by some vendors
	offset := 0
	for offset+6 <= len(payload) {
		templateID := binary.BigEndian.Uint16(payload[offset : offset+2])
		a := binary.BigEndian.Uint16(payload[offset+2 : offset+4])
		b := binary.BigEndian.Uint16(payload[offset+4 : offset+6])
		offset += 6

		var scopeFields, optFields []TemplateField
		// Heuristic: if a and b look like field counts (< 64), use count form; else byte lengths
		if a < 64 && b < 64 && a+b > 0 && a+b < 40 {
			scopeCount, optCount := int(a), int(b)
			for i := 0; i < scopeCount; i++ {
				if offset+4 > len(payload) {
					return
				}
				ftype := binary.BigEndian.Uint16(payload[offset : offset+2])
				flen := binary.BigEndian.Uint16(payload[offset+2 : offset+4])
				offset += 4
				scopeFields = append(scopeFields, TemplateField{ftype & 0x7FFF, flen})
			}
			for i := 0; i < optCount; i++ {
				if offset+4 > len(payload) {
					return
				}
				ftype := binary.BigEndian.Uint16(payload[offset : offset+2])
				flen := binary.BigEndian.Uint16(payload[offset+2 : offset+4])
				offset += 4
				optFields = append(optFields, TemplateField{ftype & 0x7FFF, flen})
			}
		} else {
			scopeLen, optLen := int(a), int(b)
			endScope := offset + scopeLen
			endOpt := endScope + optLen
			if endOpt > len(payload) {
				return
			}
			for offset+4 <= endScope {
				ftype := binary.BigEndian.Uint16(payload[offset : offset+2])
				flen := binary.BigEndian.Uint16(payload[offset+2 : offset+4])
				offset += 4
				scopeFields = append(scopeFields, TemplateField{ftype & 0x7FFF, flen})
			}
			offset = endScope
			for offset+4 <= endOpt {
				ftype := binary.BigEndian.Uint16(payload[offset : offset+2])
				flen := binary.BigEndian.Uint16(payload[offset+2 : offset+4])
				offset += 4
				optFields = append(optFields, TemplateField{ftype & 0x7FFF, flen})
			}
			offset = endOpt
		}
		if templateID >= 256 && len(optFields) > 0 {
			cache.SetOptions(exporter, &OptionsTemplate{
				ID: templateID, ScopeFields: scopeFields, OptionFields: optFields,
			})
			log.Printf("Options template %d registrado (scope=%d opt=%d) de %s",
				templateID, len(scopeFields), len(optFields), exporter)
		}
		// pad to 4
		for offset%4 != 0 && offset < len(payload) {
			offset++
		}
	}
}

func decodeOptionsData(payload []byte, ot *OptionsTemplate) {
	recSize := 0
	for _, f := range ot.ScopeFields {
		recSize += int(f.Length)
	}
	for _, f := range ot.OptionFields {
		recSize += int(f.Length)
	}
	if recSize <= 0 {
		return
	}
	for offset := 0; offset+recSize <= len(payload); offset += recSize {
		rec := payload[offset : offset+recSize]
		pos := 0
		for _, f := range ot.ScopeFields {
			pos += int(f.Length)
		}
		for _, f := range ot.OptionFields {
			flen := int(f.Length)
			if pos+flen > len(rec) {
				break
			}
			val := rec[pos : pos+flen]
			pos += flen
			switch f.Type {
			case fieldSAMPLING_INTERVAL, fieldFLOW_SAMPLER_RANDOM_INT, fieldSAMPLING_PACKET_INTERVAL:
				n := readUint(val)
				if n > 1 && n < 1000000 {
					sampling.SetNative(float64(n))
				}
			}
		}
	}
}

// Templates observados do exporter 170.245.127.191 (templates raramente reenviados).
// 1523 = IPv4 80 bytes | 1615 = IPv6 132 bytes
func decodeKnownTemplate(payload []byte, setID uint16) []RawFlow {
	switch setID {
	case 1523:
		return decodeIPv4Template1523(payload)
	case 1615:
		return decodeIPv6Template1615(payload)
	default:
		return nil
	}
}

func decodeIPv4Template1523(payload []byte) []RawFlow {
	const recSize = 80
	var flows []RawFlow
	for offset := 0; offset+recSize <= len(payload); offset += recSize {
		rec := payload[offset : offset+recSize]
		src := append(net.IP(nil), rec[0:4]...)
		dst := append(net.IP(nil), rec[4:8]...)
		if src.IsUnspecified() && dst.IsUnspecified() {
			continue
		}
		f := RawFlow{
			SrcIP:   src,
			DstIP:   dst,
			NextHop: append(net.IP(nil), rec[8:12]...), // gap antes de counters — next-hop típico Huawei
			Packets: uint64(binary.BigEndian.Uint32(rec[12:16])),
			Bytes:   uint64(binary.BigEndian.Uint32(rec[16:20])),
			SrcPort: binary.BigEndian.Uint16(rec[40:42]),
			DstPort: binary.BigEndian.Uint16(rec[42:44]),
			Proto:   rec[59],
			InIf:    uint32(binary.BigEndian.Uint16(rec[20:22])),
			OutIf:   uint32(binary.BigEndian.Uint16(rec[22:24])),
		}
		// BGP AS — candidatos comuns no fim do registro 80B
		asA := binary.BigEndian.Uint32(rec[60:64])
		asB := binary.BigEndian.Uint32(rec[64:68])
		if plausibleAS(asA) {
			f.SrcAS = asA
		} else if as16 := binary.BigEndian.Uint16(rec[60:62]); plausibleAS(uint32(as16)) {
			f.SrcAS = uint32(as16)
		}
		if plausibleAS(asB) {
			f.DstAS = asB
		} else if as16 := binary.BigEndian.Uint16(rec[62:64]); plausibleAS(uint32(as16)) {
			f.DstAS = uint32(as16)
		}
		// fallback uint16 AS próximos às portas
		if f.SrcAS == 0 {
			if a := binary.BigEndian.Uint16(rec[48:50]); plausibleAS(uint32(a)) {
				f.SrcAS = uint32(a)
			}
		}
		if f.DstAS == 0 {
			if a := binary.BigEndian.Uint16(rec[50:52]); plausibleAS(uint32(a)) {
				f.DstAS = uint32(a)
			}
		}
		flows = append(flows, f)
	}
	return flows
}

func plausibleAS(as uint32) bool {
	if as == 0 || as == 0xffffffff || as == 0xffff || as == 23456 {
		return false
	}
	// 16-bit
	if as <= 65535 {
		return true
	}
	// Privado 32-bit (RFC 6996)
	if as >= 4200000000 && as <= 4294967294 {
		return true
	}
	// Espaço público 32-bit atribuído na prática (~até ~500k)
	if as >= 65536 && as <= 500000 {
		return true
	}
	return false
}

func decodeIPv6Template1615(payload []byte) []RawFlow {
	const recSize = 132
	var flows []RawFlow
	for offset := 0; offset+recSize <= len(payload); offset += recSize {
		rec := payload[offset : offset+recSize]
		src := append(net.IP(nil), rec[0:16]...)
		dst := append(net.IP(nil), rec[16:32]...)
		if (src.IsUnspecified() && dst.IsUnspecified()) || (src[0] < 0x20 && dst[0] < 0x20) {
			continue
		}
		f := RawFlow{
			SrcIP:   src,
			DstIP:   dst,
			Packets: uint64(binary.BigEndian.Uint32(rec[48:52])),
			Bytes:   uint64(binary.BigEndian.Uint32(rec[52:56])),
			SrcPort: binary.BigEndian.Uint16(rec[96:98]),
			DstPort: binary.BigEndian.Uint16(rec[98:100]),
			Proto:   rec[107],
			InIf:    uint32(binary.BigEndian.Uint16(rec[56:58])),
			OutIf:   uint32(binary.BigEndian.Uint16(rec[58:60])),
		}
		asA := binary.BigEndian.Uint32(rec[112:116])
		asB := binary.BigEndian.Uint32(rec[116:120])
		if plausibleAS(asA) {
			f.SrcAS = asA
		}
		if plausibleAS(asB) {
			f.DstAS = asB
		}
		flows = append(flows, f)
	}
	return flows
}

func decodeDataSet(payload []byte, cache *TemplateCache, exporter string, setID uint16) []RawFlow {
	tmpl := cache.Get(exporter, setID)
	if tmpl == nil {
		return nil
	}
	recordSize := 0
	for _, f := range tmpl.Fields {
		if f.Length == 0xFFFF {
			// variable length — can't precompute; handle below
			recordSize = -1
			break
		}
		recordSize += int(f.Length)
	}

	var flows []RawFlow
	offset := 0

	if recordSize > 0 {
		for offset+recordSize <= len(payload) {
			rec := payload[offset : offset+recordSize]
			if f, ok := extractFlow(rec, tmpl.Fields); ok {
				flows = append(flows, f)
			}
			offset += recordSize
		}
	} else {
		// Variable-length fields
		for offset < len(payload) {
			start := offset
			for _, f := range tmpl.Fields {
				if offset >= len(payload) {
					return flows
				}
				flen := int(f.Length)
				if f.Length == 0xFFFF {
					if payload[offset] < 255 {
						flen = int(payload[offset])
						offset++
					} else {
						if offset+3 > len(payload) {
							return flows
						}
						flen = int(binary.BigEndian.Uint16(payload[offset+1 : offset+3]))
						offset += 3
					}
				}
				offset += flen
			}
			if start >= offset || offset > len(payload) {
				break
			}
			if f, ok := extractFlow(payload[start:offset], tmpl.Fields); ok {
				flows = append(flows, f)
			}
			// Avoid infinite loop
			if offset == start {
				break
			}
		}
	}
	return flows
}

func extractFlow(rec []byte, fields []TemplateField) (RawFlow, bool) {
	var f RawFlow
	offset := 0
	gotAddr := false

	for _, field := range fields {
		flen := int(field.Length)
		if field.Length == 0xFFFF {
			if offset >= len(rec) {
				break
			}
			if rec[offset] < 255 {
				flen = int(rec[offset])
				offset++
			} else {
				if offset+3 > len(rec) {
					break
				}
				flen = int(binary.BigEndian.Uint16(rec[offset+1 : offset+3]))
				offset += 3
			}
		}
		if offset+flen > len(rec) {
			break
		}
		val := rec[offset : offset+flen]
		offset += flen

		switch field.Type {
		case fieldIPV4_SRC_ADDR:
			if flen == 4 {
				f.SrcIP = append(net.IP(nil), val...)
				gotAddr = true
			}
		case fieldIPV4_DST_ADDR:
			if flen == 4 {
				f.DstIP = append(net.IP(nil), val...)
				gotAddr = true
			}
		case fieldIPV6_SRC_ADDR:
			if flen == 16 {
				f.SrcIP = append(net.IP(nil), val...)
				gotAddr = true
			}
		case fieldIPV6_DST_ADDR:
			if flen == 16 {
				f.DstIP = append(net.IP(nil), val...)
				gotAddr = true
			}
		case fieldL4_SRC_PORT:
			f.SrcPort = readUint16(val)
		case fieldL4_DST_PORT:
			f.DstPort = readUint16(val)
		case fieldPROTOCOL:
			if flen >= 1 {
				f.Proto = val[0]
			}
		case fieldIN_BYTES:
			f.Bytes = readUint(val)
		case fieldIN_PKTS:
			f.Packets = readUint(val)
		case fieldSRC_AS:
			f.SrcAS = uint32(readUint(val))
		case fieldDST_AS:
			f.DstAS = uint32(readUint(val))
		case fieldIPV4_NEXT_HOP, fieldBGP_NEXT_HOP:
			if flen == 4 {
				f.NextHop = append(net.IP(nil), val...)
			}
		case fieldIPV6_NEXT_HOP:
			if flen == 16 {
				f.NextHop = append(net.IP(nil), val...)
			}
		case fieldINPUT_SNMP:
			f.InIf = uint32(readUint(val))
		case fieldOUTPUT_SNMP:
			f.OutIf = uint32(readUint(val))
		}
	}

	if !gotAddr || (f.Bytes == 0 && f.Packets == 0) {
		// Still accept flows with addresses even if counters are zero occasionally
		if !gotAddr {
			return f, false
		}
	}
	if f.SrcIP == nil || f.DstIP == nil {
		return f, false
	}
	return f, true
}

func readUint16(b []byte) uint16 {
	switch len(b) {
	case 1:
		return uint16(b[0])
	case 2:
		return binary.BigEndian.Uint16(b)
	default:
		if len(b) >= 2 {
			return binary.BigEndian.Uint16(b[len(b)-2:])
		}
	}
	return 0
}

func readUint(b []byte) uint64 {
	switch len(b) {
	case 1:
		return uint64(b[0])
	case 2:
		return uint64(binary.BigEndian.Uint16(b))
	case 4:
		return uint64(binary.BigEndian.Uint32(b))
	case 8:
		return binary.BigEndian.Uint64(b)
	default:
		var v uint64
		for _, x := range b {
			v = (v << 8) | uint64(x)
		}
		return v
	}
}

func protoName(p uint8) string {
	switch p {
	case 1:
		return "ICMP"
	case 6:
		return "TCP"
	case 17:
		return "UDP"
	case 47:
		return "GRE"
	case 50:
		return "ESP"
	default:
		return fmt.Sprintf("IP/%d", p)
	}
}
