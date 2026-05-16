package activitylog

import (
	"context"
	"fmt"
)

// AuthAdapter satisfies the auth package's ActivityLogger interface.
type AuthAdapter struct {
	svc *Service
}

func NewAuthAdapter(svc *Service) *AuthAdapter { return &AuthAdapter{svc: svc} }

func (a *AuthAdapter) LogLogin(ctx context.Context, userID, userName, email string) {
	a.svc.LogSafe(ctx, CreateInput{
		Action:   "User Login",
		Detail:   fmt.Sprintf("%s (%s) logged in", userName, email),
		Type:     TypeInfo,
		UserID:   userID,
		Category: "auth",
	})
}

// MaterialRequestsAdapter satisfies the materialrequests package's ActivityLogger interface.
type MaterialRequestsAdapter struct {
	svc *Service
}

func NewMaterialRequestsAdapter(svc *Service) *MaterialRequestsAdapter {
	return &MaterialRequestsAdapter{svc: svc}
}

func (a *MaterialRequestsAdapter) LogRequestCreated(ctx context.Context, requesterID, requesterName, requestNo, requestType, priority string) {
	a.svc.LogSafe(ctx, CreateInput{
		Action:       "Material Request Submitted",
		Detail:       fmt.Sprintf("%s submitted %s (%s) with %s priority", requesterName, requestNo, requestType, priority),
		Type:         TypeInfo,
		UserID:       requesterID,
		ResourceType: "material_request",
		ResourceID:   requestNo,
		Category:     "material-request",
	})
}

func (a *MaterialRequestsAdapter) LogRequestApproved(ctx context.Context, approverID, approverName, requesterID, requestNo string) {
	a.svc.LogSafe(ctx, CreateInput{
		Action:       "Material Request Approved",
		Detail:       fmt.Sprintf("%s approved %s", approverName, requestNo),
		Type:         TypeSuccess,
		UserID:       approverID,
		ResourceType: "material_request",
		ResourceID:   requestNo,
		Category:     "material-request",
	})
}

func (a *MaterialRequestsAdapter) LogRequestRejected(ctx context.Context, approverID, approverName, requesterID, requestNo string) {
	a.svc.LogSafe(ctx, CreateInput{
		Action:       "Material Request Rejected",
		Detail:       fmt.Sprintf("%s rejected %s", approverName, requestNo),
		Type:         TypeWarning,
		UserID:       approverID,
		ResourceType: "material_request",
		ResourceID:   requestNo,
		Category:     "material-request",
	})
}
