package reports

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Summary aggregates the headline KPIs in a single round-trip.
func (r *Repository) Summary(ctx context.Context) (*Summary, error) {
	var s Summary
	err := r.pool.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(stock * price), 0)::numeric           AS total_value,
			COALESCE(SUM(stock), 0)::int                       AS total_items,
			COUNT(*)::int                                      AS material_count,
			COUNT(*) FILTER (WHERE stock <= min_stock AND stock > 0)::int AS low_stock,
			COUNT(*) FILTER (WHERE stock = 0)::int             AS out_of_stock,
			COUNT(*) FILTER (WHERE hazmat)::int                AS hazmat_count
		FROM materials
	`).Scan(
		&s.TotalInventoryValue, &s.TotalItems, &s.MaterialCount,
		&s.LowStockCount, &s.OutOfStockCount, &s.HazmatCount,
	)
	if err != nil {
		return nil, fmt.Errorf("summary: %w", err)
	}
	return &s, nil
}

// StockValuation returns per-material stock × price, with computed status.
func (r *Repository) StockValuation(ctx context.Context) ([]StockValuation, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, sku, name, category, unit,
		       stock, min_stock, price, hazmat,
		       CASE
		           WHEN stock = 0 THEN 'Out of Stock'
		           WHEN stock <= min_stock THEN 'Low Stock'
		           ELSE 'In Stock'
		       END AS status
		FROM materials
		ORDER BY stock * price DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("stock valuation: %w", err)
	}
	defer rows.Close()

	out := make([]StockValuation, 0)
	for rows.Next() {
		var v StockValuation
		if err := rows.Scan(&v.ID, &v.SKU, &v.Name, &v.Category, &v.Unit,
			&v.Stock, &v.MinStock, &v.Price, &v.Hazmat, &v.Status); err != nil {
			return nil, err
		}
		v.TotalValue = float64(v.Stock) * v.Price
		out = append(out, v)
	}
	return out, rows.Err()
}

// CategoryBreakdown aggregates inventory grouped by category.
func (r *Repository) CategoryBreakdown(ctx context.Context) ([]CategoryBreakdown, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT category,
		       COUNT(*)::int                       AS item_count,
		       COALESCE(SUM(stock), 0)::int        AS total_qty,
		       COALESCE(SUM(stock * price), 0)::numeric AS total_value
		FROM materials
		GROUP BY category
		ORDER BY total_value DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("category breakdown: %w", err)
	}
	defer rows.Close()

	out := make([]CategoryBreakdown, 0)
	for rows.Next() {
		var c CategoryBreakdown
		if err := rows.Scan(&c.Category, &c.ItemCount, &c.TotalQty, &c.TotalValue); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// TransactionSummary returns count + total qty grouped by type, optionally
// filtered by date range. startDate/endDate are inclusive in YYYY-MM-DD; pass
// empty string to skip the filter.
func (r *Repository) TransactionSummary(ctx context.Context, startDate, endDate string) ([]TransactionSummary, error) {
	args := []any{}
	conds := ""
	if startDate != "" {
		args = append(args, startDate)
		conds += fmt.Sprintf(" AND transaction_date >= $%d", len(args))
	}
	if endDate != "" {
		args = append(args, endDate+" 23:59:59")
		conds += fmt.Sprintf(" AND transaction_date <= $%d", len(args))
	}

	rows, err := r.pool.Query(ctx, `
		SELECT type, COUNT(*)::int, COALESCE(SUM(qty), 0)::int
		FROM transactions
		WHERE 1=1`+conds+`
		GROUP BY type
		ORDER BY type
	`, args...)
	if err != nil {
		return nil, fmt.Errorf("transaction summary: %w", err)
	}
	defer rows.Close()

	out := make([]TransactionSummary, 0)
	for rows.Next() {
		var t TransactionSummary
		if err := rows.Scan(&t.Type, &t.Count, &t.TotalQty); err != nil {
			return nil, err
		}
		t.Label = labelForType(t.Type)
		out = append(out, t)
	}
	return out, rows.Err()
}

// ProjectConsumption aggregates issue+scrap transactions by project.
func (r *Repository) ProjectConsumption(ctx context.Context) ([]ProjectConsumption, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT p.id::text, p.code, p.name,
		       COALESCE(SUM(t.qty), 0)::int AS total_qty,
		       COUNT(t.id)::int             AS tx_count
		FROM projects p
		LEFT JOIN transactions t ON t.project_id = p.id AND t.type IN ('issue', 'scrap')
		GROUP BY p.id, p.code, p.name
		HAVING COUNT(t.id) > 0
		ORDER BY total_qty DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("project consumption: %w", err)
	}
	defer rows.Close()

	out := make([]ProjectConsumption, 0)
	for rows.Next() {
		var p ProjectConsumption
		if err := rows.Scan(&p.ProjectID, &p.ProjectCode, &p.ProjectName, &p.TotalQty, &p.TxCount); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// InventoryTrend returns daily aggregates for the last `days` days.
// Days with no activity are filled with zeros so the chart shows a continuous line.
func (r *Repository) InventoryTrend(ctx context.Context, days int) ([]InventoryTrendPoint, error) {
	if days <= 0 || days > 365 {
		days = 30
	}

	// Use a generated date series and LEFT JOIN aggregates so days without
	// transactions still produce a row.
	rows, err := r.pool.Query(ctx, `
		WITH days AS (
			SELECT generate_series(
				CURRENT_DATE - ($1::int - 1) * INTERVAL '1 day',
				CURRENT_DATE,
				INTERVAL '1 day'
			)::date AS d
		)
		SELECT d.d,
		       COALESCE(SUM(CASE WHEN t.type IN ('receipt', 'return') THEN t.qty END), 0)::int AS inbound,
		       COALESCE(SUM(CASE WHEN t.type IN ('issue',  'scrap')   THEN t.qty END), 0)::int AS outbound
		FROM days d
		LEFT JOIN transactions t ON t.transaction_date::date = d.d
		GROUP BY d.d
		ORDER BY d.d
	`, days)
	if err != nil {
		return nil, fmt.Errorf("inventory trend: %w", err)
	}
	defer rows.Close()

	out := make([]InventoryTrendPoint, 0, days)
	for rows.Next() {
		var p InventoryTrendPoint
		var d time.Time
		if err := rows.Scan(&d, &p.Inbound, &p.Outbound); err != nil {
			return nil, err
		}
		p.Date = d
		p.DateStr = d.Format("2006-01-02")
		out = append(out, p)
	}
	return out, rows.Err()
}

// labelForType returns the user-friendly Indonesian label for a transaction type.
func labelForType(t string) string {
	switch t {
	case "receipt":
		return "Penerimaan"
	case "issue":
		return "Pengeluaran"
	case "scrap":
		return "Scrap"
	case "return":
		return "Retur"
	}
	return t
}
