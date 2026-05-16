package support

import (
	"context"
	"errors"
	"fmt"
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

func (s *Service) List(ctx context.Context, f ListFilters) ([]Ticket, error) {
	return s.repo.List(ctx, f)
}

func (s *Service) Get(ctx context.Context, id string) (*Ticket, error) {
	return s.repo.FindByID(ctx, id)
}

// Submit creates a new ticket on behalf of userID.
func (s *Service) Submit(ctx context.Context, userID string, req CreateRequest) (*Ticket, error) {
	if userID == "" {
		return nil, newValidationError("requester is required")
	}
	subject := strings.TrimSpace(req.Subject)
	message := strings.TrimSpace(req.Message)
	if subject == "" {
		return nil, newValidationError("subject is required")
	}
	if message == "" {
		return nil, newValidationError("message is required")
	}

	priority := PriorityMedium
	if req.Priority != nil {
		p := strings.ToLower(strings.TrimSpace(*req.Priority))
		if !IsValidPriority(p) {
			return nil, newValidationError(fmt.Sprintf("invalid priority %q", p))
		}
		priority = p
	}

	num, err := s.repo.NextTicketNo(ctx)
	if err != nil {
		return nil, err
	}

	t := &Ticket{
		TicketNo: num,
		Subject:  subject,
		Message:  message,
		Priority: priority,
		Status:   StatusOpen,
	}
	return s.repo.Create(ctx, t, userID)
}

// Resolve closes a ticket with an optional admin response.
func (s *Service) Resolve(ctx context.Context, id, handlerID string, response *string) (*Ticket, error) {
	if handlerID == "" {
		return nil, newValidationError("handler is required")
	}
	return s.repo.SetStatus(ctx, id, StatusResolved, handlerID, trimPtr(response))
}

// Update changes status/priority. Admin only.
func (s *Service) UpdateStatus(ctx context.Context, id, handlerID, status string, response *string) (*Ticket, error) {
	if !IsValidStatus(status) {
		return nil, newValidationError(fmt.Sprintf("invalid status %q", status))
	}
	return s.repo.SetStatus(ctx, id, status, handlerID, trimPtr(response))
}

func (s *Service) Delete(ctx context.Context, id, requesterID string) error {
	return s.repo.Delete(ctx, id, requesterID)
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
