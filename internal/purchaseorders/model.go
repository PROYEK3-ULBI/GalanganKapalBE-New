package purchaseorders

import "time"

// Allowed status transitions enforced at the DB level.
const (
	StatusDraft              = "Draft"
	StatusPending            = "Pending"
	StatusPartiallyReceived  = "Partially Received"
	StatusCompleted          = "Completed"
	StatusCancelled          = "Cancelled"
)

// PurchaseOrder is the header row plus a snapshot of vendor info and items.
type PurchaseOrder struct {
	ID          string    `json:"id"`
	PONumber    string    `json:"poNumber"`
	VendorID    string    `json:"vendorId"`
	VendorName  string    `json:"vendor"` // matches frontend mockData field name
	OrderDate   string    `json:"date"`   // ISO YYYY-MM-DD, matches frontend
	Status      string    `json:"status"`
	Notes       *string   `json:"notes,omitempty"`
	CreatedBy   *string   `json:"createdBy,omitempty"`
	Items       []Item    `json:"items"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Item is a single line on a purchase order.
type Item struct {
	ID          string    `json:"id"`
	MaterialID  string    `json:"materialId"`
	SKU         string    `json:"sku"`
	Name        string    `json:"name"`
	Unit        string    `json:"unit"`
	Ordered     int       `json:"ordered"`
	Received    int       `json:"received"`
	UnitPrice   float64   `json:"unitPrice"`
	Notes       *string   `json:"notes,omitempty"`
}

// CreateRequest is the payload for POST /api/purchase-orders.
type CreateRequest struct {
	PONumber string             `json:"poNumber"` // optional; auto-generated if empty
	VendorID string             `json:"vendorId"`
	Date     string             `json:"date"` // optional, defaults to today
	Status   string             `json:"status"` // defaults to Draft
	Notes    *string            `json:"notes"`
	Items    []CreateItemInput  `json:"items"`
}

// CreateItemInput describes a single line in a create request.
type CreateItemInput struct {
	MaterialID string  `json:"materialId"`
	Ordered    int     `json:"ordered"`
	UnitPrice  float64 `json:"unitPrice"`
	Notes      *string `json:"notes"`
}

// UpdateRequest applies partial updates to header fields only.
// Items must be modified through dedicated endpoints to keep audit clean.
type UpdateRequest struct {
	VendorID *string `json:"vendorId"`
	Date     *string `json:"date"`
	Status   *string `json:"status"`
	Notes    *string `json:"notes"`
}

// ListFilters controls filtering for GET /api/purchase-orders.
type ListFilters struct {
	Search   string // matches PO number or vendor name
	Status   string
	VendorID string
}

// IsValidStatus reports whether the given status is allowed.
func IsValidStatus(s string) bool {
	switch s {
	case StatusDraft, StatusPending, StatusPartiallyReceived, StatusCompleted, StatusCancelled:
		return true
	}
	return false
}
