package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrUserNotFound is returned when no user matches the lookup criteria.
var ErrUserNotFound = errors.New("user not found")

// Repository handles persistence operations for users.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const userColumns = `id, email, password_hash, name, role,
		COALESCE(avatar, ''), COALESCE(department, ''),
		COALESCE(phone, ''), COALESCE(position, ''),
		COALESCE(notification_preferences, '{}'::jsonb)::text,
		status, last_login_at, created_at, updated_at`

func scanUser(row pgx.Row) (*User, error) {
	var (
		u        User
		prefsRaw string
	)
	err := row.Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.Role,
		&u.Avatar, &u.Department,
		&u.Phone, &u.Position, &prefsRaw,
		&u.Status, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if prefsRaw != "" && prefsRaw != "{}" {
		if err := json.Unmarshal([]byte(prefsRaw), &u.NotificationPreferences); err != nil {
			return nil, fmt.Errorf("decode preferences: %w", err)
		}
	} else {
		u.NotificationPreferences = map[string]any{}
	}
	return &u, nil
}

// FindByEmail looks up a user by email address. Email comparison is case-insensitive.
// Returns ErrUserNotFound when no row matches.
func (r *Repository) FindByEmail(ctx context.Context, email string) (*User, error) {
	q := `SELECT ` + userColumns + ` FROM users WHERE LOWER(email) = LOWER($1) LIMIT 1`
	row := r.pool.QueryRow(ctx, q, email)
	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("find user by email: %w", err)
	}
	return u, nil
}

// FindByID looks up a user by primary key.
func (r *Repository) FindByID(ctx context.Context, id string) (*User, error) {
	q := `SELECT ` + userColumns + ` FROM users WHERE id = $1 LIMIT 1`
	row := r.pool.QueryRow(ctx, q, id)
	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("find user by id: %w", err)
	}
	return u, nil
}

// TouchLastLogin updates the last_login_at timestamp to NOW().
func (r *Repository) TouchLastLogin(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `UPDATE users SET last_login_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("touch last login: %w", err)
	}
	return nil
}

// UpdateProfile applies partial profile updates from a self-service Settings page.
// Email and role are intentionally not updatable here.
func (r *Repository) UpdateProfile(ctx context.Context, id string, req UpdateProfileRequest) (*User, error) {
	var (
		sets []string
		args []any
	)
	add := func(col string, val any) {
		args = append(args, val)
		sets = append(sets, fmt.Sprintf("%s = $%d", col, len(args)))
	}

	if req.Name != nil {
		add("name", strings.TrimSpace(*req.Name))
	}
	if req.Department != nil {
		add("department", nullIfEmpty(*req.Department))
	}
	if req.Phone != nil {
		add("phone", nullIfEmpty(*req.Phone))
	}
	if req.Position != nil {
		add("position", nullIfEmpty(*req.Position))
	}
	if req.Avatar != nil {
		add("avatar", nullIfEmpty(strings.ToUpper(strings.TrimSpace(*req.Avatar))))
	}

	if len(sets) == 0 {
		return r.FindByID(ctx, id)
	}

	args = append(args, id)
	q := fmt.Sprintf(
		"UPDATE users SET %s WHERE id = $%d RETURNING %s",
		strings.Join(sets, ", "), len(args), userColumns,
	)
	row := r.pool.QueryRow(ctx, q, args...)
	updated, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("update profile: %w", err)
	}
	return updated, nil
}

// UpdatePassword writes a new bcrypt hash for the given user.
func (r *Repository) UpdatePassword(ctx context.Context, id, hash string) error {
	tag, err := r.pool.Exec(ctx, "UPDATE users SET password_hash = $1 WHERE id = $2", hash, id)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

// UpdateNotificationPreferences replaces the JSONB preferences blob for a user.
func (r *Repository) UpdateNotificationPreferences(ctx context.Context, id string, prefs map[string]any) (*User, error) {
	// pgx requires explicit JSON encoding for map -> jsonb columns.
	jsonPrefs, err := json.Marshal(prefs)
	if err != nil {
		return nil, fmt.Errorf("encode preferences: %w", err)
	}
	q := `UPDATE users SET notification_preferences = $1::jsonb WHERE id = $2 RETURNING ` + userColumns
	row := r.pool.QueryRow(ctx, q, string(jsonPrefs), id)
	updated, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("update notification preferences: %w", err)
	}
	return updated, nil
}

func nullIfEmpty(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}
