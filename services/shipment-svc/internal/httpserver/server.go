package httpserver

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"shipment-svc/internal/repo"

	"github.com/gorilla/mux"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type Server struct {
	r    *mux.Router
	repo *repo.Repository
}

var shipmentsCreated metric.Int64Counter
var shipmentIdemReplays metric.Int64Counter

func init() {
	m := otel.GetMeterProvider().Meter("shipment-svc")
	var err error
	shipmentsCreated, err = m.Int64Counter("shipments_created_total")
	if err != nil {
		log.Printf("metrics init error: %v", err)
	}
	shipmentIdemReplays, err = m.Int64Counter("shipment_idempotency_replays_total")
	if err != nil {
		log.Printf("metrics init error: %v", err)
	}
}

func New() *Server {
	s := &Server{r: mux.NewRouter()}
	s.routes()
	return s
}

func NewWithRepo(rp *repo.Repository) *Server {
	s := &Server{r: mux.NewRouter(), repo: rp}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.r.ServeHTTP(w, r) }

func (s *Server) routes() {
	s.r.Use(requestLogger)
	s.r.Use(authMiddleware)
	s.r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	s.r.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		if s.repo != nil {
			if err := s.repo.DB.Pool.Ping(r.Context()); err != nil {
				http.Error(w, "not ready", http.StatusServiceUnavailable)
				return
			}
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
	})

	s.r.HandleFunc("/v1/shipments", s.createShipment()).Methods("POST")
	s.r.HandleFunc("/v1/shipments/{id}", s.getShipment()).Methods("GET")
	s.r.HandleFunc("/v1/shipments/{id}", s.updateShipment()).Methods("PATCH")
	s.r.HandleFunc("/v1/shipments/{id}/labels", s.createLabel()).Methods("POST")

}

func (s *Server) createShipment() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ShipperID    string         `json:"shipperId"`
			ServiceLevel string         `json:"serviceLevel"`
			From         repo.Address   `json:"from"`
			To           repo.Address   `json:"to"`
			Value        float64        `json:"value"`
			Packages     []repo.Package `json:"packages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if req.ShipperID == "" || req.ServiceLevel == "" || len(req.Packages) == 0 {
			http.Error(w, "invalid", http.StatusBadRequest)
			return
		}
		idem := r.Header.Get("X-Idempotency-Key")
		if idem == "" {
			http.Error(w, "missing X-Idempotency-Key", http.StatusBadRequest)
			return
		}
		id, replay, err := s.repo.CreateShipmentIdempotent(r.Context(), repo.Shipment{ShipperID: req.ShipperID, ServiceLevel: req.ServiceLevel, From: req.From, To: req.To, Value: req.Value}, req.Packages, "system", idem)
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		// Metrics + span attributes
		func() {
			defer func() { _ = recover() }()
			tenant := baggage.FromContext(r.Context()).Member("tenant.id").Value()
			attrs := []attribute.KeyValue{}
			if tenant != "" {
				attrs = append(attrs, attribute.String("tenant_id", tenant))
			}
			if replay {
				shipmentIdemReplays.Add(r.Context(), 1, metric.WithAttributes(attrs...))
			} else {
				shipmentsCreated.Add(r.Context(), 1, metric.WithAttributes(attrs...))
			}
			span := trace.SpanFromContext(r.Context())
			if span != nil && replay {
				span.SetAttributes(attribute.String("idempotent.replay", "true"))
			}
		}()
		if replay {
			w.Header().Set("Idempotent-Replayed", "true")
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusCreated)
		}
		json.NewEncoder(w).Encode(map[string]any{"id": id, "status": "CREATED"})
	}
}

func (s *Server) getShipment() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		sh, pkgs, err := s.repo.GetShipment(r.Context(), vars["id"])
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		if sh == nil {
			http.NotFound(w, r)
			return
		}
		labels, err := s.repo.ListLabels(r.Context(), vars["id"])
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]any{"shipment": sh, "packages": pkgs, "labels": labels})
	}
}

func (s *Server) updateShipment() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		var body struct {
			Status string `json:"status"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.Status != "" {
			_ = s.repo.UpdateShipmentStatus(r.Context(), vars["id"], body.Status, "system")
		}
		json.NewEncoder(w).Encode(map[string]string{"result": "ok"})
	}
}

func (s *Server) createLabel() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		var body struct {
			ID         string  `json:"id"`
			Format     string  `json:"format"`
			StorageURL *string `json:"storageUrl"`
			Checksum   *string `json:"checksum"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Format == "" {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		id, err := s.repo.InsertLabel(r.Context(), vars["id"], body.ID, body.Format, body.StorageURL, body.Checksum)
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{"id": id, "format": body.Format, "shipmentId": vars["id"], "storageUrl": body.StorageURL})
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
		shipmentID := ""
		if strings.HasPrefix(r.URL.Path, "/v1/shipments/") {
			parts := strings.Split(r.URL.Path, "/")
			if len(parts) > 3 {
				shipmentID = parts[3]
			}
		}
		members := []baggage.Member{}
		if tenantID != "" {
			if m, err := baggage.NewMember("tenant.id", tenantID); err == nil {
				members = append(members, m)
			}
		}
		if shipmentID != "" {
			if m, err := baggage.NewMember("app.shipment_id", shipmentID); err == nil {
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
			if shipmentID != "" {
				span.SetAttributes(attribute.String("app.shipment_id", shipmentID))
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

		log.Printf("request service=shipment-svc request_id=%s trace_id=%s span_id=%s method=%s path=%s status=%d duration_ms=%d response_bytes=%d client_ip=%s",
			rid, traceID, spanID, r.Method, r.URL.Path, rec.status, durMs, rec.size, ip)
	})
}
