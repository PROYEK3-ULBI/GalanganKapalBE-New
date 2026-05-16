package reports

import "context"

// Service is a thin pass-through layer for now; aggregations live in repository
// because they're pure SQL. We expose a service to keep the layered convention
// consistent and to allow future caching/filter logic.
type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Summary(ctx context.Context) (*Summary, error) {
	return s.repo.Summary(ctx)
}

func (s *Service) StockValuation(ctx context.Context) ([]StockValuation, error) {
	return s.repo.StockValuation(ctx)
}

func (s *Service) CategoryBreakdown(ctx context.Context) ([]CategoryBreakdown, error) {
	return s.repo.CategoryBreakdown(ctx)
}

func (s *Service) TransactionSummary(ctx context.Context, startDate, endDate string) ([]TransactionSummary, error) {
	return s.repo.TransactionSummary(ctx, startDate, endDate)
}

func (s *Service) ProjectConsumption(ctx context.Context) ([]ProjectConsumption, error) {
	return s.repo.ProjectConsumption(ctx)
}

func (s *Service) InventoryTrend(ctx context.Context, days int) ([]InventoryTrendPoint, error) {
	return s.repo.InventoryTrend(ctx, days)
}
