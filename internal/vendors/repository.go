package vendors

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound     = errors.New("vendor not found")
	ErrNameConflict = errors.New("vendor name already exists")
	ErrInUse        = errors.New("vendor is referenced by purchase orders")
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const vendorColumns = `id, name, contact, phone, email, address, status, created_at, updated_at`

func scanVendor(row pgx.Row, withCount bool) (*Vendor, error) {
	var v Vendor
	dest := []any{
		&v.ID, &v.Name, &v.Contact, &v.Phone, &v.Email, &v.Address,
		&v.Status, &v.CreatedAt, &v.UpdatedAt,
	}
	if withCount {
		dest = append(dest, &v.POCount)
	}
	if err := row.Scan(dest...); err != nil {
		return nil, err
	}
	return &v, nil
}

// List returns vendors with their purchase order count, ordered by name.
func (r *Repository) List(ctx context.Context, f ListFilters) ([]Vendor, error) {
	var (
		conds []string
		args  []any
	)
	if f.Search != "" {
		args = append(args, "%"+f.Search+"%")
		conds = append(conds, fmt.Sprintf("LOWER(v.name) LIKE LOWER($%d)", len(args)))
	}
	if f.ActiveOnly {
		conds = append(conds, "v.status = 'active'")
	} else if f.Status != "" {
		args = append(args, f.Status)
		conds = append(conds, fmt.Sprintf("v.status = $%d", len(args)))
	}

	q := `SELECT v.id, v.name, v.contact, v.phone, v.email, v.address,
	             v.status, v.created_at, v.updated_at,
	             COALESCE(po.cnt, 0) AS po_count
	      FROM vendors v
	      LEFT JOIN (
	          SELECT vendor_id, COUNT(*) AS cnt FROM purchase_orders GROUP BY vendor_id
	      ) po ON po.vendor_id = v.id`
	if len(conds) > 0 {
		q += " WHERE " + strings.Join(conds, " AND ")
	}
	q += " ORDER BY v.name ASC"

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query vendors: %w", err)
	}
	defer rows.Close()

	out := make([]Vendor, 0)
	for rows.Next() {
		v, err := scanVendor(rows, true)
		if err != nil {
			return nil, fmt.Errorf("scan vendor: %w", err)
		}
		out = append(out, *v)
	}
	return out, rows.Err()
}

func (r *Repository) FindByID(ctx context.Context, id string) (*Vendor, error) {
	q := "SELECT " + vendorColumns + " FROM vendors WHERE id = $1 LIMIT 1"
	row := r.pool.QueryRow(ctx, q, id)
	v, err := scanVendor(row, false)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find vendor: %w", err)
	}
	return v, nil
}

func (r *Repository) Create(ctx context.Context, v *Vendor) (*Vendor, error) {
	q := `INSERT INTO vendors (name, contact, phone, email, address, status)
	      VALUES ($1, $2, $3, $4, $5, $6)
	      RETURNING ` + vendorColumns
	row := r.pool.QueryRow(ctx, q,
		v.Name, v.Contact, v.Phone, v.Email, v.Address, v.Status,
	)
	created, err := scanVendor(row, false)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrNameConflict
		}
		return nil, fmt.Errorf("create vendor: %w", err)
	}
	return created, nil
}

func (r *Repository) Update(ctx context.Context, id string, req UpdateRequest) (*Vendor, error) {
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
	if req.Contact != nil {
		add("contact", *req.Contact)
	}
	if req.Phone != nil {
		add("phone", *req.Phone)
	}
	if req.Email != nil {
		add("email", *req.Email)
	}
	if req.Address != nil {
		add("address", *req.Address)
	}
	if req.Status != nil {
		add("status", *req.Status)
	}

	if len(sets) == 0 {
		return r.FindByID(ctx, id)
	}

	args = append(args, id)
	q := fmt.Sprintf(
		"UPDATE vendors SET %s WHERE id = $%d RETURNING %s",
		strings.Join(sets, ", "), len(args), vendorColumns,
	)
	row := r.pool.QueryRow(ctx, q, args...)
	updated, err := scanVendor(row, false)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		if isUniqueViolation(err) {
			return nil, ErrNameConflict
		}
		return nil, fmt.Errorf("update vendor: %w", err)
	}
	return updated, nil
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM vendors WHERE id = $1", id)
	if err != nil {
		// FK constraint violation = vendor still referenced.
		if isForeignKeyViolation(err) {
			return ErrInUse
		}
		return fmt.Errorf("delete vendor: %w", err)
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
