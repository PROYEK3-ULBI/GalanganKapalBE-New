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

type ErrValidation struct{ Msg string }

func (e *ErrValidation) Error() string { return e.Msg }

func newValidationError(msg string) error { return &ErrValidation{Msg: msg} }

func IsValidation(err error) bool {
	var v *ErrValidation
	return errors.As(err, &v)
}

type Service struct {
	pool *pgxpool.Pool
	repo *Repository
}

func NewService(pool *pgxpool.Pool, repo *Repository) *Service {
	return &Service{pool: pool, repo: repo}
}

func (s *Service) List(ctx context.Context, f ListFilters) ([]Transaction, error) {
	return s.repo.List(ctx, f)
}

func (s *Service) Get(ctx context.Context, id string) (*Transaction, error) {
	return s.repo.FindByID(ctx, id)
}

// =============================================================================
// Goods Receipt
// =============================================================================

// Receipt creates a batch of receipt transactions in one DB transaction.
// Side effects per item:
//   - INSERT transactions row
//   - UPDATE materials.stock += qty
//   - If linked to PO item: UPDATE purchase_order_items.received_qty += qty
//   - After all items committed: refresh parent PO status
func (s *Service) Receipt(ctx context.Context, req ReceiptRequest, userID *string) ([]Transaction, error) {
	if len(req.Items) == 0 {
		return nil, newValidationError("at least one item is required")
	}
	if req.PurchaseOrderID == nil && req.VendorID == nil {
		return nil, newValidationError("purchaseOrderId or vendorId is required")
	}

	// Resolve transaction date.
	txDate, err := parseDateOrNow(req.TransactionDate)
	if err != nil {
		return nil, err
	}

	// Validate item shapes early.
	for i, it := range req.Items {
		if strings.TrimSpace(it.MaterialID) == "" {
			return nil, newValidationError(fmt.Sprintf("items[%d].materialId is required", i))
		}
		if it.Qty <= 0 {
			return nil, newValidationError(fmt.Sprintf("items[%d].qty must be > 0", i))
		}
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// If linked to a PO, derive vendor automatically.
	var resolvedVendorID *string = req.VendorID
	if req.PurchaseOrderID != nil {
		var venID string
		err := tx.QueryRow(ctx, "SELECT vendor_id::text FROM purchase_orders WHERE id = $1", *req.PurchaseOrderID).Scan(&venID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, newValidationError("purchase order not found")
			}
			return nil, fmt.Errorf("lookup po vendor: %w", err)
		}
		resolvedVendorID = &venID
	}

	out := make([]Transaction, 0, len(req.Items))
	for _, item := range req.Items {
		// If item is linked to a PO line, validate ordered/received bounds.
		if item.POItemID != "" {
			if err := validatePOItemForReceipt(ctx, tx, item.POItemID, item.MaterialID, item.Qty); err != nil {
				return nil, err
			}
		}

		// Insert ledger row.
		txnNo, err := s.repo.nextTransactionNoTx(ctx, tx)
		if err != nil {
			return nil, err
		}
		var poItemPtr *string
		if item.POItemID != "" {
			id := item.POItemID
			poItemPtr = &id
		}
		var insertedID string
		err = tx.QueryRow(ctx, `
			INSERT INTO transactions (
				transaction_no, type, material_id, qty,
				vendor_id, purchase_order_id, po_item_id,
				user_id, heat_number, notes, transaction_date
			) VALUES ($1, 'receipt', $2, $3, $4, $5, $6, $7, $8, $9, $10)
			RETURNING id
		`, txnNo, item.MaterialID, item.Qty,
			resolvedVendorID, req.PurchaseOrderID, poItemPtr,
			userID, item.HeatNumber, mergedNotes(item.Notes, req.Notes), txDate,
		).Scan(&insertedID)
		if err != nil {
			return nil, fmt.Errorf("insert transaction: %w", err)
		}

		// Update material stock atomically.
		if err := updateMaterialStock(ctx, tx, item.MaterialID, item.Qty); err != nil {
			return nil, err
		}

		// Update PO line received_qty if linked.
		if item.POItemID != "" {
			_, err := tx.Exec(ctx, `
				UPDATE purchase_order_items
				SET received_qty = received_qty + $1
				WHERE id = $2
			`, item.Qty, item.POItemID)
			if err != nil {
				if isCheckViolation(err) {
					return nil, ErrReceiveExceedsOrdered
				}
				return nil, fmt.Errorf("update po item: %w", err)
			}
		}

		// Hydrate the inserted transaction for response.
		row, err := fetchTransactionByID(ctx, tx, insertedID)
		if err != nil {
			return nil, err
		}
		out = append(out, *row)
	}

	// Refresh PO header status after all items.
	if req.PurchaseOrderID != nil {
		if err := refreshPOStatus(ctx, tx, *req.PurchaseOrderID); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit receipt: %w", err)
	}
	return out, nil
}

