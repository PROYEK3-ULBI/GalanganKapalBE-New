package notifications

import (
	"context"
	"log"
	"strings"
)

// Service exposes notification operations. It is also the entry point used
// by other modules (auth, materialrequests, …) to fire user-facing notifications.
type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, f ListFilters) ([]Notification, error) {
	return s.repo.List(ctx, f)
}

func (s *Service) Stats(ctx context.Context, userID string) (*Stats, error) {
	return s.repo.Stats(ctx, userID)
}

func (s *Service) MarkRead(ctx context.Context, id, userID string) error {
	return s.repo.MarkRead(ctx, id, userID)
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) (int, error) {
	return s.repo.MarkAllRead(ctx, userID)
}

func (s *Service) Delete(ctx context.Context, id, userID string) error {
	return s.repo.Delete(ctx, id, userID)
}

// Notify sends a notification to a single user. Returns nil on best-effort success.
// Errors are logged but not propagated by callers that use NotifySafe.
func (s *Service) Notify(ctx context.Context, in CreateInput) error {
	if strings.TrimSpace(in.UserID) == "" {
		return nil
	}
	if !IsValidType(in.Type) && in.Type != "" {
		in.Type = TypeInfo
	}
	return s.repo.Create(ctx, in)
}

// NotifyRoles fans out a notification to every active user holding any of the given roles.
func (s *Service) NotifyRoles(ctx context.Context, roles []string, in CreateInput) error {
	ids, err := s.repo.UserIDsByRoles(ctx, roles)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	return s.repo.CreateBulk(ctx, ids, in)
}

// NotifySafe is a fire-and-forget wrapper used by other modules that don't
// want notification failures to break their main flow. Errors are logged.
func (s *Service) NotifySafe(ctx context.Context, in CreateInput) {
	if err := s.Notify(ctx, in); err != nil {
		log.Printf("[notification] failed to send to %s: %v", in.UserID, err)
	}
}

// NotifyRolesSafe is the fire-and-forget version of NotifyRoles.
func (s *Service) NotifyRolesSafe(ctx context.Context, roles []string, in CreateInput) {
	if err := s.NotifyRoles(ctx, roles, in); err != nil {
		log.Printf("[notification] failed to fan-out to %v: %v", roles, err)
	}
}
