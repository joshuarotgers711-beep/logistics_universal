package repo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"identity-svc/internal/db"
)

type User struct {
	ID string
	Email string
	PasswordSalt string
	PasswordHash string
	Role string
	CreatedAt time.Time
}

type Repository struct{ DB *db.DB }

func New(d *db.DB) *Repository { return &Repository{DB: d} }

func (r *Repository) CountUsers(ctx context.Context) (int64, error) {
	row := r.DB.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`)
	var c int64; err := row.Scan(&c); return c, err
}

func (r *Repository) CreateUser(ctx context.Context, email, salt, passwordHash, role string) (string, error) {
	row := r.DB.Pool.QueryRow(ctx, `INSERT INTO users(email, password_salt, password_hash, role) VALUES($1,$2,$3,$4) RETURNING id`, email, salt, passwordHash, role)
	var id string; err := row.Scan(&id); return id, err
}

func (r *Repository) FindByEmail(ctx context.Context, email string) (*User, error) {
	row := r.DB.Pool.QueryRow(ctx, `SELECT id, email, password_salt, password_hash, role, created_at FROM users WHERE email=$1`, email)
	u := &User{}
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordSalt, &u.PasswordHash, &u.Role, &u.CreatedAt); err != nil { return nil, err }
	return u, nil
}

func HashPassword(salt, password string) string {
	sum := sha256.Sum256([]byte(salt + ":" + password))
	return hex.EncodeToString(sum[:])
}

