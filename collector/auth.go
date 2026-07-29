package main

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func uiAuthEnabled() bool {
	c := GetConfig()
	return c.UIUser != "" && c.UIPassword != ""
}

func safeStringEq(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func validSessionToken(got string) bool {
	c := GetConfig()
	if c.APIToken == "" {
		return !uiAuthEnabled()
	}
	if got == "" {
		got = ""
	}
	return safeStringEq(got, c.APIToken)
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		writeJSON(w, map[string]string{"error": "POST required"})
		return
	}
	c := GetConfig()
	if !uiAuthEnabled() {
		writeJSON(w, map[string]interface{}{
			"ok": false, "error": "login por usuário não configurado",
		})
		return
	}
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]interface{}{"ok": false, "error": "JSON inválido"})
		return
	}
	user := strings.TrimSpace(req.Username)
	pass := req.Password
	if !safeStringEq(user, c.UIUser) || !safeStringEq(pass, c.UIPassword) {
		w.WriteHeader(http.StatusUnauthorized)
		writeJSON(w, map[string]interface{}{"ok": false, "error": "Usuário ou senha inválidos"})
		return
	}
	if c.APIToken == "" {
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, map[string]interface{}{"ok": false, "error": "token de sessão não configurado"})
		return
	}
	writeJSON(w, map[string]interface{}{
		"ok":    true,
		"token": c.APIToken,
		"user":  c.UIUser,
	})
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	writeJSON(w, map[string]interface{}{"ok": true})
}

func handleAuthCheck(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	c := GetConfig()
	if c.APIToken == "" && !uiAuthEnabled() {
		writeJSON(w, map[string]interface{}{"ok": true, "auth_required": false})
		return
	}
	got := r.Header.Get("X-API-Token")
	if got == "" {
		got = r.URL.Query().Get("token")
	}
	writeJSON(w, map[string]interface{}{
		"ok":            validSessionToken(got),
		"auth_required": c.APIToken != "" || uiAuthEnabled(),
		"user":          c.UIUser,
	})
}
