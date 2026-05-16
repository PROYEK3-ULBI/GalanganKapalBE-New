package materials

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// ErrValidation is returned when input fails validation rules.
type ErrValidation struct{ Msg string }

func (e *ErrValidation) Error() string { return e.Msg }

func newValidationError(msg string) error { return &ErrValidation{Msg: msg} }

// IsValidation reports whether an error is a validation error.
func IsValidation(err error) bool {
	var v *ErrValidation
	return errors.As(err, &v)
}

// Service contains business logic for materials.
type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// List returns materials matching the given filters.
func (s *Service) List(ctx context.Context, f ListFilters) ([]Material, error) {
	return s.repo.List(ctx, f)
}

// Get returns a single material.
func (s *Service) Get(ctx context.Context, id string) (*Material, error) {
	return s.repo.FindByID(ctx, id)
}

// Categories returns the distinct categories currently in use.
func (s *Service) Categories(ctx context.Context) ([]string, error) {
	return s.repo.ListCategories(ctx)
}

// Create validates and inserts a new material.
func (s *Service) Create(ctx context.Context, req CreateRequest) (*Material, error) {
	m, err := buildFromCreate(req)
	if err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, m)
}

// Update validates and applies a partial update.
func (s *Service) Update(ctx context.Context, id string, req UpdateRequest) (*Material, error) {
	if err := validateUpdate(req); err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, id, req)
}

// Delete removes a material.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func buildFromCreate(req CreateRequest) (*Material, error) {
	sku := strings.TrimSpace(strings.ToUpper(req.SKU))
	name := strings.TrimSpace(req.Name)
	category := strings.TrimSpace(req.Category)
	unit := strings.TrimSpace(req.Unit)

	if sku == "" {
		return nil, newValidationError("sku is required")
	}
	if name == "" {
		return nil, newValidationError("name is required")
	}
	if category == "" {
		return nil, newValidationError("category is required")
	}
	if unit == "" {
		return nil, newValidationError("unit is required")
	}

	m := &Material{
		SKU:      sku,
		Name:     name,
		Category: category,
		Unit:     unit,
	}
	if req.Stock != nil {
		if *req.Stock < 0 {
			return nil, newValidationError("stock must be >= 0")
		}
		m.Stock = *req.Stock
	}
	if req.MinStock != nil {
		if *req.MinStock < 0 {
			return nil, newValidationError("minStock must be >= 0")
		}
		m.MinStock = *req.MinStock
	}
	if req.ReorderPoint != nil {
		if *req.ReorderPoint < 0 {
			return nil, newValidationError("reorderPoint must be >= 0")
		}
		m.ReorderPoint = *req.ReorderPoint
	}
	if req.Price != nil {
		if *req.Price < 0 {
			return nil, newValidationError("price must be >= 0")
		}
		m.Price = *req.Price
	}
	if req.Hazmat != nil {
		m.Hazmat = *req.Hazmat
	}
	if req.HeatNumber != nil {
		trimmed := strings.TrimSpace(*req.HeatNumber)
		if trimmed != "" {
			m.HeatNumber = &trimmed
		}
	}
	if req.Location != nil {
		trimmed := strings.TrimSpace(*req.Location)
		if trimmed != "" {
			m.Location = &trimmed
		}
	}
	if req.Specifications != nil {
		trimmed := strings.TrimSpace(*req.Specifications)
		if trimmed != "" {
			m.Specifications = &trimmed
		}
	}
	return m, nil
}

func validateUpdate(req UpdateRequest) error {
	checkNonNeg := func(name string, p *int) error {
		if p != nil && *p < 0 {
			return newValidationError(fmt.Sprintf("%s must be >= 0", name))
		}
		return nil
	}
	if err := checkNonNeg("stock", req.Stock); err != nil {
		return err
	}
	if err := checkNonNeg("minStock", req.MinStock); err != nil {
		return err
	}
	if err := checkNonNeg("reorderPoint", req.ReorderPoint); err != nil {
		return err
	}
	if req.Price != nil && *req.Price < 0 {
		return newValidationError("price must be >= 0")
	}
	return nil
}
