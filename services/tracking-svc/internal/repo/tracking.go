package repo

import (
	"context"
	"encoding/json"
	"time"

	"tracking-svc/internal/db"
)

type Event struct {
	ID         string         `json:"id"`
	ShipmentID string         `json:"shipmentId"`
	Type       string         `json:"type"`
	Ts         time.Time      `json:"ts"`
	Lat        *float64       `json:"lat,omitempty"`
	Lon        *float64       `json:"lon,omitempty"`
	HubID      *string        `json:"hubId,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

type Repository struct{ DB *db.DB }

func New(d *db.DB) *Repository { return &Repository{DB: d} }

func (r *Repository) InsertEvent(ctx context.Context, e Event) error {
	// Handle optional fields correctly: if Ts is zero, pass NULL so DB defaults to now();
	// if Metadata is nil, pass NULL instead of empty string to avoid JSON parse error.
	var ts *time.Time
	if !e.Ts.IsZero() {
		ts = &e.Ts
	}
	var meta any
	if e.Metadata != nil {
		b, _ := json.Marshal(e.Metadata)
		meta = b
	} else {
		meta = nil
	}
	_, err := r.DB.Pool.Exec(ctx, `INSERT INTO tracking_events(shipment_id, type, ts, lat, lon, hub_id, metadata) VALUES($1,$2,COALESCE($3::timestamptz, now()),$4,$5,$6,$7)`, e.ShipmentID, e.Type, ts, e.Lat, e.Lon, e.HubID, meta)
	return err
}

func (r *Repository) GetEvents(ctx context.Context, shipmentID string) ([]Event, error) {
	rows, err := r.DB.Pool.Query(ctx, `SELECT id, shipment_id, type, ts, lat, lon, hub_id, metadata FROM tracking_events WHERE shipment_id=$1 ORDER BY ts DESC LIMIT 100`, shipmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Event{}
	for rows.Next() {
		var e Event
		var meta []byte
		if err := rows.Scan(&e.ID, &e.ShipmentID, &e.Type, &e.Ts, &e.Lat, &e.Lon, &e.HubID, &meta); err != nil {
			return nil, err
		}
		if len(meta) > 0 {
			_ = json.Unmarshal(meta, &e.Metadata)
		}
		out = append(out, e)
	}
	return out, nil
}
