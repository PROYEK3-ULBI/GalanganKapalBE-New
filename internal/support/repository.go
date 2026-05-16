package support

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
	ErrNotFound     = errors.New("ticket not found")
	ErrCannotDelete = errors.New("only the requester can delete an open ticket")
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const cols = `t.id, t.ticket_no, t.user_id::text, ru.name AS user_name,
		t.subject, t.message, t.status, t.priority,
		t.response,
		t.handled_by::text, hu.name AS handler_name,
		t.resolved_at, t.created_at, t.updated_at`

func scan(row pgx.Row) (*Ticket, error) {
	var t Ticket
	err := row.Scan(
		&t.ID, &t.TicketNo, &t.UserID, &t.UserName,
		&t.Subject, &t.Message, &t.Status, &t.Priority,
		&t.Response,
		&t.HandlerID, &t.HandlerName,
		&t.ResolvedAt, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *Repository) List(ctx context.Context, f ListFilters) ([]Ticket, error) {
	var (
		conds []string
		args  []any
	)
	if f.UserID != "" {
		args = append(args, f.UserID)
		conds = append(conds, fmt.Sprintf("t.user_id = $%d", len(args)))
	}
	if f.Status != "" {
		args = append(args, f.Status)
		conds = append(conds, fmt.Sprintf("t.status = $%d", len(args)))
	}
	if f.Priority != "" {
		args = append(args, f.Priority)
		conds = append(conds, fmt.Sprintf("t.priority = $%d", len(args)))
	}

	q := `SELECT ` + cols + `
	      FROM support_tickets t
	      JOIN users ru ON ru.id = t.user_id
	      LEFT JOIN users hu ON hu.id = t.handled_by`
	if len(conds) > 0 {
		q += " WHERE " + strings.Join(conds, " AND ")
	}
	q += " ORDER BY t.created_at DESC"

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query tickets: %w", err)
	}
	defer rows.Close()

	out := make([]Ticket, 0)
	for rows.Next() {
		t, err := scan(rows)
		if err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

func (r *Repository) FindByID(ctx context.Context, id string) (*Ticket, error) {
	q := `SELECT ` + cols + `
	      FROM support_tickets t
	      JOIN users ru ON ru.id = t.user_id
	      LEFT JOIN users hu ON hu.id = t.handled_by
	      WHERE t.id = $1
	      LIMIT 1`
	row := r.pool.QueryRow(ctx, q, id)
	t, err := scan(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find: %w", err)
	}
	return t, nil
}

func (r *Repository) Create(ctx context.Context, t *Ticket, userID string) (*Ticket, error) {
	var id string
	err := r.pool.QueryRow(ctx, `
		INSERT INTO support_tickets (ticket_no, user_id, subject, message, status, priority)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, t.TicketNo, userID, t.Subject, t.Message, t.Status, t.Priority).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("create ticket: %w", err)
	}
	return r.FindByID(ctx, id)
}

// SetStatus updates status + handler + response. Always recorded by an admin.
// resolvedAt is set automatically when status transitions to 'resolved'.
func (r *Repository) SetStatus(ctx context.Context, id, status, handlerID string, response *string) (*Ticket, error) {
	q := `
		UPDATE support_tickets
		SET status = $1::text,
		    handled_by = $2::uuid,
		    response = COALESCE($3, response),
		    resolved_at = CASE WHEN $1::text = 'resolved' THEN NOW() ELSE resolved_at END
		WHERE id = $4::uuid
	`
	tag, err := r.pool.Exec(ctx, q, status, handlerID, response, id)
	if err != nil {
		return nil, fmt.Errorf("set status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return r.FindByID(ctx, id)
}

// Delete removes a ticket. Only the requester can delete, and only while the
// ticket is still 'open' (preserves audit trail for resolved/closed tickets).
func (r *Repository) Delete(ctx context.Context, id, requesterID string) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM support_tickets
		WHERE id = $1 AND user_id = $2 AND status = 'open'
	`, id, requesterID)
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		exists, err := r.exists(ctx, id)
		if err != nil {
			return err
		}
		if !exists {
			return ErrNotFound
		}
		return ErrCannotDelete
	}
	return nil
}

func (r *Repository) exists(ctx context.Context, id string) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM support_tickets WHERE id = $1)", id).Scan(&ok)
	if err != nil {
		return false, err
	}
	return ok, nil
}

// NextTicketNo returns the next sequential ticket number for the current year.
func (r *Repository) NextTicketNo(ctx context.Context) (string, error) {
	year := time.Now().Year()
	prefix := fmt.Sprintf("TKT-%d-", year)
	var maxSuffix *int
	err := r.pool.QueryRow(ctx, `
		SELECT MAX(NULLIF(REGEXP_REPLACE(ticket_no, '^TKT-\d{4}-', ''), '')::int)
		FROM support_tickets
		WHERE ticket_no LIKE $1
	`, prefix+"%").Scan(&maxSuffix)
	if err != nil {
		return "", fmt.Errorf("next ticket no: %w", err)
	}
	next := 1
	if maxSuffix != nil {
		next = *maxSuffix + 1
	}
	return fmt.Sprintf("%s%04d", prefix, next), nil
}
