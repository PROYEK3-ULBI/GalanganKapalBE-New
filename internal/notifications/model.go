package notifications

import "time"

// Notification types align with frontend Badge/dot variants.
const (
	TypeInfo    = "info"
	TypeSuccess = "success"
	TypeWarning = "warning"
	TypeDanger  = "danger"
)

// Notification is one row in the per-user notification feed.
type Notification struct {
	ID        string     `json:"id"`
	UserID    string     `json:"userId"`
	Title     string     `json:"title"`
	Message   string     `json:"message"`
	Type      string     `json:"type"`
	Link      *string    `json:"link,omitempty"`
	Category  *string    `json:"category,omitempty"`
	Read      bool       `json:"read"`
	ReadAt    *time.Time `json:"readAt,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
}

// CreateInput is used by other modules to push a notification to a user.
// Type defaults to 'info' if empty.
type CreateInput struct {
	UserID   string
	Title    string
	Message  string
	Type     string
	Link     string
	Category string
}

// ListFilters controls GET /api/notifications.
type ListFilters struct {
	UserID     string // forced to current user at handler level
	Read       *bool  // nil = all, true/false = filter
	Limit      int
	Category   string
}

// Stats describes the per-user counts used for the bell badge.
type Stats struct {
	Total  int `json:"total"`
	Unread int `json:"unread"`
}

func IsValidType(t string) bool {
	switch t {
	case TypeInfo, TypeSuccess, TypeWarning, TypeDanger:
		return true
	}
	return false
}
