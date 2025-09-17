package httpserver

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"math"
	"net"
	"net/http"
	"strings"
	"time"

	"tracking-svc/internal/repo"

	"github.com/gorilla/mux"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/trace"
)

type Server struct {
	r    *mux.Router
	repo *repo.Repository
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

	s.r.HandleFunc("/v1/events/pickup", s.pickup()).Methods("POST")
	s.r.HandleFunc("/v1/events/transit", s.transit()).Methods("POST")
	s.r.HandleFunc("/v1/events/deliver", s.deliver()).Methods("POST")
	s.r.HandleFunc("/v1/track/{trackingNumber}", s.track()).Methods("GET")
}

type GPS struct{ Lat, Lon float64 }

func (s *Server) pickup() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			ShipmentID string   `json:"shipmentId"`
			Lat        *float64 `json:"lat"`
			Lon        *float64 `json:"lon"`
			HubID      *string  `json:"hubId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ShipmentID == "" {
			log.Printf("bad_request service=tracking-svc request_id=%s route=/v1/events/pickup method=%s err=%v shipment_id=%q", RequestIDFromContext(r.Context()), r.Method, err, body.ShipmentID)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if err := s.repo.InsertEvent(r.Context(), repo.Event{ShipmentID: body.ShipmentID, Type: "PICKUP", Lat: body.Lat, Lon: body.Lon, HubID: body.HubID}); err != nil {
			log.Printf("event_insert_failed service=tracking-svc request_id=%s type=PICKUP shipment_id=%s err=%v", RequestIDFromContext(r.Context()), body.ShipmentID, err)
		} else {
			log.Printf("event_inserted service=tracking-svc request_id=%s type=PICKUP shipment_id=%s", RequestIDFromContext(r.Context()), body.ShipmentID)
		}
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{"result": "accepted"})
	}
}
func (s *Server) transit() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			ShipmentID string   `json:"shipmentId"`
			Lat        *float64 `json:"lat"`
			Lon        *float64 `json:"lon"`
			HubID      *string  `json:"hubId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ShipmentID == "" {
			log.Printf("bad_request service=tracking-svc request_id=%s route=/v1/events/transit method=%s err=%v shipment_id=%q", RequestIDFromContext(r.Context()), r.Method, err, body.ShipmentID)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if err := s.repo.InsertEvent(r.Context(), repo.Event{ShipmentID: body.ShipmentID, Type: "IN_TRANSIT", Lat: body.Lat, Lon: body.Lon, HubID: body.HubID}); err != nil {
			log.Printf("event_insert_failed service=tracking-svc request_id=%s type=IN_TRANSIT shipment_id=%s err=%v", RequestIDFromContext(r.Context()), body.ShipmentID, err)
		} else {
			log.Printf("event_inserted service=tracking-svc request_id=%s type=IN_TRANSIT shipment_id=%s", RequestIDFromContext(r.Context()), body.ShipmentID)
		}
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{"result": "accepted"})
	}
}

func (s *Server) deliver() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			ShipmentID string   `json:"shipmentId"`
			Lat        *float64 `json:"lat"`
			Lon        *float64 `json:"lon"`
			HubID      *string  `json:"hubId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ShipmentID == "" {
			log.Printf("bad_request service=tracking-svc request_id=%s route=/v1/events/deliver method=%s err=%v shipment_id=%q", RequestIDFromContext(r.Context()), r.Method, err, body.ShipmentID)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if err := s.repo.InsertEvent(r.Context(), repo.Event{ShipmentID: body.ShipmentID, Type: "DELIVERED", Lat: body.Lat, Lon: body.Lon, HubID: body.HubID}); err != nil {
			log.Printf("event_insert_failed service=tracking-svc request_id=%s type=DELIVERED shipment_id=%s err=%v", RequestIDFromContext(r.Context()), body.ShipmentID, err)
		} else {
			log.Printf("event_inserted service=tracking-svc request_id=%s type=DELIVERED shipment_id=%s", RequestIDFromContext(r.Context()), body.ShipmentID)
		}
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{"result": "accepted"})
	}
}

func (s *Server) track() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		shipmentID := vars["trackingNumber"]
		events, err := s.repo.GetEvents(r.Context(), shipmentID)
		if err != nil {
			log.Printf("db_error service=tracking-svc request_id=%s route=/v1/track/%s err=%v", RequestIDFromContext(r.Context()), shipmentID, err)
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		status := "CREATED"
		if len(events) > 0 {
			status = events[0].Type
		}
		log.Printf("track_query service=tracking-svc request_id=%s shipment_id=%s events=%d status=%s", RequestIDFromContext(r.Context()), shipmentID, len(events), status)
		json.NewEncoder(w).Encode(map[string]any{"trackingNumber": shipmentID, "status": status, "events": events})
	}
}

// Request ID context key
type ctxKey int

const reqIDKey ctxKey = iota

func genReqID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "rid-" + time.Now().UTC().Format("20060102T150405.000Z0700")
	}
	return hex.EncodeToString(b[:])
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

// Request logger middleware with status and latency
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
		if strings.HasPrefix(r.URL.Path, "/v1/track/") {
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

		log.Printf("request service=tracking-svc request_id=%s trace_id=%s span_id=%s method=%s path=%s status=%d duration_ms=%d response_bytes=%d client_ip=%s", rid, traceID, spanID, r.Method, r.URL.Path, rec.status, durMs, rec.size, ip)
	})

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

// Haversine distance (km)
func DistanceKm(a, b GPS) float64 {
	const R = 6371.0
	dLat := (b.Lat - a.Lat) * math.Pi / 180
	dLon := (b.Lon - a.Lon) * math.Pi / 180
	lat1 := a.Lat * math.Pi / 180
	lat2 := b.Lat * math.Pi / 180
	ae := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(lat1)*math.Cos(lat2)
	c := 2 * math.Atan2(math.Sqrt(ae), math.Sqrt(1-ae))
	return R * c
}
