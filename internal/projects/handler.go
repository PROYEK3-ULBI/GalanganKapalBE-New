package projects

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

// Register attaches project endpoints under r.
//   - protectedMW: any authenticated user (read).
//   - mutateMW: only roles allowed to mutate (admin/supervisor).
func (h *Handler) Register(r fiber.Router, protectedMW fiber.Handler, mutateMW fiber.Handler) {
	g := r.Group("/projects", protectedMW)
	g.Get("/", h.list)
	g.Get("/:id", h.get)
	g.Post("/", mutateMW, h.create)
	g.Put("/:id", mutateMW, h.update)
	g.Delete("/:id", mutateMW, h.delete)
}

func (h *Handler) list(c *fiber.Ctx) error {
	f := ListFilters{
		Search:     strings.TrimSpace(c.Query("search")),
		Status:     strings.TrimSpace(c.Query("status")),
		Type:       strings.TrimSpace(c.Query("type")),
		ActiveOnly: c.Query("activeOnly") == "true",
	}
	items, err := h.svc.List(c.UserContext(), f)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{
		"data":  items,
		"total": len(items),
	})
}

func (h *Handler) get(c *fiber.Ctx) error {
	p, err := h.svc.Get(c.UserContext(), c.Params("id"))
	if err != nil {
		return mapError(err)
	}
	return c.JSON(p)
}

func (h *Handler) create(c *fiber.Ctx) error {
	var req CreateRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	var createdByID *string
	if uid, ok := c.Locals(auth.ContextUserIDKey).(string); ok && uid != "" {
		createdByID = &uid
	}
	created, err := h.svc.Create(c.UserContext(), req, createdByID)
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
	actorID, _ := c.Locals(auth.ContextUserIDKey).(string)
	updated, err := h.svc.Update(c.UserContext(), c.Params("id"), req, actorID)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(updated)
}

func (h *Handler) delete(c *fiber.Ctx) error {
	actorID, _ := c.Locals(auth.ContextUserIDKey).(string)
	if err := h.svc.Delete(c.UserContext(), c.Params("id"), actorID); err != nil {
		return mapError(err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func mapError(err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	case errors.Is(err, ErrCodeConflict):
		return fiber.NewError(fiber.StatusConflict, err.Error())
	case errors.Is(err, ErrInUse):
		return fiber.NewError(fiber.StatusConflict, err.Error())
	case IsValidation(err):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	default:
		return err
	}
}
