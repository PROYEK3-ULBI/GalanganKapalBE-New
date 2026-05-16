package transactions

import "time"

// Transaction types.
const (
	TypeReceipt = "receipt"
	TypeIssue   = "issue"
	TypeScrap   = "scrap"
	TypeReturn  = "return"
)

// Transaction is one row in the immutable ledger.
// All denormalised display fields are populated via JOINs in repository.
type Transaction struct {
	ID              string    `json:"id"`
	TransactionNo   string    `json:"transactionNo"`
	Type            string    `json:"type"`

	MaterialID      string    `json:"materialId"`
	SKU             string    `json:"sku"`
	MaterialName    string    `json:"material"` // matches frontend field
	Unit            string    `json:"unit"`
	Qty             int       `json:"qty"`

	ProjectID       *string   `json:"projectId,omitempty"`
	ProjectCode     *string   `json:"project,omitempty"`
	VendorID        *string   `json:"vendorId,omitempty"`
	VendorName      *string   `json:"vendor,omitempty"`
	PurchaseOrderID *string   `json:"purchaseOrderId,omitempty"`
	PONumber        *string   `json:"poNumber,omitempty"`
	POItemID        *string   `json:"poItemId,omitempty"`

	UserID          *string   `json:"userId,omitempty"`
	UserName        *string   `json:"user,omitempty"`

	HeatNumber      *string   `json:"heatNumber,omitempty"`
	Notes           *string   `json:"notes,omitempty"`

	TransactionDate time.Time `json:"date"`
	CreatedAt       time.Time `json:"createdAt"`
}

// ListFilters controls GET /api/transactions.
type ListFilters struct {
	Type       string // 'receipt'|'issue'|'scrap'|'return'
	MaterialID string
	ProjectID  string
	VendorID   string
	UserID     string
	StartDate  string // 'YYYY-MM-DD'
	EndDate    string
	Limit      int
}

// ReceiptItemInput represents one line of a goods receipt.
type ReceiptItemInput struct {
	POItemID   string  `json:"poItemId,omitempty"` // optional, for PO-linked receipts
	MaterialID string  `json:"materialId"`
	Qty        int     `json:"qty"`
	HeatNumber *string `json:"heatNumber,omitempty"`
	Notes      *string `json:"notes,omitempty"`
}

// ReceiptRequest is POST /api/goods-receipt.
// Either purchaseOrderId (link to PO) or vendorId (free-form) must be provided.
type ReceiptRequest struct {
	PurchaseOrderID *string            `json:"purchaseOrderId,omitempty"`
	VendorID        *string            `json:"vendorId,omitempty"`
	Items           []ReceiptItemInput `json:"items"`
	Notes           *string            `json:"notes,omitempty"`
	TransactionDate *string            `json:"date,omitempty"` // ISO; defaults to now
}

// IssueItemInput represents one line of a goods issue.
type IssueItemInput struct {
	MaterialID string  `json:"materialId"`
	Qty        int     `json:"qty"`
	HeatNumber *string `json:"heatNumber,omitempty"`
	Notes      *string `json:"notes,omitempty"`
}

// IssueRequest is POST /api/goods-issue.
type IssueRequest struct {
	ProjectID       string           `json:"projectId"`
	Items           []IssueItemInput `json:"items"`
	Mandor          *string          `json:"mandor,omitempty"`
	Notes           *string          `json:"notes,omitempty"`
	TransactionDate *string          `json:"date,omitempty"`
}

// ScrapReturnRequest is POST /api/scrap-return.
// returnType: 'scrap' (deduct stock) or 'return' (add stock).
type ScrapReturnRequest struct {
	ReturnType      string  `json:"type"` // "scrap" or "return"
	MaterialID      string  `json:"materialId"`
	ProjectID       *string `json:"projectId,omitempty"`
	Qty             int     `json:"qty"`
	Reason          string  `json:"reason"`
	HeatNumber      *string `json:"heatNumber,omitempty"`
	TransactionDate *string `json:"date,omitempty"`
}

// IsValidType reports whether the type string is one of the four allowed.
func IsValidType(t string) bool {
	switch t {
	case TypeReceipt, TypeIssue, TypeScrap, TypeReturn:
		return true
	}
	return false
}
