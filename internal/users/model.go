package users

import "time"

// User represents a row in the users table for the management API.
// Note: this is a separate struct from auth.User to keep the management view
// (which excludes password_hash even from internal Go types where possible).
type User struct {
	ID          string     `json:"id"`
	Email       string     `json:"email"`
	Name        string     `json:"name"`
	Role        string     `json:"role"`
	Avatar      string     `json:"avatar,omitempty"`
	Department  string     `json:"department,omitempty"`
	Status      string     `json:"status"`
	LastLoginAt *time.Time `json:"lastLoginAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

// CreateRequest is the payload for POST /api/users.
type CreateRequest struct {
	Email      string  `json:"email"`
	Password   string  `json:"password"`
	Name       string  `json:"name"`
	Role       string  `json:"role"`
	Avatar     *string `json:"avatar"`
	Department *string `json:"department"`
	Status     *string `json:"status"`
}

// UpdateRequest applies partial updates. Email cannot be changed via update for
// audit cleanliness; create a new user if needed.
type UpdateRequest struct {
	Name       *string `json:"name"`
	Role       *string `json:"role"`
	Avatar     *string `json:"avatar"`
	Department *string `json:"department"`
	Status     *string `json:"status"`
}

// ListFilters controls GET /api/users.
type ListFilters struct {
	Search     string
	Role       string
	Status     string
	Department string
}

func IsValidRole(s string) bool {
	switch s {
	case "admin", "supervisor", "staff":
		return true
	}
	return false
}

func IsValidStatus(s string) bool {
	switch s {
	case "active", "inactive":
		return true
	}
	return false
}
