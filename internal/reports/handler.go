package reports

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

// Register attaches read-only report endpoints. All require an authenticated user.
func (h *Handler) Register(r fiber.Router, protectedMW fiber.Handler) {
	g := r.Group("/reports", protectedMW)
	g.Get("/summary", h.summary)
	g.Get("/stock-valuation", h.stockValuation)
	g.Get("/category-breakdown", h.categoryBreakdown)
	g.Get("/transaction-summary", h.transactionSummary)
	g.Get("/project-consumption", h.projectConsumption)
	g.Get("/inventory-trend", h.inventoryTrend)
}

func (h *Handler) summary(c *fiber.Ctx) error {
	s, err := h.svc.Summary(c.UserContext())
	if err != nil {
		return err
	}
	return c.JSON(s)
}

func (h *Handler) stockValuation(c *fiber.Ctx) error {
	items, err := h.svc.StockValuation(c.UserContext())
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"data": items, "total": len(items)})
}

func (h *Handler) categoryBreakdown(c *fiber.Ctx) error {
	items, err := h.svc.CategoryBreakdown(c.UserContext())
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"data": items, "total": len(items)})
}

func (h *Handler) transactionSummary(c *fiber.Ctx) error {
	start := strings.TrimSpace(c.Query("startDate"))
	end := strings.TrimSpace(c.Query("endDate"))
	items, err := h.svc.TransactionSummary(c.UserContext(), start, end)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"data": items, "total": len(items)})
}

func (h *Handler) projectConsumption(c *fiber.Ctx) error {
	items, err := h.svc.ProjectConsumption(c.UserContext())
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"data": items, "total": len(items)})
}

func (h *Handler) inventoryTrend(c *fiber.Ctx) error {
	days := 30
	if v := strings.TrimSpace(c.Query("days")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			days = n
		}
	}
	items, err := h.svc.InventoryTrend(c.UserContext(), days)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"data": items, "total": len(items)})
}
