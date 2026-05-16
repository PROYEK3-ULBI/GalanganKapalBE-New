package activitylog

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) List(ctx context.Context, f ListFilters) ([]Entry, error) {
	var (
		conds []string
		args  []any
	)
	if f.UserID != "" {
		args = append(args, f.UserID)
		conds = append(conds, fmt.Sprintf("a.user_id = $%d", len(args)))
	}
	if f.Type != "" {
		args = append(args, f.Type)
		conds = append(conds, fmt.Sprintf("a.type = $%d", len(args)))
	}
	if f.Category != "" {
		args = append(args, f.Category)
		conds = append(conds, fmt.Sprintf("a.category = $%d", len(args)))
	}
	if f.StartDate != "" {
		args = append(args, f.StartDate)
		conds = append(conds, fmt.Sprintf("a.created_at >= $%d", len(args)))
	}
	if f.EndDate != "" {
		args = append(args, f.EndDate+" 23:59:59")
		conds = append(conds, fmt.Sprintf("a.created_at <= $%d", len(args)))
	}

	q := `SELECT a.id, a.action, a.detail, a.type,
	             a.user_id::text, u.name AS user_name,
	             a.resource_type, a.resource_id, a.category,
	             a.created_at
	      FROM activity_logs a
	      LEFT JOIN users u ON u.id = a.user_id`
	if len(conds) > 0 {
		q += " WHERE " + strings.Join(conds, " AND ")
	}
	q += " ORDER BY a.created_at DESC"
	if f.Limit > 0 {
		args = append(args, f.Limit)
		q += fmt.Sprintf(" LIMIT $%d", len(args))
	}

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query activity logs: %w", err)
	}
	defer rows.Close()

	out := make([]Entry, 0)
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		out = append(out, *e)
	}
	return out, rows.Err()
}

// Create inserts a new log entry. Empty user_id stored as NULL.
func (r *Repository) Create(ctx context.Context, in CreateInput) error {
	logType := strings.TrimSpace(in.Type)
	if logType == "" {
		logType = TypeInfo
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO activity_logs (action, detail, type, user_id, resource_type, resource_id, category)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`,
		in.Action, nullIfEmpty(in.Detail), logType,
		nullUUIDIfEmpty(in.UserID),
		nullIfEmpty(in.ResourceType), nullIfEmpty(in.ResourceID), nullIfEmpty(in.Category),
	)
	if err != nil {
		return fmt.Errorf("create activity log: %w", err)
	}
	return nil
}

func scanEntry(row pgx.Row) (*Entry, error) {
	var e Entry
	if err := row.Scan(
		&e.ID, &e.Action, &e.Detail, &e.Type,
		&e.UserID, &e.UserName,
		&e.ResourceType, &e.ResourceID, &e.Category,
		&e.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &e, nil
}

func nullIfEmpty(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func nullUUIDIfEmpty(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}
