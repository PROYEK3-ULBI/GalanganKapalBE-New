package support

import "time"

const (
	StatusOpen       = "open"
	StatusInProgress = "in_progress"
	StatusResolved   = "resolved"
	StatusClosed     = "closed"

	PriorityLow    = "low"
	PriorityMedium = "medium"
	PriorityHigh   = "high"
)

// Ticket represents one row in support_tickets, joined with user names.
type Ticket struct {
	ID            string     `json:"id"`
	TicketNo      string     `json:"ticketNo"`
	UserID        string     `json:"userId"`
	UserName      string     `json:"user"` // matches frontend convention
	Subject       string     `json:"subject"`
	Message       string     `json:"message"`
	Status        string     `json:"status"`
	Priority      string     `json:"priority"`
	Response      *string    `json:"response,omitempty"`
	HandlerID     *string    `json:"handlerId,omitempty"`
	HandlerName   *string    `json:"handler,omitempty"`
	ResolvedAt    *time.Time `json:"resolvedAt,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

// CreateRequest is the payload for POST /api/support/tickets.
type CreateRequest struct {
	Subject  string  `json:"subject"`
	Message  string  `json:"message"`
	Priority *string `json:"priority"`
}

// UpdateStatusRequest is the payload for PATCH /api/support/tickets/:id/status (admin).
type UpdateStatusRequest struct {
	Status   string  `json:"status"`
	Response *string `json:"response"`
}

// ResolveRequest is the payload for POST /api/support/tickets/:id/resolve (admin).
type ResolveRequest struct {
	Response *string `json:"response"`
}

// ListFilters controls GET /api/support/tickets.
type ListFilters struct {
	UserID   string // forced to current user for non-admins
	Status   string
	Priority string
}

func IsValidStatus(s string) bool {
	switch s {
	case StatusOpen, StatusInProgress, StatusResolved, StatusClosed:
		return true
	}
	return false
}

func IsValidPriority(s string) bool {
	switch s {
	case PriorityLow, PriorityMedium, PriorityHigh:
		return true
	}
	return false
}