// =============================================================================
// Goods Issue
// =============================================================================

// Issue creates issue transactions, decrementing material stock atomically.
// Per-item side effects:
//   - SELECT FOR UPDATE on materials.stock to lock the row
//   - Reject if requested qty > available stock
//   - INSERT transactions row, UPDATE materials.stock -= qty
func (s *Service) Issue(ctx context.Context, req IssueRequest, userID *string) ([]Transaction, error) {
	if strings.TrimSpace(req.ProjectID) == "" {
		return nil, newValidationError("projectId is required")
	}
	if len(req.Items) == 0 {
		return nil, newValidationError("at least one item is required")
	}

	txDate, err := parseDateOrNow(req.TransactionDate)
	if err != nil {
		return nil, err
	}

	for i, it := range req.Items {
		if strings.TrimSpace(it.MaterialID) == "" {
			return nil, newValidationError(fmt.Sprintf("items[%d].materialId is required", i))
		}
		if it.Qty <= 0 {
			return nil, newValidationError(fmt.Sprintf("items[%d].qty must be > 0", i))
		}
	}

	// Validate project exists (cheap check up-front).
	var projExists bool
	if err := s.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM projects WHERE id = $1)", req.ProjectID).Scan(&projExists); err != nil {
		return nil, fmt.Errorf("check project: %w", err)
	}
	if !projExists {
		return nil, newValidationError("project not found")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	out := make([]Transaction, 0, len(req.Items))
	for _, item := range req.Items {
		// Lock the material row to prevent concurrent stock updates.
		var currentStock int
		err := tx.QueryRow(ctx,
			"SELECT stock FROM materials WHERE id = $1 FOR UPDATE",
			item.MaterialID,
		).Scan(&currentStock)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrMaterialNotFound
			}
			return nil, fmt.Errorf("lock material: %w", err)
		}
		if currentStock < item.Qty {
			return nil, fmt.Errorf("%w (material %s: have %d, need %d)",
				ErrInsufficientStock, item.MaterialID, currentStock, item.Qty)
		}

		// Decrement stock.
		if _, err := tx.Exec(ctx,
			"UPDATE materials SET stock = stock - $1 WHERE id = $2",
			item.Qty, item.MaterialID,
		); err != nil {
			return nil, fmt.Errorf("update stock: %w", err)
		}

		// Insert ledger row.
		txnNo, err := s.repo.nextTransactionNoTx(ctx, tx)
		if err != nil {
			return nil, err
		}
		var insertedID string
		err = tx.QueryRow(ctx, `
			INSERT INTO transactions (
				transaction_no, type, material_id, qty,
				project_id, user_id,
				heat_number, notes, transaction_date
			) VALUES ($1, 'issue', $2, $3, $4, $5, $6, $7, $8)
			RETURNING id
		`, txnNo, item.MaterialID, item.Qty,
			req.ProjectID, userID,
			item.HeatNumber, mergedNotesWithMandor(item.Notes, req.Notes, req.Mandor), txDate,
		).Scan(&insertedID)
		if err != nil {
			return nil, fmt.Errorf("insert transaction: %w", err)
		}

		row, err := fetchTransactionByID(ctx, tx, insertedID)
		if err != nil {
			return nil, err
		}
		out = append(out, *row)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit issue: %w", err)
	}
	return out, nil
}

