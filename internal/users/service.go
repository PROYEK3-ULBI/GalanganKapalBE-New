package users

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
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

func (s *Service) List(ctx context.Context, f ListFilters) ([]User, error) {
	return s.repo.List(ctx, f)
}

func (s *Service) Get(ctx context.Context, id string) (*User, error) {
	return s.repo.FindByID(ctx, id)
}

// Create validates the request, hashes the password, and inserts the new user.
func (s *Service) Create(ctx context.Context, req CreateRequest) (*User, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	name := strings.TrimSpace(req.Name)
	role := strings.TrimSpace(req.Role)

	if email == "" || !strings.Contains(email, "@") {
		return nil, newValidationError("valid email is required")
	}
	if len(req.Password) < 6 {
		return nil, newValidationError("password must be at least 6 characters")
	}
	if name == "" {
		return nil, newValidationError("name is required")
	}
	if !IsValidRole(role) {
		return nil, newValidationError(fmt.Sprintf("invalid role %q (admin/supervisor/staff)", role))
	}

	u := &User{
		Email:  email,
		Name:   name,
		Role:   role,
		Status: "active",
	}
	if req.Status != nil {
		st := strings.ToLower(strings.TrimSpace(*req.Status))
		if !IsValidStatus(st) {
			return nil, newValidationError("status must be 'active' or 'inactive'")
		}
		u.Status = st
	}
	if req.Avatar != nil {
		u.Avatar = strings.ToUpper(strings.TrimSpace(*req.Avatar))
	}
	if req.Department != nil {
		u.Department = strings.TrimSpace(*req.Department)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	return s.repo.Create(ctx, u, string(hash))
}

func (s *Service) Update(ctx context.Context, id string, req UpdateRequest) (*User, error) {
	if req.Name != nil && strings.TrimSpace(*req.Name) == "" {
		return nil, newValidationError("name cannot be empty")
	}
	if req.Role != nil && !IsValidRole(strings.TrimSpace(*req.Role)) {
		return nil, newValidationError("invalid role")
	}
	if req.Status != nil {
		st := strings.ToLower(strings.TrimSpace(*req.Status))
		if !IsValidStatus(st) {
			return nil, newValidationError("status must be 'active' or 'inactive'")
		}
		req.Status = &st
	}
	return s.repo.Update(ctx, id, req)
}

// ToggleStatus flips active <-> inactive when explicit is nil, otherwise sets the
// status to the requested value (must be 'active' or 'inactive').
func (s *Service) ToggleStatus(ctx context.Context, id string, explicit *string) (*User, error) {
	var target string
	if explicit != nil {
		target = strings.ToLower(strings.TrimSpace(*explicit))
		if !IsValidStatus(target) {
			return nil, newValidationError("status must be 'active' or 'inactive'")
		}
	} else {
		current, err := s.repo.GetCurrentStatus(ctx, id)
		if err != nil {
			return nil, err
		}
		if current == "active" {
			target = "inactive"
		} else {
			target = "active"
		}
	}
	return s.repo.SetStatus(ctx, id, target)
}

// ResetPassword sets a new password for the user. password must be >=6 chars.
func (s *Service) ResetPassword(ctx context.Context, id, password string) error {
	if len(password) < 6 {
		return newValidationError("password must be at least 6 characters")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	return s.repo.SetPassword(ctx, id, string(hash))
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
