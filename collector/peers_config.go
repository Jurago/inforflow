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
	}
	log.Printf("peers.json: %d ASNs carregados", len(raw))
}

type peerConfigEntry struct {
	Name string `json:"name"`
	Role string `json:"role"`
}

var peerRoles = map[uint32]string{}
