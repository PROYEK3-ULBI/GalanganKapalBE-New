package materials

import "time"

// Stock status values derived from stock vs min_stock at query time.
const (
	StatusInStock    = "In Stock"
	StatusLowStock   = "Low Stock"
	StatusOutOfStock = "Out of Stock"
)

// Material represents a row in the materials catalog.
type Material struct {
	ID             string    `json:"id"`
	SKU            string    `json:"sku"`
	Name           string    `json:"name"`
	Category       string    `json:"category"`
	Unit           string    `json:"unit"`
	Stock          int       `json:"stock"`
	MinStock       int       `json:"minStock"`
	ReorderPoint   int       `json:"reorderPoint"`
	Price          float64   `json:"price"`
	Hazmat         bool      `json:"hazmat"`
	HeatNumber     *string   `json:"heatNumber,omitempty"`
	Location       *string   `json:"location,omitempty"`
	Specifications *string   `json:"specifications,omitempty"`
	Status         string    `json:"status"` // derived
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// ComputeStatus derives the stock status from current stock and minStock thresholds.
func ComputeStatus(stock, minStock int) string {
	switch {
	case stock <= 0:
		return StatusOutOfStock
	case stock <= minStock:
		return StatusLowStock
	default:
		return StatusInStock
	}
}

// CreateRequest is the payload for POST /api/materials.
type CreateRequest struct {
	SKU            string  `json:"sku"`
	Name           string  `json:"name"`
	Category       string  `json:"category"`
	Unit           string  `json:"unit"`
	Stock          *int    `json:"stock"`
	MinStock       *int    `json:"minStock"`
	ReorderPoint   *int    `json:"reorderPoint"`
	Price          *float64 `json:"price"`
	Hazmat         *bool   `json:"hazmat"`
	HeatNumber     *string `json:"heatNumber"`
	Location       *string `json:"location"`
	Specifications *string `json:"specifications"`
}

// UpdateRequest is the payload for PUT /api/materials/:id.
// All fields are optional; nil means "do not change".
type UpdateRequest struct {
	Name           *string  `json:"name"`
	Category       *string  `json:"category"`
	Unit           *string  `json:"unit"`
	Stock          *int     `json:"stock"`
	MinStock       *int     `json:"minStock"`
	ReorderPoint   *int     `json:"reorderPoint"`
	Price          *float64 `json:"price"`
	Hazmat         *bool    `json:"hazmat"`
	HeatNumber     *string  `json:"heatNumber"`
	Location       *string  `json:"location"`
	Specifications *string  `json:"specifications"`
}

// ListFilters controls filtering for GET /api/materials.
type ListFilters struct {
	Search       string // matches SKU or Name (case-insensitive)
	Category     string // exact match
	HazmatOnly   bool
	LowStockOnly bool
}