// =============================================================================
// Scrap & Return
// =============================================================================

// ScrapReturn creates one transaction (scrap deducts stock, return adds stock).
func (s *Service) ScrapReturn(ctx context.Context, req ScrapReturnRequest, userID *string) (*Transaction, error) {
	rt := strings.ToLower(strings.TrimSpace(req.ReturnType))
	if rt != TypeScrap && rt != TypeReturn {
		return nil, newValidationError("type must be 'scrap' or 'return'")
	}
	if strings.TrimSpace(req.MaterialID) == "" {
		return nil, newValidationError("materialId is required")
	}
	if req.Qty <= 0 {
		return nil, newValidationError("qty must be > 0")
	}
	if strings.TrimSpace(req.Reason) == "" {
		return nil, newValidationError("reason is required")
	}

	txDate, err := parseDateOrNow(req.TransactionDate)
	if err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Lock material row.
	var currentStock int
	err = tx.QueryRow(ctx,
		"SELECT stock FROM materials WHERE id = $1 FOR UPDATE",
		req.MaterialID,
	).Scan(&currentStock)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMaterialNotFound
		}
		return nil, fmt.Errorf("lock material: %w", err)
	}

	// Adjust stock.
	delta := req.Qty
	if rt == TypeScrap {
		if currentStock < req.Qty {
			return nil, fmt.Errorf("%w (have %d, need %d)", ErrInsufficientStock, currentStock, req.Qty)
		}
		_, err = tx.Exec(ctx, "UPDATE materials SET stock = stock - $1 WHERE id = $2", delta, req.MaterialID)
	} else {
		_, err = tx.Exec(ctx, "UPDATE materials SET stock = stock + $1 WHERE id = $2", delta, req.MaterialID)
	}
	if err != nil {
		return nil, fmt.Errorf("update stock: %w", err)
	}

	// Insert ledger row.
	txnNo, err := s.repo.nextTransactionNoTx(ctx, tx)
	if err != nil {
		return nil, err
	}
	notes := req.Reason
	var insertedID string
	err = tx.QueryRow(ctx, `
		INSERT INTO transactions (
			transaction_no, type, material_id, qty,
			project_id, user_id,
			heat_number, notes, transaction_date
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`, txnNo, rt, req.MaterialID, req.Qty,
		req.ProjectID, userID,
		req.HeatNumber, &notes, txDate,
	).Scan(&insertedID)
	if err != nil {
		return nil, fmt.Errorf("insert transaction: %w", err)
	}

	row, err := fetchTransactionByID(ctx, tx, insertedID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit scrap/return: %w", err)
	}
	return row, nil
}

// =============================================================================
// Helpers (transaction-scoped)
// =============================================================================

// validatePOItemForReceipt checks the PO item exists, matches the material,
// and that the additional qty does not exceed remaining ordered qty.
func validatePOItemForReceipt(ctx context.Context, tx pgx.Tx, poItemID, materialID string, qty int) error {
	var (
		matID    string
		ordered  int
		received int
	)
	err := tx.QueryRow(ctx, `
		SELECT material_id::text, ordered_qty, received_qty
		FROM purchase_order_items
		WHERE id = $1
		FOR UPDATE
	`, poItemID).Scan(&matID, &ordered, &received)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrPOItemNotFound
		}
		return fmt.Errorf("lock po item: %w", err)
	}
	if matID != materialID {
		return ErrPOItemMaterialMismatch
	}
	if received+qty > ordered {
		return fmt.Errorf("%w (item %s: ordered %d, already received %d, attempting +%d)",
			ErrReceiveExceedsOrdered, poItemID, ordered, received, qty)
	}
	return nil
}

