package tools

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

// Register attaches tool endpoints under /api/tools.
//   - protectedMW: any authenticated user (read + checkout/return — staff can borrow tools).
//   - mutateMW: only roles allowed to mutate the catalog (admin).
func (h *Handler) Register(r fiber.Router, protectedMW fiber.Handler, mutateMW fiber.Handler) {
	g := r.Group("/tools", protectedMW)
	g.Get("/", h.list)
	g.Get("/:id", h.get)
	g.Get("/:id/history", h.history)
	g.Post("/", mutateMW, h.create)
	g.Put("/:id", mutateMW, h.update)
	g.Delete("/:id", mutateMW, h.delete)

	// Workflow endpoints — any authenticated user can checkout/return.
	g.Post("/:id/checkout", h.checkout)
	g.Post("/:id/return", h.returnTool)
	g.Post("/:id/maintenance", mutateMW, h.maintenance)
	g.Post("/:id/available", mutateMW, h.makeAvailable)
}

func (h *Handler) list(c *fiber.Ctx) error {
	f := ListFilters{
		Search:             strings.TrimSpace(c.Query("search")),
		Status:             strings.TrimSpace(c.Query("status")),
		Category:           strings.TrimSpace(c.Query("category")),
		BorrowerID:         strings.TrimSpace(c.Query("borrowerId")),
		CalibrationDueOnly: c.Query("calibrationDue") == "true",
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
	return c.JSON(t)
}

func (h *Handler) history(c *fiber.Ctx) error {
	hist, err := h.svc.History(c.UserContext(), c.Params("id"))
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"data": hist, "total": len(hist)})
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

func (h *Handler) checkout(c *fiber.Ctx) error {
	userID, _ := c.Locals(auth.ContextUserIDKey).(string)
	var req CheckoutRequest
	_ = c.BodyParser(&req)
	t, err := h.svc.Checkout(c.UserContext(), c.Params("id"), userID, req)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(t)
}

func (h *Handler) returnTool(c *fiber.Ctx) error {
	userID, _ := c.Locals(auth.ContextUserIDKey).(string)
	var req ReturnRequest
	_ = c.BodyParser(&req)
	var actingID *string
	if userID != "" {
		actingID = &userID
	}
	t, err := h.svc.Return(c.UserContext(), c.Params("id"), actingID, req)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(t)
}

func (h *Handler) maintenance(c *fiber.Ctx) error {
	userID, _ := c.Locals(auth.ContextUserIDKey).(string)
	var req MaintenanceRequest
	_ = c.BodyParser(&req)
	var actingID *string
	if userID != "" {
		actingID = &userID
	}
	t, err := h.svc.Maintenance(c.UserContext(), c.Params("id"), actingID, req)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(t)
}

func (h *Handler) makeAvailable(c *fiber.Ctx) error {
	userID, _ := c.Locals(auth.ContextUserIDKey).(string)
	var body struct {
		Notes *string `json:"notes"`
	}
	_ = c.BodyParser(&body)
	var actingID *string
	if userID != "" {
		actingID = &userID
	}
	t, err := h.svc.MakeAvailable(c.UserContext(), c.Params("id"), actingID, body.Notes)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(t)
}

func mapError(err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	case errors.Is(err, ErrSKUConflict):
		return fiber.NewError(fiber.StatusConflict, err.Error())
	case errors.Is(err, ErrAlreadyCheckedOut),
		errors.Is(err, ErrNotCheckedOut),
		errors.Is(err, ErrInMaintenance):
		return fiber.NewError(fiber.StatusConflict, err.Error())
	case IsValidation(err):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	default:
		return err
	}
}
