package transactions

import (
	"errors"
	"strconv"
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

// Register attaches transaction endpoints under r.
//   - protectedMW: any authenticated user (read + create their own transactions).
func (h *Handler) Register(r fiber.Router, protectedMW fiber.Handler) {
	g := r.Group("", protectedMW)

	// Read endpoints
	g.Get("/transactions", h.list)
	g.Get("/transactions/:id", h.get)

	// Create endpoints (any authenticated user can perform; RBAC enforced via UI for now)
	g.Post("/goods-receipt", h.receipt)
	g.Post("/goods-issue", h.issue)
	g.Post("/scrap-return", h.scrapReturn)
}

func (h *Handler) list(c *fiber.Ctx) error {
	limit := 0
	if v := strings.TrimSpace(c.Query("limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	f := ListFilters{
		Type:       strings.TrimSpace(c.Query("type")),
		MaterialID: strings.TrimSpace(c.Query("materialId")),
		ProjectID:  strings.TrimSpace(c.Query("projectId")),
		VendorID:   strings.TrimSpace(c.Query("vendorId")),
		UserID:     strings.TrimSpace(c.Query("userId")),
		StartDate:  strings.TrimSpace(c.Query("startDate")),
		EndDate:    strings.TrimSpace(c.Query("endDate")),
		Limit:      limit,
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

func (h *Handler) get(c *fiber.Ctx) error {
	t, err := h.svc.Get(c.UserContext(), c.Params("id"))
	if err != nil {
		return mapError(err)
	}
	return c.JSON(t)
}

func (h *Handler) receipt(c *fiber.Ctx) error {
	var req ReceiptRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	userID := userIDFromCtx(c)
	out, err := h.svc.Receipt(c.UserContext(), req, userID)
	if err != nil {
		return mapError(err)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"data":  out,
		"total": len(out),
	})
}

func (h *Handler) issue(c *fiber.Ctx) error {
	var req IssueRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	userID := userIDFromCtx(c)
	out, err := h.svc.Issue(c.UserContext(), req, userID)
	if err != nil {
		return mapError(err)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"data":  out,
		"total": len(out),
	})
}

func (h *Handler) scrapReturn(c *fiber.Ctx) error {
	var req ScrapReturnRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	userID := userIDFromCtx(c)
	t, err := h.svc.ScrapReturn(c.UserContext(), req, userID)
	if err != nil {
		return mapError(err)
	}
	return c.Status(fiber.StatusCreated).JSON(t)
}

func userIDFromCtx(c *fiber.Ctx) *string {
	if uid, ok := c.Locals(auth.ContextUserIDKey).(string); ok && uid != "" {
		return &uid
	}
	return nil
}

func mapError(err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	case errors.Is(err, ErrInsufficientStock):
		return fiber.NewError(fiber.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, ErrReceiveExceedsOrdered):
		return fiber.NewError(fiber.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, ErrMaterialNotFound),
		errors.Is(err, ErrPOItemNotFound),
		errors.Is(err, ErrPOItemMaterialMismatch):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	case IsValidation(err):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	default:
		return err
	}
}
