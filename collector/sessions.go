package main

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

const sessionTTL = 24 * time.Hour

type sessionEntry struct {
	user      string
	expiresAt time.Time
}

var (
	sessionsMu sync.RWMutex
	sessions   = map[string]sessionEntry{}
)

func newSessionToken(user string) string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	token := hex.EncodeToString(b)
	sessionsMu.Lock()
	sessions[token] = sessionEntry{user: user, expiresAt: time.Now().Add(sessionTTL)}
	sessionsMu.Unlock()
	return token
}

func revokeSessionToken(token string) {
	sessionsMu.Lock()
	delete(sessions, token)
	sessionsMu.Unlock()
}

func validSessionToken(got string) bool {
	c := GetConfig()
	if got == "" {
		return !uiAuthEnabled() && c.APIToken == ""
	}
	if c.APIToken != "" && safeStringEq(got, c.APIToken) {
		return true
	}
	sessionsMu.RLock()
	ent, ok := sessions[got]
	sessionsMu.RUnlock()
	if !ok {
		return false
	}
	if time.Now().After(ent.expiresAt) {
		revokeSessionToken(got)
		return false
	}
	return true
}

func pruneSessions() {
	now := time.Now()
	sessionsMu.Lock()
	for tok, ent := range sessions {
		if now.After(ent.expiresAt) {
			delete(sessions, tok)
		}
	}
	sessionsMu.Unlock()
}

func startSessionJanitor() {
	go func() {
		for {
			time.Sleep(time.Hour)
			pruneSessions()
		}
	}()
}
