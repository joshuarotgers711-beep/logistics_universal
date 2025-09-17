package repo

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"shipment-svc/internal/db"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
)

type Address struct {
	Line1      string `json:"line1"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postalCode"`
	Country    string `json:"country"`
}

type Package struct {
	WeightKg float64 `json:"weightKg"`
	LengthCm float64 `json:"lengthCm"`
	WidthCm  float64 `json:"widthCm"`
	HeightCm float64 `json:"heightCm"`
}

type Label struct {
	ID         string    `json:"id"`
	ShipmentID string    `json:"shipmentId"`
	Format     string    `json:"format"`
	StorageURL *string   `json:"storageUrl,omitempty"`
	Checksum   *string   `json:"checksum,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
}

type Shipment struct {
	ID           string    `json:"id"`
	ShipperID    string    `json:"shipperId"`
	ServiceLevel string    `json:"serviceLevel"`
	From         Address   `json:"from"`
	To           Address   `json:"to"`
	Value        float64   `json:"value"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"createdAt"`
}

type Repository struct{ DB *db.DB }

func New(d *db.DB) *Repository { return &Repository{DB: d} }

func (r *Repository) CreateShipment(ctx context.Context, s Shipment, pkgs []Package, userID string) (string, error) {
	fromB, _ := json.Marshal(s.From)
	toB, _ := json.Marshal(s.To)
	var id string
	tx, err := r.DB.Pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	row := tx.QueryRow(ctx, `INSERT INTO shipments(shipper_id, service_level, from_addr, to_addr, value, status) VALUES($1,$2,$3::jsonb,$4::jsonb,$5,$6) RETURNING id`, s.ShipperID, s.ServiceLevel, string(fromB), string(toB), s.Value, "CREATED")
	if err := row.Scan(&id); err != nil {
		return "", err
	}
	for _, p := range pkgs {
		_, err := tx.Exec(ctx, `INSERT INTO packages(shipment_id, weight_kg, length_cm, width_cm, height_cm) VALUES($1,$2,$3,$4,$5)`, id, p.WeightKg, p.LengthCm, p.WidthCm, p.HeightCm)
		if err != nil {
			return "", err
		}
	}
	_, _ = tx.Exec(ctx, `INSERT INTO audit_log(entity, entity_id, action, user_id, payload) VALUES('shipment', $1, 'CREATE', $2, $3::jsonb)`, id, userID, string(`{}`))
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return id, nil
}

func (r *Repository) GetShipment(ctx context.Context, id string) (*Shipment, []Package, error) {
	row := r.DB.Pool.QueryRow(ctx, `SELECT id, shipper_id, service_level, from_addr, to_addr, value, status, created_at FROM shipments WHERE id=$1`, id)
	var s Shipment
	var fromB, toB []byte
	if err := row.Scan(&s.ID, &s.ShipperID, &s.ServiceLevel, &fromB, &toB, &s.Value, &s.Status, &s.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	_ = json.Unmarshal(fromB, &s.From)
	_ = json.Unmarshal(toB, &s.To)
	rows, err := r.DB.Pool.Query(ctx, `SELECT weight_kg, length_cm, width_cm, height_cm FROM packages WHERE shipment_id=$1`, id)
	if err != nil {
		return &s, nil, err
	}
	defer rows.Close()
	pkgs := []Package{}
	for rows.Next() {
		var p Package
		if err := rows.Scan(&p.WeightKg, &p.LengthCm, &p.WidthCm, &p.HeightCm); err != nil {
			return &s, nil, err
		}
		pkgs = append(pkgs, p)
	}
	return &s, pkgs, nil
}

func (r *Repository) UpdateShipmentStatus(ctx context.Context, id, status, userID string) error {
	cmd, err := r.DB.Pool.Exec(ctx, `UPDATE shipments SET status=$2, updated_at=now() WHERE id=$1`, id, status)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return nil
	}
	_, _ = r.DB.Pool.Exec(ctx, `INSERT INTO audit_log(entity, entity_id, action, user_id) VALUES('shipment', $1, 'UPDATE_STATUS', $2)`, id, userID)
	return nil
}

func (r *Repository) InsertLabel(ctx context.Context, shipmentID, id, format string, storageURL, checksum *string) (string, error) {
	if id == "" {
		row := r.DB.Pool.QueryRow(ctx, `INSERT INTO labels(shipment_id, format, storage_url, checksum) VALUES($1,$2,$3,$4) RETURNING id`, shipmentID, format, storageURL, checksum)
		var nid string
		if err := row.Scan(&nid); err != nil {
			return "", err
		}
		return nid, nil
	}
	_, err := r.DB.Pool.Exec(ctx, `INSERT INTO labels(id, shipment_id, format, storage_url, checksum) VALUES($1,$2,$3,$4,$5)`, id, shipmentID, format, storageURL, checksum)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (r *Repository) ListLabels(ctx context.Context, shipmentID string) ([]Label, error) {
	rows, err := r.DB.Pool.Query(ctx, `SELECT id, shipment_id, format, storage_url, checksum, created_at FROM labels WHERE shipment_id=$1 ORDER BY created_at DESC`, shipmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Label{}
	for rows.Next() {
		var l Label
		if err := rows.Scan(&l.ID, &l.ShipmentID, &l.Format, &l.StorageURL, &l.Checksum, &l.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, nil
}

// CreateShipmentIdempotent attempts to create a shipment with an idempotency key.
// If a shipment already exists for the given key, it returns the existing id and replay=true.
func (r *Repository) CreateShipmentIdempotent(ctx context.Context, s Shipment, pkgs []Package, userID string, idemKey string) (id string, replay bool, err error) {
	if idemKey == "" {
		id, err = r.CreateShipment(ctx, s, pkgs, userID)
		return id, false, err
	}
	fromB, _ := json.Marshal(s.From)
	toB, _ := json.Marshal(s.To)
	tx, err := r.DB.Pool.Begin(ctx)
	if err != nil {
		return "", false, err
	}
	defer tx.Rollback(ctx)
	row := tx.QueryRow(ctx, `INSERT INTO shipments(shipper_id, service_level, from_addr, to_addr, value, status, idempotency_key) VALUES($1,$2,$3::jsonb,$4::jsonb,$5,$6,$7) RETURNING id`, s.ShipperID, s.ServiceLevel, string(fromB), string(toB), s.Value, "CREATED", idemKey)
	if scanErr := row.Scan(&id); scanErr != nil {
		var pgErr *pgconn.PgError
		if errors.As(scanErr, &pgErr) && pgErr.Code == "23505" {
			// Unique violation: fetch existing id
			row2 := r.DB.Pool.QueryRow(ctx, `SELECT id FROM shipments WHERE idempotency_key=$1`, idemKey)
			if err2 := row2.Scan(&id); err2 != nil {
				return "", false, err2
			}
			return id, true, nil
		}
		return "", false, scanErr
	}
	for _, p := range pkgs {
		if _, err := tx.Exec(ctx, `INSERT INTO packages(shipment_id, weight_kg, length_cm, width_cm, height_cm) VALUES($1,$2,$3,$4,$5)`, id, p.WeightKg, p.LengthCm, p.WidthCm, p.HeightCm); err != nil {
			return "", false, err
		}
	}
	_, _ = tx.Exec(ctx, `INSERT INTO audit_log(entity, entity_id, action, user_id, payload) VALUES('shipment', $1, 'CREATE', $2, $3::jsonb)`, id, userID, string(`{}`))
	if err := tx.Commit(ctx); err != nil {
		return "", false, err
	}
	return id, false, nil
}
