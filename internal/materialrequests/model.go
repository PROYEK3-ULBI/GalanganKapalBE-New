package materialrequests

import "time"

// Status constants.
const (
	StatusPending  = "pending"
	StatusApproved = "approved"
	StatusRejected = "rejected"
)

// Priority constants.
const (
	PriorityLow    = "low"
	PriorityMedium = "medium"
	PriorityHigh   = "high"
)

// Type constants.
const (
	TypeMaterial = "Material Request"
	TypeTool     = "Tool Request"
	TypePurchase = "Purchase Request"
)

// MaterialRequest is the header row, joined with denormalised display fields.
type MaterialRequest struct {
	ID             string     `json:"id"`
	RequestNo      string     `json:"requestNo"`
	Type           string     `json:"type"`

	ProjectID      *string    `json:"projectId,omitempty"`
	ProjectCode    *string    `json:"projectCode,omitempty"`
	ProjectName    *string    `json:"project,omitempty"` // matches frontend mock field

	Priority       string     `json:"priority"`
	Reason         string     `json:"reason"`
	Status         string     `json:"status"`

	RequesterID    string     `json:"requesterId"`
	RequesterName  string     `json:"requester"` // matches frontend mock field

	ApproverID     *string    `json:"approverId,omitempty"`
	ApproverName   *string    `json:"approvedBy,omitempty"`
	ApprovalNotes  *string    `json:"approvalNotes,omitempty"`
	ApprovedAt     *time.Time `json:"approvedAt,omitempty"`

	RequestDate    string     `json:"date"` // YYYY-MM-DD, matches frontend
	Items          []Item     `json:"items"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

type Item struct {
	ID         string  `json:"id"`
	MaterialID string  `json:"materialId"`
	SKU        string  `json:"sku"`
	Name       string  `json:"name"`
	Unit       string  `json:"unit"`
	Qty        int     `json:"qty"`
	Notes      *string `json:"notes,omitempty"`
}

// CreateRequest is the payload for POST /api/material-requests.
type CreateRequest struct {
	Type      string         `json:"type"`     // optional, defaults to 'Material Request'
	ProjectID *string        `json:"projectId"`
	Priority  string         `json:"priority"` // optional, defaults to 'medium'
	Reason    string         `json:"reason"`
	Items     []ItemInput    `json:"items"`
}

type ItemInput struct {
	MaterialID string  `json:"materialId"`
	Qty        int     `json:"qty"`
	Notes      *string `json:"notes,omitempty"`
}

// ApprovalRequest is the payload for approve/reject.
type ApprovalRequest struct {
	Notes *string `json:"notes,omitempty"`
}

// ListFilters controls GET /api/material-requests.
type ListFilters struct {
	Status      string
	Type        string
	Priority    string
	ProjectID   string
	RequesterID string // restrict to owner (used for staff role)
	Limit       int
}

func IsValidStatus(s string) bool {
	switch s {
	case StatusPending, StatusApproved, StatusRejected:
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

func IsValidType(s string) bool {
	switch s {
	case TypeMaterial, TypeTool, TypePurchase:
		return true
	}
	return false
}
