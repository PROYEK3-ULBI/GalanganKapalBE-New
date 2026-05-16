package projects

import "time"

// Project represents a vessel hull under construction or a drydock job.
type Project struct {
	ID            string     `json:"id"`
	Code          string     `json:"code"`
	Name          string     `json:"name"`
	Type          string     `json:"type"`
	Status        string     `json:"status"`
	CompletionPct int        `json:"completion"` // matches frontend mockData field name
	StartDate     *string    `json:"startDate,omitempty"`
	TargetDate    *string    `json:"targetDate,omitempty"`
	Notes         *string    `json:"notes,omitempty"`
	CreatedBy     *string    `json:"createdBy,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

// CreateRequest is the payload for POST /api/projects.
type CreateRequest struct {
	Code          string  `json:"code"`
	Name          string  `json:"name"`
	Type          string  `json:"type"`
	Status        *string `json:"status"`
	CompletionPct *int    `json:"completion"`
	StartDate     *string `json:"startDate"`
	TargetDate    *string `json:"targetDate"`
	Notes         *string `json:"notes"`
}

// UpdateRequest applies partial updates.
type UpdateRequest struct {
	Name          *string `json:"name"`
	Type          *string `json:"type"`
	Status        *string `json:"status"`
	CompletionPct *int    `json:"completion"`
	StartDate     *string `json:"startDate"`
	TargetDate    *string `json:"targetDate"`
	Notes         *string `json:"notes"`
}

// ListFilters controls filtering for GET /api/projects.
type ListFilters struct {
	Search     string // matches code or name
	Status     string
	Type       string
	ActiveOnly bool // excludes Completed and Cancelled
}

// IsValidStatus reports whether the given status string is allowed.
func IsValidStatus(s string) bool {
	switch s {
	case "Planning", "In Progress", "In Drydock", "On Hold", "Completed", "Cancelled":
		return true
	}
	return false
}
