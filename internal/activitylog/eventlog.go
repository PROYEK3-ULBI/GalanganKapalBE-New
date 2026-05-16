package activitylog

import "context"

// EventInput is the simplified payload used by other modules. Type defaults to 'info'.
type EventInput struct {
	Action       string
	Detail       string
	Type         string
	UserID       string
	ResourceType string
	ResourceID   string
	Category     string
}

// LogSafe writes an entry asynchronously, swallowing errors. Suitable for
// embedding into mutation paths without affecting the main business flow.
//
// This is a thin wrapper that converts the public EventInput (used by other
// modules) into the internal CreateInput. Keeping a separate type prevents
// modules from needing to import internal/activitylog packages.
func (s *Service) LogEvent(ctx context.Context, in EventInput) {
	s.LogSafe(ctx, CreateInput{
		Action:       in.Action,
		Detail:       in.Detail,
		Type:         in.Type,
		UserID:       in.UserID,
		ResourceType: in.ResourceType,
		ResourceID:   in.ResourceID,
		Category:     in.Category,
	})
}
