package warehouselocations

import "time"

// Location represents a warehouse storage area (yard, indoor warehouse, etc.).
type Location struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name,omitempty"`
	Type      string    `json:"type,omitempty"`
	Capacity  *int      `json:"capacity,omitempty"`
	Notes     string    `json:"notes,omitempty"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type CreateRequest struct {
	Code     string  `json:"code"`
	Name     *string `json:"name"`
	Type     *string `json:"type"`
	Capacity *int    `json:"capacity"`
	Notes    *string `json:"notes"`
	Status   *string `json:"status"`
}

type UpdateRequest struct {
	Name     *string `json:"name"`
	Type     *string `json:"type"`
	Capacity *int    `json:"capacity"`
	Notes    *string `json:"notes"`
	Status   *string `json:"status"`
}

type ListFilters struct {
	Search     string
	Type       string
	Status     string
	ActiveOnly bool
}

func IsValidStatus(s string) bool {
	switch s {
	case "active", "inactive":
		return true
	}
	return false
}
