package users

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/PROYEK3-ULBI/sims-backend/internal/auth"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Register attaches user-management endpoints (admin-only) under /api/users.
func (h *Handler) Register(r fiber.Router, protectedMW fiber.Handler, adminMW fiber.Handler) {
	g := r.Group("/users", protectedMW, adminMW)
	g.Get("/", h.list)
	g.Get("/:id", h.get)
	g.Post("/", h.create)
	g.Put("/:id", h.update)
	g.Patch("/:id/status", h.toggleStatus)
	g.Post("/:id/reset-password", h.resetPassword)
	g.Delete("/:id", h.delete)
}

func (h *Handler) list(c *fiber.Ctx) error {
	f := ListFilters{
		Search:     strings.TrimSpace(c.Query("search")),
		Role:       strings.TrimSpace(c.Query("role")),
		Status:     strings.TrimSpace(c.Query("status")),
		Department: strings.TrimSpace(c.Query("department")),
	}
	items, err := h.svc.List(c.UserContext(), f)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"data": items, "total": len(items)})
}

func (h *Handler) get(c *fiber.Ctx) error {
	u, err := h.svc.Get(c.UserContext(), c.Params("id"))
	if err != nil {
		return mapError(err)
	}
	return c.JSON(u)
}

func (h *Handler) create(c *fiber.Ctx) error {
	var req CreateRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	created, err := h.svc.Create(c.UserContext(), req)
	if err != nil {
		return mapError(err)
	}
	return c.Status(fiber.StatusCreated).JSON(created)
}

func (h *Handler) update(c *fiber.Ctx) error {
	var req UpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	updated, err := h.svc.Update(c.UserContext(), c.Params("id"), req)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(updated)
}

// toggleStatus is a convenience endpoint matching the frontend toggle in the admin
// dashboard. Body is optional; when omitted, the status flips active <-> inactive.
func (h *Handler) toggleStatus(c *fiber.Ctx) error {
	var req struct {
		Status *string `json:"status"`
	}
	_ = c.BodyParser(&req) // body is optional
	updated, err := h.svc.ToggleStatus(c.UserContext(), c.Params("id"), req.Status)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(updated)
}

func (h *Handler) resetPassword(c *fiber.Ctx) error {
	var req struct {
		Password string `json:"password"`
	}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if err := h.svc.ResetPassword(c.UserContext(), c.Params("id"), req.Password); err != nil {
		return mapError(err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) delete(c *fiber.Ctx) error {
	// Prevent admin from deleting themselves.
	currentID, _ := c.Locals(auth.ContextUserIDKey).(string)
	if currentID == c.Params("id") {
		return fiber.NewError(fiber.StatusBadRequest, "cannot delete your own account")
	}
	if err := h.svc.Delete(c.UserContext(), c.Params("id")); err != nil {
		return mapError(err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func mapError(err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	case errors.Is(err, ErrEmailConflict):
		return fiber.NewError(fiber.StatusConflict, err.Error())
	case errors.Is(err, ErrInUse):
		return fiber.NewError(fiber.StatusConflict, err.Error())
	case IsValidation(err):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	default:
		return err
	}
}
