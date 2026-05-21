package purchaseorders

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

// Register attaches purchase order endpoints under r.
//   - protectedMW: any authenticated user (read).
//   - mutateMW: roles allowed to mutate (admin, supervisor).
func (h *Handler) Register(r fiber.Router, protectedMW fiber.Handler, mutateMW fiber.Handler) {
	g := r.Group("/purchase-orders", protectedMW)
	g.Get("/", h.list)
	g.Get("/stats", h.stats)
	g.Get("/:id", h.get)
	g.Post("/", mutateMW, h.create)
	g.Put("/:id", mutateMW, h.update)
	g.Delete("/:id", mutateMW, h.delete)
}

func (h *Handler) list(c *fiber.Ctx) error {
	f := ListFilters{
		Search:   strings.TrimSpace(c.Query("search")),
		Status:   strings.TrimSpace(c.Query("status")),
		VendorID: strings.TrimSpace(c.Query("vendorId")),
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

func (h *Handler) stats(c *fiber.Ctx) error {
	st, err := h.svc.Stats(c.UserContext())
	if err != nil {
		return err
	}
	return c.JSON(st)
}

func (h *Handler) get(c *fiber.Ctx) error {
	po, err := h.svc.Get(c.UserContext(), c.Params("id"))
	if err != nil {
		return mapError(err)
	}
	return c.JSON(po)
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
	po, err := h.svc.CreateAndLog(c.UserContext(), req, createdByID)
	if err != nil {
		return mapError(err)
	}
	return c.Status(fiber.StatusCreated).JSON(po)
}

func (h *Handler) update(c *fiber.Ctx) error {
	var req UpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	actorID, _ := c.Locals(auth.ContextUserIDKey).(string)
	po, err := h.svc.Update(c.UserContext(), c.Params("id"), req, actorID)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(po)
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
	case errors.Is(err, ErrPONumberExists):
		return fiber.NewError(fiber.StatusConflict, err.Error())
	case errors.Is(err, ErrVendorNotFound):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	case errors.Is(err, ErrMaterialNotFound):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	case IsValidation(err):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	default:
		return err
	}
}
