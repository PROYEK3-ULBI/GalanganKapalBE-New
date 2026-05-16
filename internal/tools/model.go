package tools

import "time"

// Tool status constants.
const (
	StatusAvailable   = "Available"
	StatusInUse       = "In Use"
	StatusMaintenance = "Maintenance"
)

// Tool condition constants.
const (
	ConditionGood        = "Good"
	ConditionFair        = "Fair"
	ConditionNeedsRepair = "Needs Repair"
	ConditionOutOfOrder  = "Out of Order"
)

// Tool represents a row in the tools catalog joined with current borrower info.
type Tool struct {
	ID                 string     `json:"id"`
	SKU                string     `json:"sku"`
	Name               string     `json:"name"`
	Category           string     `json:"category"`
	Status             string     `json:"status"`
	Condition          string     `json:"condition"`
	Location           *string    `json:"location,omitempty"`
	BorrowerID         *string    `json:"borrowerId,omitempty"`
	BorrowerName       *string    `json:"borrower,omitempty"` // matches frontend
	BorrowDate         *string    `json:"borrowDate,omitempty"`
	CalibrationDueDate *string    `json:"calibrationDue,omitempty"` // matches frontend
	Notes              *string    `json:"notes,omitempty"`
	ImageURL           *string    `json:"image,omitempty"` // matches frontend
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
}

// HistoryEntry is one row in tool_history.
type HistoryEntry struct {
	ID        string    `json:"id"`
	ToolID    string    `json:"toolId"`
	Action    string    `json:"action"`
	UserID    *string   `json:"userId,omitempty"`
	UserName  *string   `json:"user,omitempty"`
	Notes     *string   `json:"notes,omitempty"`
	CreatedAt time.Time `json:"date"`
}

// CreateRequest payload.
type CreateRequest struct {
	SKU                string  `json:"sku"`
	Name               string  `json:"name"`
	Category           string  `json:"category"`
	Status             *string `json:"status"`
	Condition          *string `json:"condition"`
	Location           *string `json:"location"`
	CalibrationDueDate *string `json:"calibrationDue"`
	Notes              *string `json:"notes"`
	ImageURL           *string `json:"image"`
}

// UpdateRequest applies partial header updates.
// Status changes that trigger checkout/return must use the dedicated endpoints
// since they involve borrower bookkeeping.
type UpdateRequest struct {
	Name               *string `json:"name"`
	Category           *string `json:"category"`
	Condition          *string `json:"condition"`
	Location           *string `json:"location"`
	CalibrationDueDate *string `json:"calibrationDue"`
	Notes              *string `json:"notes"`
	ImageURL           *string `json:"image"`
}

// CheckoutRequest is POST /api/tools/:id/checkout.
type CheckoutRequest struct {
	BorrowerID *string `json:"borrowerId,omitempty"` // optional; defaults to current user
	Notes      *string `json:"notes,omitempty"`
}

// ReturnRequest is POST /api/tools/:id/return.
type ReturnRequest struct {
	Condition *string `json:"condition,omitempty"` // optional update
	Notes     *string `json:"notes,omitempty"`
}

// MaintenanceRequest is POST /api/tools/:id/maintenance.
type MaintenanceRequest struct {
	Condition *string `json:"condition,omitempty"`
	Notes     *string `json:"notes,omitempty"`
}

// ListFilters controls GET /api/tools.
type ListFilters struct {
	Search             string
	Status             string
	Category           string
	BorrowerID         string
	CalibrationDueOnly bool // tools whose calibration is due within 30 days
}

func IsValidStatus(s string) bool {
	switch s {
	case StatusAvailable, StatusInUse, StatusMaintenance:
		return true
	}
	return false
}

func IsValidCondition(s string) bool {
	switch s {
	case ConditionGood, ConditionFair, ConditionNeedsRepair, ConditionOutOfOrder:
		return true
	}
	return false
}
