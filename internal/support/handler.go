package support

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

// Register attaches support ticket endpoints under /api/support/tickets.
//   - protectedMW: any authenticated user can submit + view own tickets.
//   - adminMW: only admin can list all + change status.
func (h *Handler) Register(r fiber.Router, protectedMW fiber.Handler, adminMW fiber.Handler) {
	g := r.Group("/support/tickets", protectedMW)
	g.Get("/", h.list)
	g.Get("/:id", h.get)
	g.Post("/", h.create)
	g.Patch("/:id/status", adminMW, h.updateStatus)
	g.Post("/:id/resolve", adminMW, h.resolve)
	g.Delete("/:id", h.delete)
}

func (h *Handler) list(c *fiber.Ctx) error {
	role, _ := c.Locals(auth.ContextUserRoleKey).(string)
	userID, _ := c.Locals(auth.ContextUserIDKey).(string)

	f := ListFilters{
		Status:   strings.TrimSpace(c.Query("status")),
		Priority: strings.TrimSpace(c.Query("priority")),
	}
	// Non-admin users can only see their own tickets.
	if role != "admin" {
		f.UserID = userID
	} else if v := strings.TrimSpace(c.Query("userId")); v != "" {
		f.UserID = v
	}
	if f.Status != "" && !IsValidStatus(f.Status) {
		return fiber.NewError(fiber.StatusBadRequest, "invalid status filter")
	}

	items, err := h.svc.List(c.UserContext(), f)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"data": items, "total": len(items)})
}

func (h *Handler) get(c *fiber.Ctx) error {
	t, err := h.svc.Get(c.UserContext(), c.Params("id"))
	if err != nil {
		return mapError(err)
	}
	role, _ := c.Locals(auth.ContextUserRoleKey).(string)
	userID, _ := c.Locals(auth.ContextUserIDKey).(string)
	if role != "admin" && t.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "cannot view another user's ticket")
	}
	return c.JSON(t)
}

func (h *Handler) create(c *fiber.Ctx) error {
	userID, _ := c.Locals(auth.ContextUserIDKey).(string)
	if userID == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "not authenticated")
	}
	var req CreateRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	created, err := h.svc.Submit(c.UserContext(), userID, req)
	if err != nil {
		return mapError(err)
	}
	return c.Status(fiber.StatusCreated).JSON(created)
}

func (h *Handler) updateStatus(c *fiber.Ctx) error {
	handlerID, _ := c.Locals(auth.ContextUserIDKey).(string)
	var req UpdateStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	t, err := h.svc.UpdateStatus(c.UserContext(), c.Params("id"), handlerID, req.Status, req.Response)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(t)
}

func (h *Handler) resolve(c *fiber.Ctx) error {
	handlerID, _ := c.Locals(auth.ContextUserIDKey).(string)
	var req ResolveRequest
	_ = c.BodyParser(&req)
	t, err := h.svc.Resolve(c.UserContext(), c.Params("id"), handlerID, req.Response)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(t)
}

func (h *Handler) delete(c *fiber.Ctx) error {
	userID, _ := c.Locals(auth.ContextUserIDKey).(string)
	if err := h.svc.Delete(c.UserContext(), c.Params("id"), userID); err != nil {
		return mapError(err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func mapError(err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	case errors.Is(err, ErrCannotDelete):
		return fiber.NewError(fiber.StatusConflict, err.Error())
	case IsValidation(err):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	default:
		return err
	}
}
