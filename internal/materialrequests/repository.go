package materialrequests

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
	ErrNotFound          = errors.New("material request not found")
	ErrMaterialNotFound  = errors.New("material not found")
	ErrAlreadyDecided    = errors.New("request already decided")
	ErrNotPending        = errors.New("request is not pending")
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// List returns requests with denormalised display fields populated and items eagerly loaded.
func (r *Repository) List(ctx context.Context, f ListFilters) ([]MaterialRequest, error) {
	var (
		conds []string
		args  []any
	)
	add := func(col string, val any) {
		args = append(args, val)
		conds = append(conds, fmt.Sprintf("%s = $%d", col, len(args)))
	}
	if f.Status != "" {
		add("mr.status", f.Status)
	}
	if f.Type != "" {
		add("mr.type", f.Type)
	}
	if f.Priority != "" {
		add("mr.priority", f.Priority)
	}
	if f.ProjectID != "" {
		add("mr.project_id", f.ProjectID)
	}
	if f.RequesterID != "" {
		add("mr.requester_id", f.RequesterID)
	}

	q := `SELECT mr.id, mr.request_no, mr.type,
	             mr.project_id::text, p.code, p.name AS project_name,
	             mr.priority, mr.reason, mr.status,
	             mr.requester_id::text, ru.name AS requester_name,
	             mr.approver_id::text,  au.name AS approver_name,
	             mr.approval_notes, mr.approved_at,
	             mr.request_date, mr.created_at, mr.updated_at
	      FROM material_requests mr
	      JOIN users ru             ON ru.id = mr.requester_id
	      LEFT JOIN users au        ON au.id = mr.approver_id
	      LEFT JOIN projects p      ON p.id = mr.project_id`
	if len(conds) > 0 {
		q += " WHERE " + strings.Join(conds, " AND ")
	}
	q += " ORDER BY mr.request_date DESC, mr.created_at DESC"
	if f.Limit > 0 {
		args = append(args, f.Limit)
		q += fmt.Sprintf(" LIMIT $%d", len(args))
	}

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query material requests: %w", err)
	}
	defer rows.Close()

	var (
		out []MaterialRequest
		ids []string
	)
	for rows.Next() {
		req, err := scanHeader(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *req)
		ids = append(ids, req.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return []MaterialRequest{}, nil
	}

	itemsByReq, err := r.fetchItemsForRequests(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i].Items = itemsByReq[out[i].ID]
		if out[i].Items == nil {
			out[i].Items = []Item{}
		}
	}
	return out, nil
}

func (r *Repository) FindByID(ctx context.Context, id string) (*MaterialRequest, error) {
	q := `SELECT mr.id, mr.request_no, mr.type,
	             mr.project_id::text, p.code, p.name AS project_name,
	             mr.priority, mr.reason, mr.status,
	             mr.requester_id::text, ru.name AS requester_name,
	             mr.approver_id::text,  au.name AS approver_name,
	             mr.approval_notes, mr.approved_at,
	             mr.request_date, mr.created_at, mr.updated_at
	      FROM material_requests mr
	      JOIN users ru             ON ru.id = mr.requester_id
	      LEFT JOIN users au        ON au.id = mr.approver_id
	      LEFT JOIN projects p      ON p.id = mr.project_id
	      WHERE mr.id = $1
	      LIMIT 1`
	row := r.pool.QueryRow(ctx, q, id)
	req, err := scanHeader(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find request: %w", err)
	}
	itemsByReq, err := r.fetchItemsForRequests(ctx, []string{req.ID})
	if err != nil {
		return nil, err
	}
	req.Items = itemsByReq[req.ID]
	if req.Items == nil {
		req.Items = []Item{}
	}
	return req, nil
}

