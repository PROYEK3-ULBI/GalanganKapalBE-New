package activitylog

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Register attaches activity log endpoints under /api/activity-logs.
//   - protectedMW: any authenticated user (read).
//   - adminMW: only admin can read full activity log (system-wide audit).
func (h *Handler) Register(r fiber.Router, protectedMW fiber.Handler, adminMW fiber.Handler) {
	g := r.Group("/activity-logs", protectedMW, adminMW)
	g.Get("/", h.list)
}

func (h *Handler) list(c *fiber.Ctx) error {
	f := ListFilters{
		UserID:    strings.TrimSpace(c.Query("userId")),
		Type:      strings.TrimSpace(c.Query("type")),
		Category:  strings.TrimSpace(c.Query("category")),
		StartDate: strings.TrimSpace(c.Query("startDate")),
		EndDate:   strings.TrimSpace(c.Query("endDate")),
	}
	if v := strings.TrimSpace(c.Query("limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			f.Limit = n
		}
	} else {
		// Default to last 50 entries to avoid pulling the entire history.
		f.Limit = 50
	}
	if f.Type != "" && !IsValidType(f.Type) {
		return fiber.NewError(fiber.StatusBadRequest, "invalid type filter")
	}

	items, err := h.svc.List(c.UserContext(), f)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"data": items, "total": len(items)})
}
