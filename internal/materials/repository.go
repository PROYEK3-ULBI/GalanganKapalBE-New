package materials

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when a lookup yields no rows.
var ErrNotFound = errors.New("material not found")

// ErrSKUConflict is returned when an insert/update would violate the unique SKU constraint.
var ErrSKUConflict = errors.New("sku already exists")

// Repository handles persistence operations for materials.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const materialColumns = `id, sku, name, category, unit,
		stock, min_stock, reorder_point, price, hazmat,
		heat_number, location, specifications,
		created_at, updated_at`

func scanMaterial(row pgx.Row) (*Material, error) {
	var m Material
	err := row.Scan(
		&m.ID, &m.SKU, &m.Name, &m.Category, &m.Unit,
		&m.Stock, &m.MinStock, &m.ReorderPoint, &m.Price, &m.Hazmat,
		&m.HeatNumber, &m.Location, &m.Specifications,
		&m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	m.Status = ComputeStatus(m.Stock, m.MinStock)
	return &m, nil
}

// List returns materials matching the given filters, ordered by name.
func (r *Repository) List(ctx context.Context, f ListFilters) ([]Material, error) {
	var (
		conds []string
		args  []any
	)

	if f.Search != "" {
		args = append(args, "%"+f.Search+"%")
		idx := len(args)
		conds = append(conds,
			fmt.Sprintf("(LOWER(sku) LIKE LOWER($%d) OR LOWER(name) LIKE LOWER($%d))", idx, idx))
	}
	if f.Category != "" {
		args = append(args, f.Category)
		conds = append(conds, fmt.Sprintf("category = $%d", len(args)))
	}
	if f.HazmatOnly {
		conds = append(conds, "hazmat = TRUE")
	}
	if f.LowStockOnly {
		conds = append(conds, "stock <= min_stock")
	}

	q := "SELECT " + materialColumns + " FROM materials"
	if len(conds) > 0 {
		q += " WHERE " + strings.Join(conds, " AND ")
	}
	q += " ORDER BY name ASC"

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query materials: %w", err)
	}
	defer rows.Close()

	out := make([]Material, 0)
	for rows.Next() {
		m, err := scanMaterial(rows)
		if err != nil {
			return nil, fmt.Errorf("scan material: %w", err)
		}
		out = append(out, *m)
	}
	return out, rows.Err()
}

// FindByID retrieves a single material by primary key.
func (r *Repository) FindByID(ctx context.Context, id string) (*Material, error) {
	q := "SELECT " + materialColumns + " FROM materials WHERE id = $1 LIMIT 1"
	row := r.pool.QueryRow(ctx, q, id)
	m, err := scanMaterial(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find material: %w", err)
	}
	return m, nil
}

// Create inserts a new material and returns the created record.
func (r *Repository) Create(ctx context.Context, m *Material) (*Material, error) {
	q := `INSERT INTO materials (
		sku, name, category, unit,
		stock, min_stock, reorder_point, price, hazmat,
		heat_number, location, specifications
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	RETURNING ` + materialColumns

	row := r.pool.QueryRow(ctx, q,
		m.SKU, m.Name, m.Category, m.Unit,
		m.Stock, m.MinStock, m.ReorderPoint, m.Price, m.Hazmat,
		m.HeatNumber, m.Location, m.Specifications,
	)
	created, err := scanMaterial(row)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrSKUConflict
		}
		return nil, fmt.Errorf("create material: %w", err)
	}
	return created, nil
}

// Update applies a partial update and returns the resulting record.
func (r *Repository) Update(ctx context.Context, id string, req UpdateRequest) (*Material, error) {
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
	if req.Category != nil {
		add("category", *req.Category)
	}
	if req.Unit != nil {
		add("unit", *req.Unit)
	}
	if req.Stock != nil {
		add("stock", *req.Stock)
	}
	if req.MinStock != nil {
		add("min_stock", *req.MinStock)
	}
	if req.ReorderPoint != nil {
		add("reorder_point", *req.ReorderPoint)
	}
	if req.Price != nil {
		add("price", *req.Price)
	}
	if req.Hazmat != nil {
		add("hazmat", *req.Hazmat)
	}
	if req.HeatNumber != nil {
		add("heat_number", *req.HeatNumber)
	}
	if req.Location != nil {
		add("location", *req.Location)
	}
	if req.Specifications != nil {
		add("specifications", *req.Specifications)
	}

	if len(sets) == 0 {
		// Nothing to update; just return current state.
		return r.FindByID(ctx, id)
	}

	args = append(args, id)
	q := fmt.Sprintf(
		"UPDATE materials SET %s WHERE id = $%d RETURNING %s",
		strings.Join(sets, ", "), len(args), materialColumns,
	)
	row := r.pool.QueryRow(ctx, q, args...)
	updated, err := scanMaterial(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("update material: %w", err)
	}
	return updated, nil
}

// Delete removes a material by ID.
func (r *Repository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM materials WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete material: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ListCategories returns distinct category values currently in use.
func (r *Repository) ListCategories(ctx context.Context) ([]string, error) {
	rows, err := r.pool.Query(ctx,
		"SELECT DISTINCT category FROM materials ORDER BY category ASC")
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()
	out := make([]string, 0)
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// isUniqueViolation reports whether the error is a PostgreSQL unique constraint violation.
func isUniqueViolation(err error) bool {
	// pgconn.PgError code 23505 = unique_violation. Use string contains to avoid
	// importing pgconn just for this check.
	return err != nil && strings.Contains(err.Error(), "23505")
}
