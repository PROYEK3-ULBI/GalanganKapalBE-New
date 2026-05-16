package materialrequests

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
	repo     *Repository
	notifSvc NotificationSender
	activity ActivityLogger // optional
}

// NotificationSender is the minimal interface this module needs from the
// notifications package. Defined here to avoid a hard dependency.
type NotificationSender interface {
	NotifySafe(ctx context.Context, in NotifyInput)
	NotifyRolesSafe(ctx context.Context, roles []string, in NotifyInput)
}

// ActivityLogger is the minimal interface this module needs from activitylog.
type ActivityLogger interface {
	LogRequestCreated(ctx context.Context, requesterID, requesterName, requestNo, requestType, priority string)
	LogRequestApproved(ctx context.Context, approverID, approverName, requesterID, requestNo string)
	LogRequestRejected(ctx context.Context, approverID, approverName, requesterID, requestNo string)
}

// NotifyInput mirrors notifications.CreateInput so callers can stay decoupled.
type NotifyInput struct {
	UserID   string
	Title    string
	Message  string
	Type     string
	Link     string
	Category string
}

func NewService(repo *Repository, notifSvc NotificationSender) *Service {
	return &Service{repo: repo, notifSvc: notifSvc}
}

// SetActivityLogger wires an optional logger for audit trail.
func (s *Service) SetActivityLogger(l ActivityLogger) {
	s.activity = l
}

func (s *Service) List(ctx context.Context, f ListFilters) ([]MaterialRequest, error) {
	return s.repo.List(ctx, f)
}

func (s *Service) Get(ctx context.Context, id string) (*MaterialRequest, error) {
	return s.repo.FindByID(ctx, id)
}

// Create validates and inserts a new request.
// requesterID must be the authenticated user's UUID.
func (s *Service) Create(ctx context.Context, req CreateRequest, requesterID string) (*MaterialRequest, error) {
	if requesterID == "" {
		return nil, newValidationError("requester is required")
	}
	if strings.TrimSpace(req.Reason) == "" {
		return nil, newValidationError("reason is required")
	}
	if len(req.Items) == 0 {
		return nil, newValidationError("at least one item is required")
	}

	mr := &MaterialRequest{
		Type:     TypeMaterial,
		Priority: PriorityMedium,
		Reason:   strings.TrimSpace(req.Reason),
		Status:   StatusPending,
	}
	if t := strings.TrimSpace(req.Type); t != "" {
		if !IsValidType(t) {
			return nil, newValidationError(fmt.Sprintf("invalid type %q", t))
		}
		mr.Type = t
	}
	if p := strings.ToLower(strings.TrimSpace(req.Priority)); p != "" {
		if !IsValidPriority(p) {
			return nil, newValidationError(fmt.Sprintf("invalid priority %q", p))
		}
		mr.Priority = p
	}
	if req.ProjectID != nil {
		v := strings.TrimSpace(*req.ProjectID)
		if v != "" {
			mr.ProjectID = &v
		}
	}

	for i, it := range req.Items {
		if strings.TrimSpace(it.MaterialID) == "" {
			return nil, newValidationError(fmt.Sprintf("items[%d].materialId is required", i))
		}
		if it.Qty <= 0 {
			return nil, newValidationError(fmt.Sprintf("items[%d].qty must be > 0", i))
		}
		var notes *string
		if it.Notes != nil {
			t := strings.TrimSpace(*it.Notes)
			if t != "" {
				notes = &t
			}
		}
		mr.Items = append(mr.Items, Item{
			MaterialID: it.MaterialID,
			Qty:        it.Qty,
			Notes:      notes,
		})
	}

	// Auto-generate request number.
	num, err := s.repo.NextRequestNo(ctx)
	if err != nil {
		return nil, err
	}
	mr.RequestNo = num

	created, err := s.repo.Create(ctx, mr, requesterID)
	if err != nil {
		return nil, err
	}

	// Notify all admin/supervisor users that a new request needs review.
	if s.notifSvc != nil {
		s.notifSvc.NotifyRolesSafe(ctx, []string{"admin", "supervisor"}, NotifyInput{
			Title:    "Permintaan Material Baru",
			Message:  fmt.Sprintf("%s mengajukan %s (%s) — prioritas %s", created.RequesterName, created.RequestNo, created.Type, created.Priority),
			Type:     "info",
			Link:     "/material-request",
			Category: "material-request",
		})
	}

	// Audit trail.
	if s.activity != nil {
		s.activity.LogRequestCreated(ctx, requesterID, created.RequesterName, created.RequestNo, created.Type, created.Priority)
	}

	return created, nil
}

// Approve transitions a pending request to approved.
func (s *Service) Approve(ctx context.Context, id, approverID string, notes *string) (*MaterialRequest, error) {
	if approverID == "" {
		return nil, newValidationError("approver is required")
	}
	mr, err := s.repo.Decide(ctx, id, StatusApproved, approverID, trimPtr(notes))
	if err != nil {
		return nil, err
	}
	// Notify the requester.
	if s.notifSvc != nil && mr != nil {
		s.notifSvc.NotifySafe(ctx, NotifyInput{
			UserID:   mr.RequesterID,
			Title:    "Permintaan Disetujui",
			Message:  fmt.Sprintf("%s telah disetujui oleh %s", mr.RequestNo, deref(mr.ApproverName)),
			Type:     "success",
			Link:     "/material-request",
			Category: "material-request",
		})
	}
	// Audit trail.
	if s.activity != nil && mr != nil {
		s.activity.LogRequestApproved(ctx, approverID, deref(mr.ApproverName), mr.RequesterID, mr.RequestNo)
	}
	return mr, nil
}

// Reject transitions a pending request to rejected.
func (s *Service) Reject(ctx context.Context, id, approverID string, notes *string) (*MaterialRequest, error) {
	if approverID == "" {
		return nil, newValidationError("approver is required")
	}
	mr, err := s.repo.Decide(ctx, id, StatusRejected, approverID, trimPtr(notes))
	if err != nil {
		return nil, err
	}
	if s.notifSvc != nil && mr != nil {
		s.notifSvc.NotifySafe(ctx, NotifyInput{
			UserID:   mr.RequesterID,
			Title:    "Permintaan Ditolak",
			Message:  fmt.Sprintf("%s ditolak oleh %s", mr.RequestNo, deref(mr.ApproverName)),
			Type:     "warning",
			Link:     "/material-request",
			Category: "material-request",
		})
	}
	if s.activity != nil && mr != nil {
		s.activity.LogRequestRejected(ctx, approverID, deref(mr.ApproverName), mr.RequesterID, mr.RequestNo)
	}
	return mr, nil
}

// Delete removes a pending request owned by requesterID.
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

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
