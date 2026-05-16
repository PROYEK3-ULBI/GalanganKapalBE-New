package auth

import "time"

// User represents a user account in the system.
// Mirrors the users table schema.
type User struct {
	ID                      string         `json:"id"`
	Email                   string         `json:"email"`
	PasswordHash            string         `json:"-"` // never expose in JSON
	Name                    string         `json:"name"`
	Role                    string         `json:"role"`
	Avatar                  string         `json:"avatar,omitempty"`
	Department              string         `json:"department,omitempty"`
	Phone                   string         `json:"phone,omitempty"`
	Position                string         `json:"position,omitempty"`
	NotificationPreferences map[string]any `json:"notificationPreferences,omitempty"`
	Status                  string         `json:"status"`
	LastLoginAt             *time.Time     `json:"lastLoginAt,omitempty"`
	CreatedAt               time.Time      `json:"createdAt"`
	UpdatedAt               time.Time      `json:"updatedAt"`
}

// LoginRequest is the payload accepted by POST /api/auth/login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse is returned after successful authentication.
type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// UpdateProfileRequest is the payload for PUT /api/auth/profile.
type UpdateProfileRequest struct {
	Name       *string `json:"name"`
	Department *string `json:"department"`
	Phone      *string `json:"phone"`
	Position   *string `json:"position"`
	Avatar     *string `json:"avatar"`
}

// ChangePasswordRequest is the payload for POST /api/auth/password.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

// UpdateNotificationPreferencesRequest is the payload for PUT /api/auth/notification-preferences.
type UpdateNotificationPreferencesRequest struct {
	Preferences map[string]any `json:"preferences"`
}
