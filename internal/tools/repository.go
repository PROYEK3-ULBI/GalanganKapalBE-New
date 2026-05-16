package tools

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
	ErrNotFound         = errors.New("tool not found")
	ErrSKUConflict      = errors.New("tool sku already exists")
	ErrAlreadyCheckedOut = errors.New("tool is already checked out")
	ErrNotCheckedOut    = errors.New("tool is not currently checked out")
	ErrInMaintenance    = errors.New("tool is in maintenance")
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const toolColumns = `t.id, t.sku, t.name, t.category, t.status, t.condition,
		t.location, t.borrower_id::text, u.name AS borrower_name,
		t.borrow_date, t.calibration_due_date,
		t.notes, t.image_url, t.created_at, t.updated_at`

func scanTool(row pgx.Row) (*Tool, error) {
	var (
		t          Tool
		borrowDate *time.Time
		calibDate  *time.Time
	)
	err := row.Scan(
		&t.ID, &t.SKU, &t.Name, &t.Category, &t.Status, &t.Condition,
		&t.Location, &t.BorrowerID, &t.BorrowerName,
		&borrowDate, &calibDate,
		&t.Notes, &t.ImageURL, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if borrowDate != nil {
		s := borrowDate.Format("2006-01-02")
		t.BorrowDate = &s
	}
	if calibDate != nil {
		s := calibDate.Format("2006-01-02")
		t.CalibrationDueDate = &s
	}
	return &t, nil
}

func (r *Repository) List(ctx context.Context, f ListFilters) ([]Tool, error) {
	var (
		conds []string
		args  []any
	)
	if f.Search != "" {
		args = append(args, "%"+f.Search+"%")
		idx := len(args)
		conds = append(conds, fmt.Sprintf("(LOWER(t.sku) LIKE LOWER($%d) OR LOWER(t.name) LIKE LOWER($%d))", idx, idx))
	}
	if f.Status != "" {
		args = append(args, f.Status)
		conds = append(conds, fmt.Sprintf("t.status = $%d", len(args)))
	}
	if f.Category != "" {
		args = append(args, f.Category)
		conds = append(conds, fmt.Sprintf("t.category = $%d", len(args)))
	}
	if f.BorrowerID != "" {
		args = append(args, f.BorrowerID)
		conds = append(conds, fmt.Sprintf("t.borrower_id = $%d", len(args)))
	}
	if f.CalibrationDueOnly {
		// Calibration due within next 30 days, or already overdue.
		conds = append(conds, "t.calibration_due_date IS NOT NULL AND t.calibration_due_date <= CURRENT_DATE + INTERVAL '30 days'")
	}

	q := "SELECT " + toolColumns + " FROM tools t LEFT JOIN users u ON u.id = t.borrower_id"
	if len(conds) > 0 {
		q += " WHERE " + strings.Join(conds, " AND ")
	}
	q += " ORDER BY t.name ASC"

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query tools: %w", err)
	}
	defer rows.Close()

	out := make([]Tool, 0)
	for rows.Next() {
		t, err := scanTool(rows)
		if err != nil {
			return nil, fmt.Errorf("scan tool: %w", err)
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

func (r *Repository) FindByID(ctx context.Context, id string) (*Tool, error) {
	q := "SELECT " + toolColumns + " FROM tools t LEFT JOIN users u ON u.id = t.borrower_id WHERE t.id = $1 LIMIT 1"
	row := r.pool.QueryRow(ctx, q, id)
	t, err := scanTool(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find tool: %w", err)
	}
	return t, nil
}

func (r *Repository) Create(ctx context.Context, t *Tool) (*Tool, error) {
	q := `INSERT INTO tools (sku, name, category, status, condition, location, calibration_due_date, notes, image_url)
	      VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	      RETURNING id`
	var (
		id        string
		calibDate any
	)
	if t.CalibrationDueDate != nil && *t.CalibrationDueDate != "" {
		calibDate = *t.CalibrationDueDate
	}
	err := r.pool.QueryRow(ctx, q,
		strings.ToUpper(strings.TrimSpace(t.SKU)),
		t.Name, t.Category, t.Status, t.Condition,
		t.Location, calibDate, t.Notes, t.ImageURL,
	).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrSKUConflict
		}
		return nil, fmt.Errorf("create tool: %w", err)
	}
	return r.FindByID(ctx, id)
}

func (r *Repository) Update(ctx context.Context, id string, req UpdateRequest) (*Tool, error) {
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
	if req.Condition != nil {
		add("condition", *req.Condition)
	}
	if req.Location != nil {
		add("location", nullIfEmpty(*req.Location))
	}
	if req.CalibrationDueDate != nil {
		if *req.CalibrationDueDate == "" {
			add("calibration_due_date", nil)
		} else {
			add("calibration_due_date", *req.CalibrationDueDate)
		}
	}
	if req.Notes != nil {
		add("notes", nullIfEmpty(*req.Notes))
	}
	if req.ImageURL != nil {
		add("image_url", nullIfEmpty(*req.ImageURL))
	}
	if len(sets) == 0 {
		return r.FindByID(ctx, id)
	}
	args = append(args, id)
	q := fmt.Sprintf("UPDATE tools SET %s WHERE id = $%d", strings.Join(sets, ", "), len(args))
	tag, err := r.pool.Exec(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("update tool: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return r.FindByID(ctx, id)
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM tools WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete tool: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Checkout transitions tool from Available to In Use, sets borrower, logs history.
// All in one tx for atomicity.
func (r *Repository) Checkout(ctx context.Context, toolID, borrowerID string, notes *string) (*Tool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Lock the tool row to prevent concurrent checkout.
	var currentStatus string
	err = tx.QueryRow(ctx, "SELECT status FROM tools WHERE id = $1 FOR UPDATE", toolID).Scan(&currentStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("lock tool: %w", err)
	}
	switch currentStatus {
	case StatusInUse:
		return nil, ErrAlreadyCheckedOut
	case StatusMaintenance:
		return nil, ErrInMaintenance
	}

	_, err = tx.Exec(ctx, `
		UPDATE tools
		SET status = 'In Use', borrower_id = $1, borrow_date = CURRENT_DATE
		WHERE id = $2
	`, borrowerID, toolID)
	if err != nil {
		return nil, fmt.Errorf("checkout tool: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO tool_history (tool_id, action, user_id, notes)
		VALUES ($1, 'checkout', $2, $3)
	`, toolID, borrowerID, notes)
	if err != nil {
		return nil, fmt.Errorf("log history: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return r.FindByID(ctx, toolID)
}

// Return transitions tool from In Use to Available, clears borrower, logs history.
// userID is the actor (current authenticated user), used for audit.
func (r *Repository) Return(ctx context.Context, toolID string, userID *string, condition *string, notes *string) (*Tool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var currentStatus string
	err = tx.QueryRow(ctx, "SELECT status FROM tools WHERE id = $1 FOR UPDATE", toolID).Scan(&currentStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("lock tool: %w", err)
	}
	if currentStatus != StatusInUse {
		return nil, ErrNotCheckedOut
	}

	if condition != nil && *condition != "" {
		_, err = tx.Exec(ctx, `
			UPDATE tools
			SET status = 'Available', borrower_id = NULL, borrow_date = NULL, condition = $1
			WHERE id = $2
		`, *condition, toolID)
	} else {
		_, err = tx.Exec(ctx, `
			UPDATE tools
			SET status = 'Available', borrower_id = NULL, borrow_date = NULL
			WHERE id = $1
		`, toolID)
	}
	if err != nil {
		return nil, fmt.Errorf("return tool: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO tool_history (tool_id, action, user_id, notes)
		VALUES ($1, 'return', $2, $3)
	`, toolID, userID, notes)
	if err != nil {
		return nil, fmt.Errorf("log history: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return r.FindByID(ctx, toolID)
}

// SetMaintenance moves tool to Maintenance status (or back to Available).
// Use action='maintenance' or 'available'.
func (r *Repository) SetStatus(ctx context.Context, toolID, newStatus string, userID *string, condition *string, notes *string) (*Tool, error) {
	if newStatus != StatusAvailable && newStatus != StatusMaintenance {
		return nil, fmt.Errorf("invalid status transition target: %s", newStatus)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var currentStatus string
	err = tx.QueryRow(ctx, "SELECT status FROM tools WHERE id = $1 FOR UPDATE", toolID).Scan(&currentStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("lock tool: %w", err)
	}
	if currentStatus == StatusInUse {
		return nil, ErrAlreadyCheckedOut
	}

	if condition != nil && *condition != "" {
		_, err = tx.Exec(ctx,
			"UPDATE tools SET status = $1, condition = $2 WHERE id = $3",
			newStatus, *condition, toolID,
		)
	} else {
		_, err = tx.Exec(ctx, "UPDATE tools SET status = $1 WHERE id = $2", newStatus, toolID)
	}
	if err != nil {
		return nil, fmt.Errorf("update tool status: %w", err)
	}

	historyAction := "available"
	if newStatus == StatusMaintenance {
		historyAction = "maintenance"
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO tool_history (tool_id, action, user_id, notes)
		VALUES ($1, $2, $3, $4)
	`, toolID, historyAction, userID, notes)
	if err != nil {
		return nil, fmt.Errorf("log history: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return r.FindByID(ctx, toolID)
}

// History returns the full action log for a tool, newest first.
func (r *Repository) History(ctx context.Context, toolID string) ([]HistoryEntry, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT h.id, h.tool_id::text, h.action, h.user_id::text, u.name, h.notes, h.created_at
		FROM tool_history h
		LEFT JOIN users u ON u.id = h.user_id
		WHERE h.tool_id = $1
		ORDER BY h.created_at DESC
	`, toolID)
	if err != nil {
		return nil, fmt.Errorf("query history: %w", err)
	}
	defer rows.Close()

	out := make([]HistoryEntry, 0)
	for rows.Next() {
		var h HistoryEntry
		if err := rows.Scan(&h.ID, &h.ToolID, &h.Action, &h.UserID, &h.UserName, &h.Notes, &h.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
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
