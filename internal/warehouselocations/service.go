package warehouselocations

import (
	"context"
	"errors"
	"strings"
)

type ErrValidation struct{ Msg string }

func (e *ErrValidation) Error() string { return e.Msg }

func newValidationError(msg string) error { return &ErrValidation{Msg: msg} }

func IsValidation(err error) bool {
	var v *ErrValidation
	return errors.As(err, &v)
}

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, f ListFilters) ([]Location, error) {
	return s.repo.List(ctx, f)
}

func (s *Service) Get(ctx context.Context, id string) (*Location, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (*Location, error) {
	code := strings.ToUpper(strings.TrimSpace(req.Code))
	if code == "" {
		return nil, newValidationError("code is required")
	}
	loc := &Location{
		Code:   code,
		Status: "active",
	}
	if req.Name != nil {
		loc.Name = strings.TrimSpace(*req.Name)
	}
	if loc.Name == "" {
		loc.Name = code // default name to code
	}
	if req.Type != nil {
		loc.Type = strings.TrimSpace(*req.Type)
	}
	if req.Capacity != nil && *req.Capacity > 0 {
		v := *req.Capacity
		loc.Capacity = &v
	}
	if req.Notes != nil {
		loc.Notes = strings.TrimSpace(*req.Notes)
	}
	if req.Status != nil {
		st := strings.ToLower(strings.TrimSpace(*req.Status))
		if !IsValidStatus(st) {
			return nil, newValidationError("status must be 'active' or 'inactive'")
		}
		loc.Status = st
	}
	return s.repo.Create(ctx, loc)
}

func (s *Service) Update(ctx context.Context, id string, req UpdateRequest) (*Location, error) {
	if req.Status != nil {
		st := strings.ToLower(strings.TrimSpace(*req.Status))
		if !IsValidStatus(st) {
			return nil, newValidationError("status must be 'active' or 'inactive'")
		}
		req.Status = &st
	}
	return s.repo.Update(ctx, id, req)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
