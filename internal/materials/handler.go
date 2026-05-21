package materials

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/PROYEK3-ULBI/sims-backend/internal/auth"
)

// Handler wires materials endpoints into the Fiber router.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Register attaches routes under r.
//   - protectedMW: authenticates any logged-in user (required for read).
//   - adminOnlyMW: restricts mutating endpoints to admin role.
func (h *Handler) Register(r fiber.Router, protectedMW fiber.Handler, adminOnlyMW fiber.Handler) {
	g := r.Group("/materials", protectedMW)
	g.Get("/", h.list)
	g.Get("/categories", h.categories)
	g.Get("/:id", h.get)
	g.Post("/", adminOnlyMW, h.create)
	g.Put("/:id", adminOnlyMW, h.update)
	g.Delete("/:id", adminOnlyMW, h.delete)
}

func (h *Handler) list(c *fiber.Ctx) error {
	f := ListFilters{
		Search:       strings.TrimSpace(c.Query("search")),
		Category:     strings.TrimSpace(c.Query("category")),
		HazmatOnly:   c.Query("hazmat") == "true",
		LowStockOnly: c.Query("lowStock") == "true",
	}
	if strings.EqualFold(f.Category, "All Categories") {
		f.Category = ""
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

func (h *Handler) categories(c *fiber.Ctx) error {
	cats, err := h.svc.Categories(c.UserContext())
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"data": cats})
}

func (h *Handler) get(c *fiber.Ctx) error {
	id := c.Params("id")
	m, err := h.svc.Get(c.UserContext(), id)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(m)
}

func (h *Handler) create(c *fiber.Ctx) error {
	var req CreateRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	actorID, _ := c.Locals(auth.ContextUserIDKey).(string)
	created, err := h.svc.Create(c.UserContext(), req, actorID)
	if err != nil {
		return mapError(err)
	}
	return c.Status(fiber.StatusCreated).JSON(created)
}

func (h *Handler) update(c *fiber.Ctx) error {
	id := c.Params("id")
	var req UpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	actorID, _ := c.Locals(auth.ContextUserIDKey).(string)
	updated, err := h.svc.Update(c.UserContext(), id, req, actorID)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(updated)
}

func (h *Handler) delete(c *fiber.Ctx) error {
	id := c.Params("id")
	actorID, _ := c.Locals(auth.ContextUserIDKey).(string)
	if err := h.svc.Delete(c.UserContext(), id, actorID); err != nil {
		return mapError(err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// mapError translates domain errors into Fiber HTTP errors.
func mapError(err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	case errors.Is(err, ErrSKUConflict):
		return fiber.NewError(fiber.StatusConflict, err.Error())
	case IsValidation(err):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	default:
		return err
	}
}
