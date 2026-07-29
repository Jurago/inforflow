package main

import "testing"

func TestNormalizeASNQuery(t *testing.T) {
	tests := []struct{ in, want string }{
		{"AS15169", "AS15169"},
		{"15169", "AS15169"},
		{" as267521 ", "AS267521"},
		{"", ""},
	}
	for _, tc := range tests {
		got := normalizeASNQuery(tc.in)
		if got != tc.want {
			t.Fatalf("normalizeASNQuery(%q) = %q want %q", tc.in, got, tc.want)
		}
	}
}

func TestValidSessionToken(t *testing.T) {
	cfgMu.Lock()
	cfg.APIToken = "master-token"
	cfg.UIUser = ""
	cfg.UIPassword = ""
	cfgMu.Unlock()

	if !validSessionToken("master-token") {
		t.Fatal("master token should be valid")
	}
	tok := newSessionToken("admin")
	if tok == "" {
		t.Fatal("session token empty")
	}
	if !validSessionToken(tok) {
		t.Fatal("new session should be valid")
	}
	revokeSessionToken(tok)
	if validSessionToken(tok) {
		t.Fatal("revoked session should be invalid")
	}
}
