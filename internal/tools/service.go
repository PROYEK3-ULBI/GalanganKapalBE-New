package tools

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
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, f ListFilters) ([]Tool, error) {
	return s.repo.List(ctx, f)
}

func (s *Service) Get(ctx context.Context, id string) (*Tool, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) History(ctx context.Context, toolID string) ([]HistoryEntry, error) {
	return s.repo.History(ctx, toolID)
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (*Tool, error) {
	sku := strings.TrimSpace(req.SKU)
	name := strings.TrimSpace(req.Name)
	category := strings.TrimSpace(req.Category)
	if sku == "" {
		return nil, newValidationError("sku is required")
	}
	if name == "" {
		return nil, newValidationError("name is required")
	}
	if category == "" {
		return nil, newValidationError("category is required")
	}

	t := &Tool{
		SKU:       sku,
		Name:      name,
		Category:  category,
		Status:    StatusAvailable,
		Condition: ConditionGood,
	}
	if req.Status != nil {
		st := strings.TrimSpace(*req.Status)
		if !IsValidStatus(st) {
			return nil, newValidationError(fmt.Sprintf("invalid status %q", st))
		}
		// Note: a tool created with 'In Use' would violate the borrower CHECK; we skip
		// allowing this and force the user to use the checkout endpoint.
		if st == StatusInUse {
			return nil, newValidationError("cannot create a tool already 'In Use'; use checkout endpoint")
		}
		t.Status = st
	}
	if req.Condition != nil {
		c := strings.TrimSpace(*req.Condition)
		if !IsValidCondition(c) {
			return nil, newValidationError(fmt.Sprintf("invalid condition %q", c))
		}
		t.Condition = c
	}
	if req.Location != nil {
		v := strings.TrimSpace(*req.Location)
		if v != "" {
			t.Location = &v
		}
	}
	if req.CalibrationDueDate != nil && *req.CalibrationDueDate != "" {
		v := strings.TrimSpace(*req.CalibrationDueDate)
		if _, err := time.Parse("2006-01-02", v); err != nil {
			return nil, newValidationError("calibrationDue must be YYYY-MM-DD")
		}
		t.CalibrationDueDate = &v
	}
	if req.Notes != nil {
		v := strings.TrimSpace(*req.Notes)
		if v != "" {
			t.Notes = &v
		}
	}
	if req.ImageURL != nil {
		v := strings.TrimSpace(*req.ImageURL)
		if v != "" {
			t.ImageURL = &v
		}
	}
	return s.repo.Create(ctx, t)
}

func (s *Service) Update(ctx context.Context, id string, req UpdateRequest) (*Tool, error) {
	if req.Name != nil && strings.TrimSpace(*req.Name) == "" {
		return nil, newValidationError("name cannot be empty")
	}
	if req.Category != nil && strings.TrimSpace(*req.Category) == "" {
		return nil, newValidationError("category cannot be empty")
	}
	if req.Condition != nil && !IsValidCondition(strings.TrimSpace(*req.Condition)) {
		return nil, newValidationError("invalid condition")
	}
	if req.CalibrationDueDate != nil && *req.CalibrationDueDate != "" {
		if _, err := time.Parse("2006-01-02", *req.CalibrationDueDate); err != nil {
			return nil, newValidationError("calibrationDue must be YYYY-MM-DD")
		}
	}
	return s.repo.Update(ctx, id, req)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// Checkout assigns the tool to the given borrower (default: actingUserID).
func (s *Service) Checkout(ctx context.Context, toolID string, actingUserID string, req CheckoutRequest) (*Tool, error) {
	borrowerID := actingUserID
	if req.BorrowerID != nil && strings.TrimSpace(*req.BorrowerID) != "" {
		borrowerID = strings.TrimSpace(*req.BorrowerID)
	}
	if borrowerID == "" {
		return nil, newValidationError("borrower is required")
	}
	return s.repo.Checkout(ctx, toolID, borrowerID, trimPtr(req.Notes))
}

func (s *Service) Return(ctx context.Context, toolID string, actingUserID *string, req ReturnRequest) (*Tool, error) {
	if req.Condition != nil && *req.Condition != "" && !IsValidCondition(*req.Condition) {
		return nil, newValidationError("invalid condition")
	}
	return s.repo.Return(ctx, toolID, actingUserID, trimPtr(req.Condition), trimPtr(req.Notes))
}

// Maintenance moves a tool to Maintenance status.
func (s *Service) Maintenance(ctx context.Context, toolID string, actingUserID *string, req MaintenanceRequest) (*Tool, error) {
	if req.Condition != nil && *req.Condition != "" && !IsValidCondition(*req.Condition) {
		return nil, newValidationError("invalid condition")
	}
	return s.repo.SetStatus(ctx, toolID, StatusMaintenance, actingUserID, trimPtr(req.Condition), trimPtr(req.Notes))
}

// MakeAvailable moves a tool from Maintenance back to Available.
func (s *Service) MakeAvailable(ctx context.Context, toolID string, actingUserID *string, notes *string) (*Tool, error) {
	return s.repo.SetStatus(ctx, toolID, StatusAvailable, actingUserID, nil, trimPtr(notes))
}

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
