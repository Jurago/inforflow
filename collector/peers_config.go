package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func loadPeersConfig() {
	path := filepath.Join(GetConfig().DataDir, "peers.json")
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var raw map[string]peerConfigEntry
	if json.Unmarshal(b, &raw) != nil {
		return
	}
	for asnStr, ent := range raw {
		asnStr = strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(asnStr)), "AS")
		// Metadado opcional: {"_ix": {"name":"IX.br"}} não é ASN
		if asnStr == "_IX" || asnStr == "IX" {
			continue
		}
		n, err := strconv.ParseUint(asnStr, 10, 32)
		if err != nil {
			continue
		}
		as := uint32(n)
		if ent.Name != "" {
			knownASNNames[as] = ent.Name
		}
		if ent.Role != "" {
			peerRoles[as] = ent.Role
		}
		if ent.Role == "ix" {
			cfgMu.Lock()
			if cfg.IXASN == 0 || cfg.IXASN == 26162 {
				cfg.IXASN = as
			}
			cfgMu.Unlock()
		}
	}
	log.Printf("peers.json: %d ASNs carregados (ix_asn=%d)", len(raw), GetConfig().IXASN)
}

type peerConfigEntry struct {
	Name string `json:"name"`
	Role string `json:"role"`
}

var peerRoles = map[uint32]string{}
