package warehouselocations

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound     = errors.New("location not found")
	ErrCodeConflict = errors.New("location code already exists")
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const cols = `id, code, COALESCE(name, ''), COALESCE(type, ''), capacity, COALESCE(notes, ''), status, created_at, updated_at`

func scan(row pgx.Row) (*Location, error) {
	var loc Location
	if err := row.Scan(
		&loc.ID, &loc.Code, &loc.Name, &loc.Type, &loc.Capacity,
		&loc.Notes, &loc.Status, &loc.CreatedAt, &loc.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &loc, nil
}

func (r *Repository) List(ctx context.Context, f ListFilters) ([]Location, error) {
	var (
		conds []string
		args  []any
	)
	if f.Search != "" {
		args = append(args, "%"+f.Search+"%")
		idx := len(args)
		conds = append(conds, fmt.Sprintf("(LOWER(code) LIKE LOWER($%d) OR LOWER(name) LIKE LOWER($%d))", idx, idx))
	}
	if f.Type != "" {
		args = append(args, f.Type)
		conds = append(conds, fmt.Sprintf("type = $%d", len(args)))
	}
	if f.ActiveOnly {
		conds = append(conds, "status = 'active'")
	} else if f.Status != "" {
		args = append(args, f.Status)
		conds = append(conds, fmt.Sprintf("status = $%d", len(args)))
	}

	q := "SELECT " + cols + " FROM warehouse_locations"
	if len(conds) > 0 {
		q += " WHERE " + strings.Join(conds, " AND ")
	}
	q += " ORDER BY code ASC"

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query locations: %w", err)
	}
	defer rows.Close()

	out := make([]Location, 0)
	for rows.Next() {
		loc, err := scan(rows)
		if err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		out = append(out, *loc)
	}
	return out, rows.Err()
}

func (r *Repository) FindByID(ctx context.Context, id string) (*Location, error) {
	q := "SELECT " + cols + " FROM warehouse_locations WHERE id = $1 LIMIT 1"
	row := r.pool.QueryRow(ctx, q, id)
	loc, err := scan(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find: %w", err)
	}
	return loc, nil
}

func (r *Repository) Create(ctx context.Context, loc *Location) (*Location, error) {
	q := `INSERT INTO warehouse_locations (code, name, type, capacity, notes, status)
	      VALUES ($1, $2, $3, $4, $5, $6)
	      RETURNING ` + cols
	row := r.pool.QueryRow(ctx, q,
		loc.Code, nullIfEmpty(loc.Name), nullIfEmpty(loc.Type),
		loc.Capacity, nullIfEmpty(loc.Notes), loc.Status,
	)
	created, err := scan(row)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrCodeConflict
		}
		return nil, fmt.Errorf("create: %w", err)
	}
	return created, nil
}

func (r *Repository) Update(ctx context.Context, id string, req UpdateRequest) (*Location, error) {
	var (
		sets []string
		args []any
	)
	add := func(col string, val any) {
		args = append(args, val)
		sets = append(sets, fmt.Sprintf("%s = $%d", col, len(args)))
	}
	if req.Name != nil {
		add("name", nullIfEmpty(strings.TrimSpace(*req.Name)))
	}
	if req.Type != nil {
		add("type", nullIfEmpty(strings.TrimSpace(*req.Type)))
	}
	if req.Capacity != nil {
		if *req.Capacity > 0 {
			add("capacity", *req.Capacity)
		} else {
			add("capacity", nil)
		}
	}
	if req.Notes != nil {
		add("notes", nullIfEmpty(*req.Notes))
	}
	if req.Status != nil {
		add("status", *req.Status)
	}
	if len(sets) == 0 {
		return r.FindByID(ctx, id)
	}

	args = append(args, id)
	q := fmt.Sprintf(
		"UPDATE warehouse_locations SET %s WHERE id = $%d RETURNING %s",
		strings.Join(sets, ", "), len(args), cols,
	)
	row := r.pool.QueryRow(ctx, q, args...)
	updated, err := scan(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("update: %w", err)
	}
	return updated, nil
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM warehouse_locations WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete: %w", err)
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
