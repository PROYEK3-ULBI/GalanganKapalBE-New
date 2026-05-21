package purchaseorders

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
	LogPOCreated(ctx context.Context, actorID, poID, poNumber, vendorName string)
	LogPOUpdated(ctx context.Context, actorID, poID, poNumber, status string)
	LogPODeleted(ctx context.Context, actorID, poID, poNumber string)
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// SetActivityLogger wires an optional logger for audit trail.
func (s *Service) SetActivityLogger(l ActivityLogger) {
	s.activity = l
}

func (s *Service) List(ctx context.Context, f ListFilters) ([]PurchaseOrder, error) {
	return s.repo.List(ctx, f)
}

func (s *Service) Get(ctx context.Context, id string) (*PurchaseOrder, error) {
	return s.repo.FindByID(ctx, id)
}

// Stats returns aggregate counts for the dashboard.
type Stats struct {
	Total     int `json:"total"`
	Active    int `json:"active"`
	Completed int `json:"completed"`
	Draft     int `json:"draft"`
}

func (s *Service) Stats(ctx context.Context) (*Stats, error) {
	all, err := s.repo.List(ctx, ListFilters{})
	if err != nil {
		return nil, err
	}
	st := &Stats{Total: len(all)}
	for _, po := range all {
		switch po.Status {
		case StatusCompleted:
			st.Completed++
		case StatusDraft:
			st.Draft++
		default:
			st.Active++
		}
	}
	return st, nil
}

// Create validates and inserts a new PO with its items in one transaction.
// createdByID is the authenticated user's UUID, or nil if unknown.
func (s *Service) Create(ctx context.Context, req CreateRequest, createdByID *string) (*PurchaseOrder, error) {
	if strings.TrimSpace(req.VendorID) == "" {
		return nil, newValidationError("vendorId is required")
	}
	if len(req.Items) == 0 {
		return nil, newValidationError("at least one item is required")
	}

	po := &PurchaseOrder{
		VendorID: req.VendorID,
		Status:   StatusDraft,
	}

	// PO number: auto-generate if empty.
	if num := strings.TrimSpace(req.PONumber); num != "" {
		po.PONumber = num
	} else {
		next, err := s.repo.NextPONumber(ctx)
		if err != nil {
			return nil, err
		}
		po.PONumber = next
	}

	// Order date: default to today.
	if d := strings.TrimSpace(req.Date); d != "" {
		if _, err := time.Parse("2006-01-02", d); err != nil {
			return nil, newValidationError("date must be in YYYY-MM-DD format")
		}
		po.OrderDate = d
	} else {
		po.OrderDate = time.Now().Format("2006-01-02")
	}

	// Status: default to Draft, validate if provided.
	if st := strings.TrimSpace(req.Status); st != "" {
		if !IsValidStatus(st) {
			return nil, newValidationError(fmt.Sprintf("invalid status %q", st))
		}
		po.Status = st
	}

	if req.Notes != nil {
		trimmed := strings.TrimSpace(*req.Notes)
		if trimmed != "" {
			po.Notes = &trimmed
		}
	}

	for i, it := range req.Items {
		if strings.TrimSpace(it.MaterialID) == "" {
			return nil, newValidationError(fmt.Sprintf("items[%d].materialId is required", i))
		}
		if it.Ordered <= 0 {
			return nil, newValidationError(fmt.Sprintf("items[%d].ordered must be > 0", i))
		}
		if it.UnitPrice < 0 {
			return nil, newValidationError(fmt.Sprintf("items[%d].unitPrice must be >= 0", i))
		}
		var notes *string
		if it.Notes != nil {
			t := strings.TrimSpace(*it.Notes)
			if t != "" {
				notes = &t
			}
		}
		po.Items = append(po.Items, Item{
			MaterialID: it.MaterialID,
			Ordered:    it.Ordered,
			UnitPrice:  it.UnitPrice,
			Notes:      notes,
		})
	}

	return s.repo.Create(ctx, po, createdByID)
}

// CreateAndLog wraps Create with audit logging using the actor user ID.
func (s *Service) CreateAndLog(ctx context.Context, req CreateRequest, createdByID *string) (*PurchaseOrder, error) {
	created, err := s.Create(ctx, req, createdByID)
	if err != nil {
		return nil, err
	}
	if s.activity != nil && created != nil {
		actor := ""
		if createdByID != nil {
			actor = *createdByID
		}
		s.activity.LogPOCreated(ctx, actor, created.ID, created.PONumber, created.VendorName)
	}
	return created, nil
}

// Update applies partial header updates.
func (s *Service) Update(ctx context.Context, id string, req UpdateRequest, actorID string) (*PurchaseOrder, error) {
	if req.Date != nil {
		if _, err := time.Parse("2006-01-02", *req.Date); err != nil {
			return nil, newValidationError("date must be in YYYY-MM-DD format")
		}
	}
	if req.Status != nil {
		if !IsValidStatus(*req.Status) {
			return nil, newValidationError(fmt.Sprintf("invalid status %q", *req.Status))
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
		s.activity.LogPOUpdated(ctx, actorID, updated.ID, updated.PONumber, updated.Status)
	}
	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id, actorID string) error {
	var poNumber string
	if existing, err := s.repo.FindByID(ctx, id); err == nil && existing != nil {
		poNumber = existing.PONumber
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	if s.activity != nil {
		s.activity.LogPODeleted(ctx, actorID, id, poNumber)
	}
	return nil
}
