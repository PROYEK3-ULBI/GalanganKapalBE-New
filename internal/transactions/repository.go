package transactions

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
	ErrNotFound          = errors.New("transaction not found")
	ErrInsufficientStock = errors.New("insufficient stock")
	ErrMaterialNotFound  = errors.New("material not found")
	ErrPOItemNotFound    = errors.New("purchase order item not found")
	ErrPOItemMaterialMismatch = errors.New("po item does not match material")
	ErrReceiveExceedsOrdered  = errors.New("received qty exceeds ordered qty")
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// List returns transactions with all denormalised display fields populated.
// Sorted by transaction_date DESC, with optional limit.
func (r *Repository) List(ctx context.Context, f ListFilters) ([]Transaction, error) {
	var (
		conds []string
		args  []any
	)
	add := func(col string, val any) {
		args = append(args, val)
		conds = append(conds, fmt.Sprintf("%s = $%d", col, len(args)))
	}
	if f.Type != "" {
		add("t.type", f.Type)
	}
	if f.MaterialID != "" {
		add("t.material_id", f.MaterialID)
	}
	if f.ProjectID != "" {
		add("t.project_id", f.ProjectID)
	}
	if f.VendorID != "" {
		add("t.vendor_id", f.VendorID)
	}
	if f.UserID != "" {
		add("t.user_id", f.UserID)
	}
	if f.StartDate != "" {
		args = append(args, f.StartDate)
		conds = append(conds, fmt.Sprintf("t.transaction_date >= $%d", len(args)))
	}
	if f.EndDate != "" {
		args = append(args, f.EndDate+" 23:59:59")
		conds = append(conds, fmt.Sprintf("t.transaction_date <= $%d", len(args)))
	}

	q := `SELECT t.id, t.transaction_no, t.type,
	             t.material_id::text, m.sku, m.name, m.unit, t.qty,
	             t.project_id::text, p.code,
	             t.vendor_id::text, v.name AS vendor_name,
	             t.purchase_order_id::text, po.po_number,
	             t.po_item_id::text,
	             t.user_id::text, u.name AS user_name,
	             t.heat_number, t.notes,
	             t.transaction_date, t.created_at
	      FROM transactions t
	      JOIN materials m ON m.id = t.material_id
	      LEFT JOIN projects p        ON p.id = t.project_id
	      LEFT JOIN vendors v         ON v.id = t.vendor_id
	      LEFT JOIN purchase_orders po ON po.id = t.purchase_order_id
	      LEFT JOIN users u           ON u.id = t.user_id`
	if len(conds) > 0 {
		q += " WHERE " + strings.Join(conds, " AND ")
	}
	q += " ORDER BY t.transaction_date DESC, t.created_at DESC"

	if f.Limit > 0 {
		args = append(args, f.Limit)
		q += fmt.Sprintf(" LIMIT $%d", len(args))
	}

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query transactions: %w", err)
	}
	defer rows.Close()

	out := make([]Transaction, 0)
	for rows.Next() {
		t, err := scanTransaction(rows)
		if err != nil {
			return nil, fmt.Errorf("scan transaction: %w", err)
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

// FindByID retrieves a single transaction.
func (r *Repository) FindByID(ctx context.Context, id string) (*Transaction, error) {
	q := `SELECT t.id, t.transaction_no, t.type,
	             t.material_id::text, m.sku, m.name, m.unit, t.qty,
	             t.project_id::text, p.code,
	             t.vendor_id::text, v.name AS vendor_name,
	             t.purchase_order_id::text, po.po_number,
	             t.po_item_id::text,
	             t.user_id::text, u.name AS user_name,
	             t.heat_number, t.notes,
	             t.transaction_date, t.created_at
	      FROM transactions t
	      JOIN materials m ON m.id = t.material_id
	      LEFT JOIN projects p        ON p.id = t.project_id
	      LEFT JOIN vendors v         ON v.id = t.vendor_id
	      LEFT JOIN purchase_orders po ON po.id = t.purchase_order_id
	      LEFT JOIN users u           ON u.id = t.user_id
	      WHERE t.id = $1
	      LIMIT 1`
	row := r.pool.QueryRow(ctx, q, id)
	t, err := scanTransaction(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find transaction: %w", err)
	}
	return t, nil
}

// NextTransactionNo returns the next sequential txn number for the current year:
// e.g. "TRX-2026-0042". Computed inside the calling tx for atomicity-friendliness,
// but here we read with the pool — collisions are protected by the UNIQUE constraint
// (caller can retry once on conflict).
func (r *Repository) nextTransactionNoTx(ctx context.Context, tx pgx.Tx) (string, error) {
	year := time.Now().Year()
	prefix := fmt.Sprintf("TRX-%d-", year)
	var maxSuffix *int
	err := tx.QueryRow(ctx, `
		SELECT MAX(NULLIF(REGEXP_REPLACE(transaction_no, '^TRX-\d{4}-', ''), '')::int)
		FROM transactions
		WHERE transaction_no LIKE $1
	`, prefix+"%").Scan(&maxSuffix)
	if err != nil {
		return "", fmt.Errorf("next transaction no: %w", err)
	}
	next := 1
	if maxSuffix != nil {
		next = *maxSuffix + 1
	}
	return fmt.Sprintf("%s%04d", prefix, next), nil
}

// scanTransaction scans a row from the joined SELECT into a Transaction.
func scanTransaction(row pgx.Row) (*Transaction, error) {
	var t Transaction
	err := row.Scan(
		&t.ID, &t.TransactionNo, &t.Type,
		&t.MaterialID, &t.SKU, &t.MaterialName, &t.Unit, &t.Qty,
		&t.ProjectID, &t.ProjectCode,
		&t.VendorID, &t.VendorName,
		&t.PurchaseOrderID, &t.PONumber,
		&t.POItemID,
		&t.UserID, &t.UserName,
		&t.HeatNumber, &t.Notes,
		&t.TransactionDate, &t.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
