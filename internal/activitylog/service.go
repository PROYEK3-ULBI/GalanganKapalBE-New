package activitylog

import (
	"context"
	"log"
)

// Service is also the entry point used by other modules to log events.
type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, f ListFilters) ([]Entry, error) {
	return s.repo.List(ctx, f)
}

// Log inserts a new activity entry. Errors are returned to the caller.
func (s *Service) Log(ctx context.Context, in CreateInput) error {
	if !IsValidType(in.Type) && in.Type != "" {
		in.Type = TypeInfo
	}
	return s.repo.Create(ctx, in)
}

// LogSafe is the fire-and-forget variant. Errors are logged but not returned,
// so business logic stays unaffected when the audit trail write fails.
func (s *Service) LogSafe(ctx context.Context, in CreateInput) {
	if err := s.Log(ctx, in); err != nil {
		log.Printf("[activity-log] failed to record %q: %v", in.Action, err)
	}
}