// updateMaterialStock increments stock by delta. Caller already validated material exists
// or holds a row lock (e.g. from FOR UPDATE earlier in the tx).
func updateMaterialStock(ctx context.Context, tx pgx.Tx, materialID string, delta int) error {
	tag, err := tx.Exec(ctx,
		"UPDATE materials SET stock = stock + $1 WHERE id = $2",
		delta, materialID,
	)
	if err != nil {
		return fmt.Errorf("update material stock: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrMaterialNotFound
	}
	return nil
}

// refreshPOStatus inspects all items of a PO and sets the header status to:
//   - Completed if all items fully received (sum received == sum ordered for every line)
//   - Partially Received if at least one item has received > 0 but not fully received
//   - leaves Draft/Pending alone otherwise (no items received yet)
func refreshPOStatus(ctx context.Context, tx pgx.Tx, poID string) error {
	var (
		totalOrdered  int
		totalReceived int
	)
	err := tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(ordered_qty), 0), COALESCE(SUM(received_qty), 0)
		FROM purchase_order_items
		WHERE purchase_order_id = $1
	`, poID).Scan(&totalOrdered, &totalReceived)
	if err != nil {
		return fmt.Errorf("aggregate po items: %w", err)
	}
	var newStatus string
	switch {
	case totalReceived == 0:
		// no change needed; keep current status (Draft/Pending)
		return nil
	case totalReceived >= totalOrdered:
		newStatus = "Completed"
	default:
		newStatus = "Partially Received"
	}
	_, err = tx.Exec(ctx, "UPDATE purchase_orders SET status = $1 WHERE id = $2", newStatus, poID)
	if err != nil {
		return fmt.Errorf("update po status: %w", err)
	}
	return nil
}

// fetchTransactionByID re-reads a transaction inside the same tx, fully joined.
func fetchTransactionByID(ctx context.Context, tx pgx.Tx, id string) (*Transaction, error) {
	row := tx.QueryRow(ctx, `
		SELECT t.id, t.transaction_no, t.type,
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
	`, id)
	t, err := scanTransaction(row)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// parseDateOrNow parses an optional ISO date string or returns time.Now().
// Accepts both "YYYY-MM-DD" and full RFC3339 timestamps.
func parseDateOrNow(s *string) (time.Time, error) {
	if s == nil || strings.TrimSpace(*s) == "" {
		return time.Now(), nil
	}
	v := strings.TrimSpace(*s)
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02", v); err == nil {
		return t, nil
	}
	return time.Time{}, newValidationError("date must be YYYY-MM-DD or RFC3339")
}

// mergedNotes returns item-level notes if set, falling back to the request-level notes.
func mergedNotes(itemNotes, reqNotes *string) *string {
	if itemNotes != nil && strings.TrimSpace(*itemNotes) != "" {
		return itemNotes
	}
	return reqNotes
}

// mergedNotesWithMandor adds mandor info to the issue notes for traceability.
func mergedNotesWithMandor(itemNotes, reqNotes, mandor *string) *string {
	parts := make([]string, 0, 3)
	if mandor != nil && strings.TrimSpace(*mandor) != "" {
		parts = append(parts, "Mandor: "+strings.TrimSpace(*mandor))
	}
	if itemNotes != nil && strings.TrimSpace(*itemNotes) != "" {
		parts = append(parts, *itemNotes)
	} else if reqNotes != nil && strings.TrimSpace(*reqNotes) != "" {
		parts = append(parts, *reqNotes)
	}
	if len(parts) == 0 {
		return nil
	}
	merged := strings.Join(parts, " | ")
	return &merged
}

// isCheckViolation reports whether the error is a PostgreSQL check constraint violation.
// Used to detect the received_qty > ordered_qty constraint trip.
func isCheckViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "23514")
}
