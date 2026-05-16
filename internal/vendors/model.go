package vendors

import "time"

// Vendor represents a supplier in the system.
type Vendor struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Contact   *string   `json:"contact,omitempty"`
	Phone     *string   `json:"phone,omitempty"`
	Email     *string   `json:"email,omitempty"`
	Address   *string   `json:"address,omitempty"`
	Status    string    `json:"status"`
	POCount   int       `json:"poCount"` // populated on list queries
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type CreateRequest struct {
	Name    string  `json:"name"`
	Contact *string `json:"contact"`
	Phone   *string `json:"phone"`
	Email   *string `json:"email"`
	Address *string `json:"address"`
	Status  *string `json:"status"`
}

type UpdateRequest struct {
	Name    *string `json:"name"`
	Contact *string `json:"contact"`
	Phone   *string `json:"phone"`
	Email   *string `json:"email"`
	Address *string `json:"address"`
	Status  *string `json:"status"`
}

type ListFilters struct {
	Search       string // matches name (case-insensitive)
	Status       string // "active" or "inactive" or "" for all
	ActiveOnly   bool
}
