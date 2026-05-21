package notifications

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("notification not found")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const cols = `id, user_id::text, title, message, type, link, category, read, read_at, created_at`

func scan(row pgx.Row) (*Notification, error) {
	var n Notification
	if err := row.Scan(&n.ID, &n.UserID, &n.Title, &n.Message, &n.Type,
		&n.Link, &n.Category, &n.Read, &n.ReadAt, &n.CreatedAt); err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *Repository) List(ctx context.Context, f ListFilters) ([]Notification, error) {
	if f.UserID == "" {
		return nil, fmt.Errorf("userID is required")
	}
	args := []any{f.UserID}
	conds := "user_id = $1"
	if f.Read != nil {
		args = append(args, *f.Read)
		conds += fmt.Sprintf(" AND read = $%d", len(args))
	}
	if f.Category != "" {
		args = append(args, f.Category)
		conds += fmt.Sprintf(" AND category = $%d", len(args))
	}

	q := "SELECT " + cols + " FROM notifications WHERE " + conds + " ORDER BY created_at DESC"
	if f.Limit > 0 {
		args = append(args, f.Limit)
		q += fmt.Sprintf(" LIMIT $%d", len(args))
	}

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query notifications: %w", err)
	}
	defer rows.Close()

	out := make([]Notification, 0)
	for rows.Next() {
		n, err := scan(rows)
		if err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		out = append(out, *n)
	}
	return out, rows.Err()
}

func (r *Repository) Stats(ctx context.Context, userID string) (*Stats, error) {
	var s Stats
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)::int, COUNT(*) FILTER (WHERE NOT read)::int
		FROM notifications WHERE user_id = $1
	`, userID).Scan(&s.Total, &s.Unread)
	if err != nil {
		return nil, fmt.Errorf("stats: %w", err)
	}
	return &s, nil
}

// Create inserts a new notification, but only if the recipient hasn't opted
// out of the notification's category in users.notification_preferences.
//
// The opt-in default is TRUE: if the user has never visited Settings or the
// category is unmapped, the notification is delivered. Only an explicit
// `false` value in the JSONB blob suppresses delivery.
func (r *Repository) Create(ctx context.Context, in CreateInput) error {
	notifType := strings.TrimSpace(in.Type)
	if notifType == "" {
		notifType = TypeInfo
	}

	prefKey := preferenceKey(in.Category)
	if prefKey == "" {
		// Category not mapped to a user preference — always deliver.
		_, err := r.pool.Exec(ctx, `
			INSERT INTO notifications (user_id, title, message, type, link, category)
			VALUES ($1, $2, $3, $4, $5, $6)
		`,
			in.UserID, in.Title, in.Message, notifType,
			nullIfEmpty(in.Link), nullIfEmpty(in.Category),
		)
		if err != nil {
			return fmt.Errorf("create notification: %w", err)
		}
		return nil
	}

	// Filter via the users table: only insert if the user opted in (default TRUE).
	_, err := r.pool.Exec(ctx, `
		INSERT INTO notifications (user_id, title, message, type, link, category)
		SELECT id, $2, $3, $4, $5, $6
		FROM users
		WHERE id = $1
		  AND COALESCE((notification_preferences->>$7)::boolean, TRUE) = TRUE
	`,
		in.UserID, in.Title, in.Message, notifType,
		nullIfEmpty(in.Link), nullIfEmpty(in.Category), prefKey,
	)
	if err != nil {
		return fmt.Errorf("create notification: %w", err)
	}
	return nil
}

// CreateBulk inserts the same notification for multiple users in one query,
// honouring per-user notification preferences. Users who opted out of the
// notification's category are silently skipped.
func (r *Repository) CreateBulk(ctx context.Context, userIDs []string, in CreateInput) error {
	if len(userIDs) == 0 {
		return nil
	}
	notifType := strings.TrimSpace(in.Type)
	if notifType == "" {
		notifType = TypeInfo
	}

	prefKey := preferenceKey(in.Category)
	if prefKey == "" {
		_, err := r.pool.Exec(ctx, `
			INSERT INTO notifications (user_id, title, message, type, link, category)
			SELECT u, $2, $3, $4, $5, $6 FROM unnest($1::uuid[]) AS u
		`,
			userIDs, in.Title, in.Message, notifType,
			nullIfEmpty(in.Link), nullIfEmpty(in.Category),
		)
		if err != nil {
			return fmt.Errorf("bulk create notification: %w", err)
		}
		return nil
	}

	_, err := r.pool.Exec(ctx, `
		INSERT INTO notifications (user_id, title, message, type, link, category)
		SELECT u.id, $2, $3, $4, $5, $6
		FROM unnest($1::uuid[]) AS ids(id)
		JOIN users u ON u.id = ids.id
		WHERE COALESCE((u.notification_preferences->>$7)::boolean, TRUE) = TRUE
	`,
		userIDs, in.Title, in.Message, notifType,
		nullIfEmpty(in.Link), nullIfEmpty(in.Category), prefKey,
	)
	if err != nil {
		return fmt.Errorf("bulk create notification: %w", err)
	}
	return nil
}

// MarkRead flips a single notification to read=true. Owner-scoped.
func (r *Repository) MarkRead(ctx context.Context, id, userID string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE notifications SET read = TRUE, read_at = NOW()
		WHERE id = $1 AND user_id = $2 AND NOT read
	`, id, userID)
	if err != nil {
		return fmt.Errorf("mark read: %w", err)
	}
	if tag.RowsAffected() == 0 {
		// Either not found or already read; verify which.
		var exists bool
		if err := r.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM notifications WHERE id = $1 AND user_id = $2)", id, userID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return ErrNotFound
		}
		// Already read — treat as success (idempotent).
	}
	return nil
}

// MarkAllRead flips every unread notification of the user to read.
// Returns the number of rows updated.
func (r *Repository) MarkAllRead(ctx context.Context, userID string) (int, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE notifications SET read = TRUE, read_at = NOW()
		WHERE user_id = $1 AND NOT read
	`, userID)
	if err != nil {
		return 0, fmt.Errorf("mark all read: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

// Delete removes a notification. Owner-scoped.
func (r *Repository) Delete(ctx context.Context, id, userID string) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM notifications WHERE id = $1 AND user_id = $2", id, userID)
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// UserIDsByRoles returns the IDs of all active users with one of the given roles.
// Used to fan out notifications to e.g. all supervisors.
func (r *Repository) UserIDsByRoles(ctx context.Context, roles []string) ([]string, error) {
	if len(roles) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx,
		"SELECT id::text FROM users WHERE status = 'active' AND role = ANY($1)",
		roles,
	)
	if err != nil {
		return nil, fmt.Errorf("user ids by roles: %w", err)
	}
	defer rows.Close()
	out := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func nullIfEmpty(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}
