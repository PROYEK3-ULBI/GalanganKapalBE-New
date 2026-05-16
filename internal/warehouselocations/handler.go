package warehouselocations

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Register attaches warehouse-location endpoints under /api/warehouse-locations.
//   - protectedMW: any authenticated user can read.
//   - adminMW: only admin can mutate the catalog.
func (h *Handler) Register(r fiber.Router, protectedMW fiber.Handler, adminMW fiber.Handler) {
	g := r.Group("/warehouse-locations", protectedMW)
	g.Get("/", h.list)
	g.Get("/:id", h.get)
	g.Post("/", adminMW, h.create)
	g.Put("/:id", adminMW, h.update)
	g.Delete("/:id", adminMW, h.delete)
}

func (h *Handler) list(c *fiber.Ctx) error {
	f := ListFilters{
		Search:     strings.TrimSpace(c.Query("search")),
		Type:       strings.TrimSpace(c.Query("type")),
		Status:     strings.TrimSpace(c.Query("status")),
		ActiveOnly: c.Query("activeOnly") == "true",
	}
	items, err := h.svc.List(c.UserContext(), f)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"data": items, "total": len(items)})
}

func (h *Handler) get(c *fiber.Ctx) error {
	loc, err := h.svc.Get(c.UserContext(), c.Params("id"))
	if err != nil {
		return mapError(err)
	}
	return c.JSON(loc)
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

func (h *Handler) delete(c *fiber.Ctx) error {
	if err := h.svc.Delete(c.UserContext(), c.Params("id")); err != nil {
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
	case IsValidation(err):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	default:
		return err
	}
}
