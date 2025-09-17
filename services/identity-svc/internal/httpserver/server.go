package httpserver

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"identity-svc/internal/jwtutil"
	"identity-svc/internal/repo"

	"github.com/gorilla/mux"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/trace"
)

type Server struct {
	router *mux.Router
	repo   *repo.Repository
}

func New() *Server {
	s := &Server{router: mux.NewRouter()}
	s.routes()
	return s
}

func NewWithRepo(rp *repo.Repository) *Server {
	s := &Server{router: mux.NewRouter(), repo: rp}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.router.ServeHTTP(w, r) }

func (s *Server) routes() {
	s.router.Use(requestLogger)

	s.router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	s.router.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		if s.repo != nil {
			if err := s.repo.DB.Pool.Ping(r.Context()); err != nil {
				http.Error(w, "not ready", http.StatusServiceUnavailable)
				return
			}
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
	})

	// Auth
	s.router.HandleFunc("/v1/auth/login", s.login()).Methods("POST")
	s.router.HandleFunc("/v1/auth/refresh", s.refresh()).Methods("POST")
	s.router.HandleFunc("/v1/auth/mfa/verify", s.mfaVerify()).Methods("POST")

	// Users (admin)
	s.router.Handle("/v1/users", s.auth(http.HandlerFunc(s.createUser()))).Methods("POST")
	s.router.Handle("/v1/users/{id}", s.auth(http.HandlerFunc(s.getUser()))).Methods("GET")
}

type loginReq struct{ Email, Password, MfaCode string }

type tokenResp struct {
	AccessToken, RefreshToken string
	MfaRequired               bool
}

func (s *Server) login() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req loginReq
		json.NewDecoder(r.Body).Decode(&req)
		if req.Email == "" || req.Password == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		// lookup user
		u, err := s.repo.FindByEmail(r.Context(), req.Email)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if repo.HashPassword(u.PasswordSalt, req.Password) != u.PasswordHash {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		// TODO: MFA handling from user.mfa_secret
		access, _ := jwtutil.Sign(map[string]any{"sub": u.ID, "email": u.Email, "role": u.Role})
		refresh, _ := jwtutil.Sign(map[string]any{"sub": u.ID, "typ": "refresh"})
		json.NewEncoder(w).Encode(tokenResp{AccessToken: access, RefreshToken: refresh})
	}
}

func (s *Server) refresh() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(tokenResp{AccessToken: os.Getenv("DUMMY_ACCESS"), RefreshToken: ""})
	}
}

func (s *Server) mfaVerify() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(tokenResp{AccessToken: "token", RefreshToken: "token"})
	}
}

func (s *Server) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ah := r.Header.Get("Authorization")
		if len(ah) < 8 || ah[:7] != "Bearer " {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		tok := ah[7:]
		_, err := jwtutil.Parse(tok)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) createUser() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct{ Email, Password, Role string }
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Email == "" || body.Password == "" {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		salt := os.Getenv("DEFAULT_SALT")
		if salt == "" {
			salt = "localsalt"
		}
		h := repo.HashPassword(salt, body.Password)
		id, err := s.repo.CreateUser(r.Context(), body.Email, salt, h, func() string {
			if body.Role == "" {
				return "SHIPPER"
			}
			return body.Role
		}())
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{"id": id, "email": body.Email, "role": body.Role})
	}
}

func (s *Server) getUser() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		json.NewEncoder(w).Encode(map[string]any{"id": vars["id"], "email": "admin@example.com", "role": "ADMIN"})
	}
}

// ---- request logging middleware ----

type ctxKey string

const reqIDKey ctxKey = "req_id"

func genReqID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func RequestIDFromContext(ctx context.Context) string {
	if v := ctx.Value(reqIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	size   int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.status = code
	sr.ResponseWriter.WriteHeader(code)
}

func (sr *statusRecorder) Write(b []byte) (int, error) {
	n, err := sr.ResponseWriter.Write(b)
	sr.size += n
	return n, err
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		ip := strings.TrimSpace(parts[0])
		if ip != "" {
			return ip
		}
	}
	hostPort := r.RemoteAddr
	if host, _, err := net.SplitHostPort(hostPort); err == nil {
		return host
	}
	return hostPort
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := r.Header.Get("X-Request-ID")
		if rid == "" {
			rid = r.Header.Get("X-Correlation-ID")
		}
		if rid == "" {
			rid = genReqID()
		}
		ctx := context.WithValue(r.Context(), reqIDKey, rid)

		// Baggage & domain context
		tenantID := r.Header.Get("X-Tenant-Id")
		debugTrace := r.Header.Get("X-Debug-Trace")
		members := []baggage.Member{}
		if tenantID != "" {
			if m, err := baggage.NewMember("tenant.id", tenantID); err == nil {
				members = append(members, m)
			}
		}
		if debugTrace != "" {
			if m, err := baggage.NewMember("debug", "true"); err == nil {
				members = append(members, m)
			}
		}
		if len(members) > 0 {
			bg, _ := baggage.New(members...)
			ctx = baggage.ContextWithBaggage(ctx, bg)
		}

		r = r.WithContext(ctx)
		w.Header().Set("X-Request-ID", rid)
		w.Header().Set("X-Correlation-ID", rid)

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()
		next.ServeHTTP(rec, r)
		durMs := time.Since(start).Milliseconds()
		ip := clientIP(r)

		span := trace.SpanFromContext(r.Context())
		if span != nil {
			if tenantID != "" {
				span.SetAttributes(attribute.String("tenant.id", tenantID))
			}
			if debugTrace != "" {
				span.SetAttributes(attribute.String("debug", "true"))
			}
			span.SetAttributes(attribute.String("http.route", r.URL.Path))
		}

		sc := span.SpanContext()
		traceID := ""
		spanID := ""
		if sc.IsValid() {
			traceID = sc.TraceID().String()
			spanID = sc.SpanID().String()
		}

		log.Printf("request service=identity-svc request_id=%s trace_id=%s span_id=%s method=%s path=%s status=%d duration_ms=%d response_bytes=%d client_ip=%s",
			rid, traceID, spanID, r.Method, r.URL.Path, rec.status, durMs, rec.size, ip)
	})
}
