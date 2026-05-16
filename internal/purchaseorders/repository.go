package purchaseorders

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
	ErrNotFound       = errors.New("purchase order not found")
	ErrPONumberExists = errors.New("po number already exists")
	ErrVendorNotFound = errors.New("vendor not found")
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// List returns purchase orders with their items, ordered by date desc.
func (r *Repository) List(ctx context.Context, f ListFilters) ([]PurchaseOrder, error) {
	var (
		conds []string
		args  []any
	)
	if f.Search != "" {
		args = append(args, "%"+f.Search+"%")
		idx := len(args)
		conds = append(conds, fmt.Sprintf("(LOWER(po.po_number) LIKE LOWER($%d) OR LOWER(v.name) LIKE LOWER($%d))", idx, idx))
	}
	if f.Status != "" {
		args = append(args, f.Status)
		conds = append(conds, fmt.Sprintf("po.status = $%d", len(args)))
	}
	if f.VendorID != "" {
		args = append(args, f.VendorID)
		conds = append(conds, fmt.Sprintf("po.vendor_id = $%d", len(args)))
	}

	q := `SELECT po.id, po.po_number, po.vendor_id, v.name AS vendor_name,
	             po.order_date, po.status, po.notes, po.created_by::text,
	             po.created_at, po.updated_at
	      FROM purchase_orders po
	      JOIN vendors v ON v.id = po.vendor_id`
	if len(conds) > 0 {
		q += " WHERE " + strings.Join(conds, " AND ")
	}
	q += " ORDER BY po.order_date DESC, po.created_at DESC"

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query purchase orders: %w", err)
	}
	defer rows.Close()

	var (
		orders []PurchaseOrder
		ids    []string
	)
	for rows.Next() {
		po, err := scanHeader(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, *po)
		ids = append(ids, po.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(orders) == 0 {
		return []PurchaseOrder{}, nil
	}

	itemsByPO, err := r.fetchItemsForOrders(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range orders {
		orders[i].Items = itemsByPO[orders[i].ID]
		if orders[i].Items == nil {
			orders[i].Items = []Item{}
		}
	}
	return orders, nil
}

func (r *Repository) FindByID(ctx context.Context, id string) (*PurchaseOrder, error) {
	q := `SELECT po.id, po.po_number, po.vendor_id, v.name AS vendor_name,
	             po.order_date, po.status, po.notes, po.created_by::text,
	             po.created_at, po.updated_at
	      FROM purchase_orders po
	      JOIN vendors v ON v.id = po.vendor_id
	      WHERE po.id = $1
	      LIMIT 1`
	row := r.pool.QueryRow(ctx, q, id)
	po, err := scanHeader(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find po: %w", err)
	}
	itemsByPO, err := r.fetchItemsForOrders(ctx, []string{po.ID})
	if err != nil {
		return nil, err
	}
	po.Items = itemsByPO[po.ID]
	if po.Items == nil {
		po.Items = []Item{}
	}
	return po, nil
}

// Create inserts the header and all items in a single transaction.
func (r *Repository) Create(ctx context.Context, po *PurchaseOrder, createdByID *string) (*PurchaseOrder, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	headerQ := `INSERT INTO purchase_orders (po_number, vendor_id, order_date, status, notes, created_by)
	            VALUES ($1, $2, $3, $4, $5, $6)
	            RETURNING id, po_number, vendor_id, order_date, status, notes,
	                      created_by::text, created_at, updated_at`
	row := tx.QueryRow(ctx, headerQ,
		po.PONumber, po.VendorID, po.OrderDate, po.Status, po.Notes, createdByID,
	)
	var (
		out       PurchaseOrder
		orderDate time.Time
	)
	err = row.Scan(
		&out.ID, &out.PONumber, &out.VendorID, &orderDate, &out.Status,
		&out.Notes, &out.CreatedBy, &out.CreatedAt, &out.UpdatedAt,
	)
	if err != nil {
		switch {
		case isUniqueViolation(err):
			return nil, ErrPONumberExists
		case isForeignKeyViolation(err):
			return nil, ErrVendorNotFound
		}
		return nil, fmt.Errorf("insert po header: %w", err)
	}
	out.OrderDate = orderDate.Format("2006-01-02")

	for _, item := range po.Items {
		var inserted Item
		err := tx.QueryRow(ctx,
			`INSERT INTO purchase_order_items (purchase_order_id, material_id, ordered_qty, received_qty, unit_price, notes)
			 VALUES ($1, $2, $3, 0, $4, $5)
			 RETURNING id, material_id, ordered_qty, received_qty, unit_price, notes`,
			out.ID, item.MaterialID, item.Ordered, item.UnitPrice, item.Notes,
		).Scan(&inserted.ID, &inserted.MaterialID, &inserted.Ordered, &inserted.Received, &inserted.UnitPrice, &inserted.Notes)
		if err != nil {
			if isForeignKeyViolation(err) {
				return nil, fmt.Errorf("material %s: %w", item.MaterialID, ErrMaterialNotFound)
			}
			return nil, fmt.Errorf("insert po item: %w", err)
		}
		out.Items = append(out.Items, inserted)
	}

	// Look up vendor name + enrich item details for response.
	if err := tx.QueryRow(ctx, "SELECT name FROM vendors WHERE id = $1", out.VendorID).Scan(&out.VendorName); err != nil {
		return nil, fmt.Errorf("fetch vendor name: %w", err)
	}
	if err := enrichItemDetailsTx(ctx, tx, out.Items); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	if out.Items == nil {
		out.Items = []Item{}
	}
	return &out, nil
}

// Update applies partial updates to the header.
func (r *Repository) Update(ctx context.Context, id string, req UpdateRequest) (*PurchaseOrder, error) {
	var (
		sets []string
		args []any
	)
	if req.VendorID != nil {
		args = append(args, *req.VendorID)
		sets = append(sets, fmt.Sprintf("vendor_id = $%d", len(args)))
	}
	if req.Date != nil {
		args = append(args, *req.Date)
		sets = append(sets, fmt.Sprintf("order_date = $%d", len(args)))
	}
	if req.Status != nil {
		args = append(args, *req.Status)
		sets = append(sets, fmt.Sprintf("status = $%d", len(args)))
	}
	if req.Notes != nil {
		args = append(args, *req.Notes)
		sets = append(sets, fmt.Sprintf("notes = $%d", len(args)))
	}
	if len(sets) == 0 {
		return r.FindByID(ctx, id)
	}
	args = append(args, id)
	q := fmt.Sprintf(
		"UPDATE purchase_orders SET %s WHERE id = $%d",
		strings.Join(sets, ", "), len(args),
	)
	tag, err := r.pool.Exec(ctx, q, args...)
	if err != nil {
		if isForeignKeyViolation(err) {
			return nil, ErrVendorNotFound
		}
		return nil, fmt.Errorf("update po: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return r.FindByID(ctx, id)
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM purchase_orders WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete po: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// NextPONumber returns the next sequential PO number for the current year,
// e.g. "PO-2026-0042". Reads the highest existing number for the year.
func (r *Repository) NextPONumber(ctx context.Context) (string, error) {
	year := time.Now().Year()
	prefix := fmt.Sprintf("PO-%d-", year)
	var maxSuffix *int
	err := r.pool.QueryRow(ctx, `
		SELECT MAX(NULLIF(REGEXP_REPLACE(po_number, '^PO-\d{4}-', ''), '')::int)
		FROM purchase_orders
		WHERE po_number LIKE $1
	`, prefix+"%").Scan(&maxSuffix)
	if err != nil {
		return "", fmt.Errorf("next po number: %w", err)
	}
	next := 1
	if maxSuffix != nil {
		next = *maxSuffix + 1
	}
	return fmt.Sprintf("%s%04d", prefix, next), nil
}

// fetchItemsForOrders loads items for the given PO IDs and groups them.
func (r *Repository) fetchItemsForOrders(ctx context.Context, poIDs []string) (map[string][]Item, error) {
	if len(poIDs) == 0 {
		return map[string][]Item{}, nil
	}
	q := `SELECT i.purchase_order_id, i.id, i.material_id, m.sku, m.name, m.unit,
	             i.ordered_qty, i.received_qty, i.unit_price, i.notes
	      FROM purchase_order_items i
	      JOIN materials m ON m.id = i.material_id
	      WHERE i.purchase_order_id = ANY($1)
	      ORDER BY m.name`
	rows, err := r.pool.Query(ctx, q, poIDs)
	if err != nil {
		return nil, fmt.Errorf("query po items: %w", err)
	}
	defer rows.Close()

	out := map[string][]Item{}
	for rows.Next() {
		var (
			poID string
			it   Item
		)
		if err := rows.Scan(&poID, &it.ID, &it.MaterialID, &it.SKU, &it.Name, &it.Unit,
			&it.Ordered, &it.Received, &it.UnitPrice, &it.Notes); err != nil {
			return nil, err
		}
		out[poID] = append(out[poID], it)
	}
	return out, rows.Err()
}

// enrichItemDetailsTx populates SKU/Name/Unit on items by joining materials inside a tx.
func enrichItemDetailsTx(ctx context.Context, tx pgx.Tx, items []Item) error {
	for i := range items {
		err := tx.QueryRow(ctx,
			"SELECT sku, name, unit FROM materials WHERE id = $1",
			items[i].MaterialID,
		).Scan(&items[i].SKU, &items[i].Name, &items[i].Unit)
		if err != nil {
			return fmt.Errorf("enrich item %s: %w", items[i].ID, err)
		}
	}
	return nil
}

// scanHeader scans a header row (no items column).
func scanHeader(row pgx.Row) (*PurchaseOrder, error) {
	var (
		po        PurchaseOrder
		orderDate time.Time
	)
	err := row.Scan(
		&po.ID, &po.PONumber, &po.VendorID, &po.VendorName,
		&orderDate, &po.Status, &po.Notes, &po.CreatedBy,
		&po.CreatedAt, &po.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	po.OrderDate = orderDate.Format("2006-01-02")
	return &po, nil
}

// ErrMaterialNotFound is wrapped by Create when a referenced material is missing.
var ErrMaterialNotFound = errors.New("material not found")

func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "23505")
}

func isForeignKeyViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "23503")
}
