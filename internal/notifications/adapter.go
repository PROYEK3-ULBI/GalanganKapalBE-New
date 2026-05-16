package notifications

import (
	"context"

	"github.com/PROYEK3-ULBI/sims-backend/internal/materialrequests"
)

// MaterialRequestsAdapter wraps the notifications Service to satisfy the
// NotificationSender interface defined in the materialrequests package.
//
// This indirection lets the materialrequests module avoid importing the
// notifications package directly (which would create a tight coupling and
// risk import cycles in the future).
type MaterialRequestsAdapter struct {
	svc *Service
}

func NewMaterialRequestsAdapter(svc *Service) *MaterialRequestsAdapter {
	return &MaterialRequestsAdapter{svc: svc}
}

func toCreateInput(in materialrequests.NotifyInput) CreateInput {
	return CreateInput{
		UserID:   in.UserID,
		Title:    in.Title,
		Message:  in.Message,
		Type:     in.Type,
		Link:     in.Link,
		Category: in.Category,
	}
}

func (a *MaterialRequestsAdapter) NotifySafe(ctx context.Context, in materialrequests.NotifyInput) {
	a.svc.NotifySafe(ctx, toCreateInput(in))
}

func (a *MaterialRequestsAdapter) NotifyRolesSafe(ctx context.Context, roles []string, in materialrequests.NotifyInput) {
	a.svc.NotifyRolesSafe(ctx, roles, toCreateInput(in))
}
