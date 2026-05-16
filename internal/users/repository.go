package users

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound      = errors.New("user not found")
	ErrEmailConflict = errors.New("email already exists")
	ErrInUse         = errors.New("user is referenced by other records")
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const userColumns = `id, email, name, role,
		COALESCE(avatar, ''), COALESCE(department, ''),
		status, last_login_at, created_at, updated_at`

func scanUser(row pgx.Row) (*User, error) {
	var u User
	err := row.Scan(
		&u.ID, &u.Email, &u.Name, &u.Role,
		&u.Avatar, &u.Department,
		&u.Status, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) List(ctx context.Context, f ListFilters) ([]User, error) {
	var (
		conds []string
		args  []any
	)
	if f.Search != "" {
		args = append(args, "%"+f.Search+"%")
		idx := len(args)
		conds = append(conds, fmt.Sprintf("(LOWER(email) LIKE LOWER($%d) OR LOWER(name) LIKE LOWER($%d))", idx, idx))
	}
	if f.Role != "" {
		args = append(args, f.Role)
		conds = append(conds, fmt.Sprintf("role = $%d", len(args)))
	}
	if f.Status != "" {
		args = append(args, f.Status)
		conds = append(conds, fmt.Sprintf("status = $%d", len(args)))
	}
	if f.Department != "" {
		args = append(args, f.Department)
		conds = append(conds, fmt.Sprintf("department = $%d", len(args)))
	}

	q := "SELECT " + userColumns + " FROM users"
	if len(conds) > 0 {
		q += " WHERE " + strings.Join(conds, " AND ")
	}
	q += " ORDER BY name ASC"

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query users: %w", err)
	}
	defer rows.Close()

	out := make([]User, 0)
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		out = append(out, *u)
	}
	return out, rows.Err()
}

func (r *Repository) FindByID(ctx context.Context, id string) (*User, error) {
	q := "SELECT " + userColumns + " FROM users WHERE id = $1 LIMIT 1"
	row := r.pool.QueryRow(ctx, q, id)
	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find user: %w", err)
	}
	return u, nil
}

func (r *Repository) GetCurrentStatus(ctx context.Context, id string) (string, error) {
	var status string
	err := r.pool.QueryRow(ctx, "SELECT status FROM users WHERE id = $1", id).Scan(&status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	return status, nil
}

// Create inserts a new user. passwordHash must be a bcrypt hash.
func (r *Repository) Create(ctx context.Context, u *User, passwordHash string) (*User, error) {
	q := `INSERT INTO users (email, password_hash, name, role, avatar, department, status)
	      VALUES ($1, $2, $3, $4, $5, $6, $7)
	      RETURNING ` + userColumns
	row := r.pool.QueryRow(ctx, q,
		strings.ToLower(strings.TrimSpace(u.Email)),
		passwordHash, u.Name, u.Role,
		nullIfEmpty(u.Avatar), nullIfEmpty(u.Department),
		u.Status,
	)
	created, err := scanUser(row)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrEmailConflict
		}
		return nil, fmt.Errorf("create user: %w", err)
	}
	return created, nil
}

func (r *Repository) Update(ctx context.Context, id string, req UpdateRequest) (*User, error) {
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
	if req.Role != nil {
		add("role", *req.Role)
	}
	if req.Avatar != nil {
		add("avatar", nullIfEmpty(strings.TrimSpace(*req.Avatar)))
	}
	if req.Department != nil {
		add("department", nullIfEmpty(strings.TrimSpace(*req.Department)))
	}
	if req.Status != nil {
		add("status", *req.Status)
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
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("update user: %w", err)
	}
	return updated, nil
}

// SetStatus is a quick-path UPDATE used by the toggle endpoint.
func (r *Repository) SetStatus(ctx context.Context, id, status string) (*User, error) {
	row := r.pool.QueryRow(ctx,
		"UPDATE users SET status = $1 WHERE id = $2 RETURNING "+userColumns,
		status, id,
	)
	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("set status: %w", err)
	}
	return u, nil
}

// SetPassword updates only the password_hash for the given user.
func (r *Repository) SetPassword(ctx context.Context, id, hash string) error {
	tag, err := r.pool.Exec(ctx, "UPDATE users SET password_hash = $1 WHERE id = $2", hash, id)
	if err != nil {
		return fmt.Errorf("set password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM users WHERE id = $1", id)
	if err != nil {
		// FK from material_requests.requester_id is RESTRICT — referenced users cannot be deleted.
		if isForeignKeyViolation(err) {
			return ErrInUse
		}
		return fmt.Errorf("delete user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func nullIfEmpty(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "23505")
}

func isForeignKeyViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "23503")
}
