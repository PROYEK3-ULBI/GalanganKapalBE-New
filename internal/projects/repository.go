package projects

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound     = errors.New("project not found")
	ErrCodeConflict = errors.New("project code already exists")
	// ErrInUse is reserved for future when transactions reference projects (FK protect).
	ErrInUse = errors.New("project is referenced by transactions")
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const projectColumns = `id, code, name, type, status, completion_pct,
		start_date, target_date, notes, created_by::text,
		created_at, updated_at`

func scanProject(row pgx.Row) (*Project, error) {
	var (
		p          Project
		startDate  *time.Time
		targetDate *time.Time
	)
	err := row.Scan(
		&p.ID, &p.Code, &p.Name, &p.Type, &p.Status, &p.CompletionPct,
		&startDate, &targetDate, &p.Notes, &p.CreatedBy,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if startDate != nil {
		s := startDate.Format("2006-01-02")
		p.StartDate = &s
	}
	if targetDate != nil {
		t := targetDate.Format("2006-01-02")
		p.TargetDate = &t
	}
	return &p, nil
}

func (r *Repository) List(ctx context.Context, f ListFilters) ([]Project, error) {
	var (
		conds []string
		args  []any
	)
	if f.Search != "" {
		args = append(args, "%"+f.Search+"%")
		idx := len(args)
		conds = append(conds, fmt.Sprintf("(LOWER(code) LIKE LOWER($%d) OR LOWER(name) LIKE LOWER($%d))", idx, idx))
	}
	if f.Status != "" {
		args = append(args, f.Status)
		conds = append(conds, fmt.Sprintf("status = $%d", len(args)))
	}
	if f.Type != "" {
		args = append(args, f.Type)
		conds = append(conds, fmt.Sprintf("type = $%d", len(args)))
	}
	if f.ActiveOnly {
		conds = append(conds, "status NOT IN ('Completed', 'Cancelled')")
	}

	q := "SELECT " + projectColumns + " FROM projects"
	if len(conds) > 0 {
		q += " WHERE " + strings.Join(conds, " AND ")
	}
	q += " ORDER BY code ASC"

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query projects: %w", err)
	}
	defer rows.Close()

	out := make([]Project, 0)
	for rows.Next() {
		p, err := scanProject(rows)
		if err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func (r *Repository) FindByID(ctx context.Context, id string) (*Project, error) {
	q := "SELECT " + projectColumns + " FROM projects WHERE id = $1 LIMIT 1"
	row := r.pool.QueryRow(ctx, q, id)
	p, err := scanProject(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find project: %w", err)
	}
	return p, nil
}

func (r *Repository) Create(ctx context.Context, p *Project, createdByID *string) (*Project, error) {
	q := `INSERT INTO projects (code, name, type, status, completion_pct,
		    start_date, target_date, notes, created_by)
		  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		  RETURNING ` + projectColumns
	row := r.pool.QueryRow(ctx, q,
		p.Code, p.Name, p.Type, p.Status, p.CompletionPct,
		p.StartDate, p.TargetDate, p.Notes, createdByID,
	)
	created, err := scanProject(row)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrCodeConflict
		}
		return nil, fmt.Errorf("create project: %w", err)
	}
	return created, nil
}

func (r *Repository) Update(ctx context.Context, id string, req UpdateRequest) (*Project, error) {
	var (
		sets []string
		args []any
	)
	add := func(col string, val any) {
		args = append(args, val)
		sets = append(sets, fmt.Sprintf("%s = $%d", col, len(args)))
	}

	if req.Name != nil {
		add("name", *req.Name)
	}
	if req.Type != nil {
		add("type", *req.Type)
	}
	if req.Status != nil {
		add("status", *req.Status)
	}
	if req.CompletionPct != nil {
		add("completion_pct", *req.CompletionPct)
	}
	if req.StartDate != nil {
		// empty string clears the value
		if *req.StartDate == "" {
			add("start_date", nil)
		} else {
			add("start_date", *req.StartDate)
		}
	}
	if req.TargetDate != nil {
		if *req.TargetDate == "" {
			add("target_date", nil)
		} else {
			add("target_date", *req.TargetDate)
		}
	}
	if req.Notes != nil {
		add("notes", *req.Notes)
	}

	if len(sets) == 0 {
		return r.FindByID(ctx, id)
	}

	args = append(args, id)
	q := fmt.Sprintf(
		"UPDATE projects SET %s WHERE id = $%d RETURNING %s",
		strings.Join(sets, ", "), len(args), projectColumns,
	)
	row := r.pool.QueryRow(ctx, q, args...)
	updated, err := scanProject(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("update project: %w", err)
	}
	return updated, nil
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM projects WHERE id = $1", id)
	if err != nil {
		if isForeignKeyViolation(err) {
			return ErrInUse
		}
		return fmt.Errorf("delete project: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "23505")
}

func isForeignKeyViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "23503")
}
