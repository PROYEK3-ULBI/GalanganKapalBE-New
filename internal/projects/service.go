package projects

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

type ErrValidation struct{ Msg string }

func (e *ErrValidation) Error() string { return e.Msg }

func newValidationError(msg string) error { return &ErrValidation{Msg: msg} }

func IsValidation(err error) bool {
	var v *ErrValidation
	return errors.As(err, &v)
}

type Service struct {
	repo     *Repository
	activity ActivityLogger // optional
}

// ActivityLogger is the minimal interface this module needs from activitylog.
type ActivityLogger interface {
	LogProjectCreated(ctx context.Context, actorID, projectID, code, name string)
	LogProjectUpdated(ctx context.Context, actorID, projectID, code, name string)
	LogProjectDeleted(ctx context.Context, actorID, projectID, code, name string)
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// SetActivityLogger wires an optional logger for audit trail.
func (s *Service) SetActivityLogger(l ActivityLogger) {
	s.activity = l
}

func (s *Service) List(ctx context.Context, f ListFilters) ([]Project, error) {
	return s.repo.List(ctx, f)
}

func (s *Service) Get(ctx context.Context, id string) (*Project, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) Create(ctx context.Context, req CreateRequest, createdByID *string) (*Project, error) {
	code := strings.TrimSpace(strings.ToUpper(req.Code))
	name := strings.TrimSpace(req.Name)
	ptype := strings.TrimSpace(req.Type)

	if code == "" {
		return nil, newValidationError("code is required")
	}
	if name == "" {
		return nil, newValidationError("name is required")
	}
	if ptype == "" {
		return nil, newValidationError("type is required")
	}

	p := &Project{
		Code:          code,
		Name:          name,
		Type:          ptype,
		Status:        "Planning",
		CompletionPct: 0,
	}
	if req.Status != nil {
		st := strings.TrimSpace(*req.Status)
		if !IsValidStatus(st) {
			return nil, newValidationError(fmt.Sprintf("invalid status %q", st))
		}
		p.Status = st
	}
	if req.CompletionPct != nil {
		if *req.CompletionPct < 0 || *req.CompletionPct > 100 {
			return nil, newValidationError("completion must be between 0 and 100")
		}
		p.CompletionPct = *req.CompletionPct
	}
	if req.StartDate != nil && *req.StartDate != "" {
		if _, err := time.Parse("2006-01-02", *req.StartDate); err != nil {
			return nil, newValidationError("startDate must be YYYY-MM-DD")
		}
		p.StartDate = req.StartDate
	}
	if req.TargetDate != nil && *req.TargetDate != "" {
		if _, err := time.Parse("2006-01-02", *req.TargetDate); err != nil {
			return nil, newValidationError("targetDate must be YYYY-MM-DD")
		}
		p.TargetDate = req.TargetDate
	}
	if req.Notes != nil {
		trimmed := strings.TrimSpace(*req.Notes)
		if trimmed != "" {
			p.Notes = &trimmed
		}
	}
	created, err := s.repo.Create(ctx, p, createdByID)
	if err != nil {
		return nil, err
	}
	if s.activity != nil && created != nil {
		actor := ""
		if createdByID != nil {
			actor = *createdByID
		}
		s.activity.LogProjectCreated(ctx, actor, created.ID, created.Code, created.Name)
	}
	return created, nil
}

func (s *Service) Update(ctx context.Context, id string, req UpdateRequest, actorID string) (*Project, error) {
	if req.Name != nil && strings.TrimSpace(*req.Name) == "" {
		return nil, newValidationError("name cannot be empty")
	}
	if req.Type != nil && strings.TrimSpace(*req.Type) == "" {
		return nil, newValidationError("type cannot be empty")
	}
	if req.Status != nil {
		st := strings.TrimSpace(*req.Status)
		if !IsValidStatus(st) {
			return nil, newValidationError(fmt.Sprintf("invalid status %q", st))
		}
		req.Status = &st
	}
	if req.CompletionPct != nil {
		if *req.CompletionPct < 0 || *req.CompletionPct > 100 {
			return nil, newValidationError("completion must be between 0 and 100")
		}
	}
	if req.StartDate != nil && *req.StartDate != "" {
		if _, err := time.Parse("2006-01-02", *req.StartDate); err != nil {
			return nil, newValidationError("startDate must be YYYY-MM-DD")
		}
	}
	if req.TargetDate != nil && *req.TargetDate != "" {
		if _, err := time.Parse("2006-01-02", *req.TargetDate); err != nil {
			return nil, newValidationError("targetDate must be YYYY-MM-DD")
		}
	}
	if req.Notes != nil {
		trimmed := strings.TrimSpace(*req.Notes)
		req.Notes = &trimmed
	}
	updated, err := s.repo.Update(ctx, id, req)
	if err != nil {
		return nil, err
	}
	if s.activity != nil && updated != nil {
		s.activity.LogProjectUpdated(ctx, actorID, updated.ID, updated.Code, updated.Name)
	}
	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id, actorID string) error {
	var code, name string
	if existing, err := s.repo.FindByID(ctx, id); err == nil && existing != nil {
		code = existing.Code
		name = existing.Name
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	if s.activity != nil {
		s.activity.LogProjectDeleted(ctx, actorID, id, code, name)
	}
	return nil
}
