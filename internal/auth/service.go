package auth

import (
	"context"
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// Public service errors.
var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInactiveAccount    = errors.New("account is inactive")
)

// Service contains the authentication business logic.
type Service struct {
	repo     *Repository
	jwt      *JWTManager
	activity ActivityLogger // optional, may be nil
}

// ActivityLogger is the minimal interface this module needs from activitylog.
// Defined here to avoid a hard dependency.
type ActivityLogger interface {
	LogLogin(ctx context.Context, userID, userName, email string)
}

func NewService(repo *Repository, jwt *JWTManager) *Service {
	return &Service{repo: repo, jwt: jwt}
}

// SetActivityLogger wires an optional activity logger that records login events.
func (s *Service) SetActivityLogger(l ActivityLogger) {
	s.activity = l
}

// Login validates credentials and returns a signed JWT plus user details.
func (s *Service) Login(ctx context.Context, email, password string) (*LoginResponse, error) {
	email = strings.TrimSpace(email)
	if email == "" || password == "" {
		return nil, ErrInvalidCredentials
	}

	user, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			// Same error to avoid leaking which emails exist.
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if user.Status != "active" {
		return nil, ErrInactiveAccount
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, err := s.jwt.Generate(user)
	if err != nil {
		return nil, err
	}

	// Best-effort timestamp update; ignore errors.
	_ = s.repo.TouchLastLogin(ctx, user.ID)

	// Best-effort activity log entry.
	if s.activity != nil {
		s.activity.LogLogin(ctx, user.ID, user.Name, user.Email)
	}

	return &LoginResponse{Token: token, User: *user}, nil
}

// Me retrieves the user record for the currently authenticated request.
func (s *Service) Me(ctx context.Context, userID string) (*User, error) {
	return s.repo.FindByID(ctx, userID)
}

// UpdateProfile lets the authenticated user edit their own profile fields.
func (s *Service) UpdateProfile(ctx context.Context, userID string, req UpdateProfileRequest) (*User, error) {
	if req.Name != nil && strings.TrimSpace(*req.Name) == "" {
		return nil, errors.New("name cannot be empty")
	}
	return s.repo.UpdateProfile(ctx, userID, req)
}

// ChangePassword verifies the current password and updates to the new one.
func (s *Service) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	if len(newPassword) < 6 {
		return errors.New("new password must be at least 6 characters")
	}
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
		return ErrInvalidCredentials
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.repo.UpdatePassword(ctx, userID, string(hash))
}

// UpdateNotificationPreferences replaces the user's preferences map.
func (s *Service) UpdateNotificationPreferences(ctx context.Context, userID string, prefs map[string]any) (*User, error) {
	if prefs == nil {
		prefs = map[string]any{}
	}
	return s.repo.UpdateNotificationPreferences(ctx, userID, prefs)
}
