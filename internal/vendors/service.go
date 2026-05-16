package vendors

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

func (s *Service) List(ctx context.Context, f ListFilters) ([]Vendor, error) {
	return s.repo.List(ctx, f)
}

func (s *Service) Get(ctx context.Context, id string) (*Vendor, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (*Vendor, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, newValidationError("name is required")
	}

	v := &Vendor{
		Name:    name,
		Contact: trimPtr(req.Contact),
		Phone:   trimPtr(req.Phone),
		Email:   trimPtr(req.Email),
		Address: trimPtr(req.Address),
		Status:  "active",
	}
	if req.Status != nil {
		st := strings.ToLower(strings.TrimSpace(*req.Status))
		if st != "active" && st != "inactive" {
			return nil, newValidationError("status must be 'active' or 'inactive'")
		}
		v.Status = st
	}
	return s.repo.Create(ctx, v)
}

func (s *Service) Update(ctx context.Context, id string, req UpdateRequest) (*Vendor, error) {
	if req.Name != nil && strings.TrimSpace(*req.Name) == "" {
		return nil, newValidationError("name cannot be empty")
	}
	if req.Status != nil {
		st := strings.ToLower(strings.TrimSpace(*req.Status))
		if st != "active" && st != "inactive" {
			return nil, newValidationError("status must be 'active' or 'inactive'")
		}
		req.Status = &st
	}
	return s.repo.Update(ctx, id, req)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// trimPtr returns nil if the trimmed value is empty, else a pointer to the trimmed string.
func trimPtr(p *string) *string {
	if p == nil {
		return nil
	}
	t := strings.TrimSpace(*p)
	if t == "" {
		return nil
	}
	return &t
}
