package httpserver

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"
)

func internalSecret() string {
	s := os.Getenv("INTERNAL_JWT_SECRET")
	if s == "" { s = os.Getenv("JWT_SECRET") }
	if s == "" { s = "devsecret" }
	return s
}

func verifyJWT(token string) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 3 { return false }
	signingInput := parts[0] + "." + parts[1]
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil { return false }
	h := hmac.New(sha256.New, []byte(internalSecret()))
	h.Write([]byte(signingInput))
	if !hmac.Equal(sig, h.Sum(nil)) { return false }
	// check exp if present
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil { return false }
	var payload map[string]any
	if err := json.Unmarshal(payloadBytes, &payload); err == nil {
		if expRaw, ok := payload["exp"]; ok {
			switch v := expRaw.(type) {
			case float64:
				if int64(v) < time.Now().Unix() { return false }
			}
		}
	}
	return true
}

func isOpenPath(p string) bool {
	if p == "/health" || p == "/ready" { return true }
	return false
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isOpenPath(r.URL.Path) { next.ServeHTTP(w, r); return }
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
		token := strings.TrimPrefix(auth, "Bearer ")
		if !verifyJWT(token) { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
		next.ServeHTTP(w, r)
	})
}

