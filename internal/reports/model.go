package reports

import "time"

// Summary is the top-level KPI snapshot for the Reports page.
type Summary struct {
	TotalInventoryValue float64 `json:"totalInventoryValue"`
	TotalItems          int     `json:"totalItems"`
	MaterialCount       int     `json:"materialCount"`
	LowStockCount       int     `json:"lowStockCount"`
	OutOfStockCount     int     `json:"outOfStockCount"`
	HazmatCount         int     `json:"hazmatCount"`
}

// StockValuation is one row in the stock valuation table.
type StockValuation struct {
	ID         string  `json:"id"`
	SKU        string  `json:"sku"`
	Name       string  `json:"name"`
	Category   string  `json:"category"`
	Unit       string  `json:"unit"`
	Stock      int     `json:"stock"`
	MinStock   int     `json:"minStock"`
	Price      float64 `json:"price"`
	TotalValue float64 `json:"totalValue"`
	Status     string  `json:"status"` // computed
	Hazmat     bool    `json:"hazmat"`
}

// CategoryBreakdown groups materials by category with aggregates.
type CategoryBreakdown struct {
	Category   string  `json:"category"`
	ItemCount  int     `json:"items"`
	TotalQty   int     `json:"qty"`
	TotalValue float64 `json:"value"`
}

// TransactionSummary is the aggregate count per transaction type.
type TransactionSummary struct {
	Type     string `json:"type"`
	Label    string `json:"label"`
	Count    int    `json:"count"`
	TotalQty int    `json:"totalQty"`
}

// ProjectConsumption shows aggregate material outflow per project.
type ProjectConsumption struct {
	ProjectID   string `json:"projectId"`
	ProjectCode string `json:"project"`
	ProjectName string `json:"projectName"`
	TotalQty    int    `json:"totalQty"`
	TxCount     int    `json:"txCount"`
}

// InventoryTrendPoint is one day on the inventory trend chart.
type InventoryTrendPoint struct {
	Date     time.Time `json:"-"`
	DateStr  string    `json:"date"`     // YYYY-MM-DD
	Inbound  int       `json:"inbound"`  // sum of receipt + return qty
	Outbound int       `json:"outbound"` // sum of issue + scrap qty
}