// Create inserts header + items in a single tx.
func (r *Repository) Create(ctx context.Context, req *MaterialRequest, requesterID string) (*MaterialRequest, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	headerQ := `INSERT INTO material_requests
	            (request_no, type, project_id, priority, reason, requester_id)
	            VALUES ($1, $2, $3, $4, $5, $6)
	            RETURNING id`
	var insertedID string
	if err := tx.QueryRow(ctx, headerQ,
		req.RequestNo, req.Type, req.ProjectID, req.Priority, req.Reason, requesterID,
	).Scan(&insertedID); err != nil {
		return nil, fmt.Errorf("insert request header: %w", err)
	}

	for _, it := range req.Items {
		_, err := tx.Exec(ctx,
			`INSERT INTO material_request_items (request_id, material_id, qty, notes)
			 VALUES ($1, $2, $3, $4)`,
			insertedID, it.MaterialID, it.Qty, it.Notes,
		)
		if err != nil {
			if isForeignKeyViolation(err) {
				return nil, fmt.Errorf("material %s: %w", it.MaterialID, ErrMaterialNotFound)
			}
			return nil, fmt.Errorf("insert item: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return r.FindByID(ctx, insertedID)
}

// Decide updates the status of a request to approved or rejected.
// Atomic check: only changes status if currently pending (prevents double-decide race).
func (r *Repository) Decide(ctx context.Context, id, status, approverID string, notes *string) (*MaterialRequest, error) {
	if status != StatusApproved && status != StatusRejected {
		return nil, fmt.Errorf("invalid decision status %q", status)
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE material_requests
		SET status = $1,
		    approver_id = $2,
		    approval_notes = $3,
		    approved_at = NOW()
		WHERE id = $4 AND status = 'pending'
	`, status, approverID, notes, id)
	if err != nil {
		return nil, fmt.Errorf("decide request: %w", err)
	}
	if tag.RowsAffected() == 0 {
		// Either not found or not pending. Distinguish for better error message.
		exists, err := r.exists(ctx, id)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrNotFound
		}
		return nil, ErrAlreadyDecided
	}
	return r.FindByID(ctx, id)
}

// Delete removes a pending request owned by the given user.
// Approved or rejected requests cannot be deleted (immutable audit trail).
func (r *Repository) Delete(ctx context.Context, id, requesterID string) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM material_requests
		WHERE id = $1 AND requester_id = $2 AND status = 'pending'
	`, id, requesterID)
	if err != nil {
		return fmt.Errorf("delete request: %w", err)
	}
	if tag.RowsAffected() == 0 {
		exists, err := r.exists(ctx, id)
		if err != nil {
			return err
		}
		if !exists {
			return ErrNotFound
		}
		return ErrNotPending
	}
	return nil
}

// NextRequestNo generates the next sequential request number for the current year.
func (r *Repository) NextRequestNo(ctx context.Context) (string, error) {
	year := time.Now().Year()
	prefix := fmt.Sprintf("MR-%d-", year)
	var maxSuffix *int
	err := r.pool.QueryRow(ctx, `
		SELECT MAX(NULLIF(REGEXP_REPLACE(request_no, '^MR-\d{4}-', ''), '')::int)
		FROM material_requests
		WHERE request_no LIKE $1
	`, prefix+"%").Scan(&maxSuffix)
	if err != nil {
		return "", fmt.Errorf("next request no: %w", err)
	}
	next := 1
	if maxSuffix != nil {
		next = *maxSuffix + 1
	}
	return fmt.Sprintf("%s%04d", prefix, next), nil
}

func (r *Repository) exists(ctx context.Context, id string) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM material_requests WHERE id = $1)", id).Scan(&ok)
	if err != nil {
		return false, err
	}
	return ok, nil
}

func (r *Repository) fetchItemsForRequests(ctx context.Context, ids []string) (map[string][]Item, error) {
	if len(ids) == 0 {
		return map[string][]Item{}, nil
	}
	q := `SELECT i.request_id, i.id, i.material_id, m.sku, m.name, m.unit, i.qty, i.notes
	      FROM material_request_items i
	      JOIN materials m ON m.id = i.material_id
	      WHERE i.request_id = ANY($1)
	      ORDER BY m.name`
	rows, err := r.pool.Query(ctx, q, ids)
	if err != nil {
		return nil, fmt.Errorf("query items: %w", err)
	}
	defer rows.Close()

	out := map[string][]Item{}
	for rows.Next() {
		var (
			reqID string
			it    Item
		)
		if err := rows.Scan(&reqID, &it.ID, &it.MaterialID, &it.SKU, &it.Name, &it.Unit, &it.Qty, &it.Notes); err != nil {
			return nil, err
		}
		out[reqID] = append(out[reqID], it)
	}
	return out, rows.Err()
}

func scanHeader(row pgx.Row) (*MaterialRequest, error) {
	var (
		req       MaterialRequest
		reqDate   time.Time
	)
	err := row.Scan(
		&req.ID, &req.RequestNo, &req.Type,
		&req.ProjectID, &req.ProjectCode, &req.ProjectName,
		&req.Priority, &req.Reason, &req.Status,
		&req.RequesterID, &req.RequesterName,
		&req.ApproverID, &req.ApproverName,
		&req.ApprovalNotes, &req.ApprovedAt,
		&reqDate, &req.CreatedAt, &req.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	req.RequestDate = reqDate.Format("2006-01-02")
	return &req, nil
}

func isForeignKeyViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "23503")
}
