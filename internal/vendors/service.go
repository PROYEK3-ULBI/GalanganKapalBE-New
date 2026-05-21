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
	repo     *Repository
	activity ActivityLogger // optional
}

// ActivityLogger is the minimal interface this module needs from activitylog.
type ActivityLogger interface {
	LogVendorCreated(ctx context.Context, actorID, vendorID, name string)
	LogVendorUpdated(ctx context.Context, actorID, vendorID, name string)
	LogVendorDeleted(ctx context.Context, actorID, vendorID, name string)
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// SetActivityLogger wires an optional logger for audit trail.
func (s *Service) SetActivityLogger(l ActivityLogger) {
	s.activity = l
}

func (s *Service) List(ctx context.Context, f ListFilters) ([]Vendor, error) {
	return s.repo.List(ctx, f)
}

func (s *Service) Get(ctx context.Context, id string) (*Vendor, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) Create(ctx context.Context, req CreateRequest, actorID string) (*Vendor, error) {
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
	created, err := s.repo.Create(ctx, v)
	if err != nil {
		return nil, err
	}
	if s.activity != nil && created != nil {
		s.activity.LogVendorCreated(ctx, actorID, created.ID, created.Name)
	}
	return created, nil
}

func (s *Service) Update(ctx context.Context, id string, req UpdateRequest, actorID string) (*Vendor, error) {
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
	updated, err := s.repo.Update(ctx, id, req)
	if err != nil {
		return nil, err
	}
	if s.activity != nil && updated != nil {
		s.activity.LogVendorUpdated(ctx, actorID, updated.ID, updated.Name)
	}
	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id, actorID string) error {
	var name string
	if existing, err := s.repo.FindByID(ctx, id); err == nil && existing != nil {
		name = existing.Name
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	if s.activity != nil {
		s.activity.LogVendorDeleted(ctx, actorID, id, name)
	}
	return nil
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
