package activitylog

import "time"

// Type constants align with frontend Badge variants.
const (
	TypeInfo    = "info"
	TypeSuccess = "success"
	TypeWarning = "warning"
	TypeDanger  = "danger"
)

// Entry is one row in the activity log feed.
type Entry struct {
	ID           string    `json:"id"`
	Action       string    `json:"action"`
	Detail       *string   `json:"detail,omitempty"`
	Type         string    `json:"type"`
	UserID       *string   `json:"userId,omitempty"`
	UserName     *string   `json:"user,omitempty"` // matches frontend mock field
	ResourceType *string   `json:"resourceType,omitempty"`
	ResourceID   *string   `json:"resourceId,omitempty"`
	Category     *string   `json:"category,omitempty"`
	CreatedAt    time.Time `json:"time"` // matches frontend mock field
}

// CreateInput is used by other modules to log an event.
type CreateInput struct {
	Action       string
	Detail       string
	Type         string
	UserID       string
	ResourceType string
	ResourceID   string
	Category     string
}

// ListFilters controls GET /api/activity-logs.
type ListFilters struct {
	UserID    string
	Type      string
	Category  string
	StartDate string
	EndDate   string
	Limit     int
}

func IsValidType(s string) bool {
	switch s {
	case TypeInfo, TypeSuccess, TypeWarning, TypeDanger:
		return true
	}
	return false
}
