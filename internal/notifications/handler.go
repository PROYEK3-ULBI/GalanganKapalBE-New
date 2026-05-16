package notifications

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

// Register attaches user-scoped notification endpoints. Every endpoint is
// implicitly scoped to the authenticated user — there is no admin endpoint
// to read other users' notifications.
func (h *Handler) Register(r fiber.Router, protectedMW fiber.Handler) {
	g := r.Group("/notifications", protectedMW)
	g.Get("/", h.list)
	g.Get("/stats", h.stats)
	g.Patch("/read-all", h.markAllRead)
	g.Patch("/:id/read", h.markRead)
	g.Delete("/:id", h.delete)
}

func userID(c *fiber.Ctx) string {
	id, _ := c.Locals(auth.ContextUserIDKey).(string)
	return id
}

func (h *Handler) list(c *fiber.Ctx) error {
	uid := userID(c)
	if uid == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "not authenticated")
	}

	f := ListFilters{UserID: uid}
	switch strings.ToLower(strings.TrimSpace(c.Query("read"))) {
	case "true":
		t := true
		f.Read = &t
	case "false":
		t := false
		f.Read = &t
	}
	if v := strings.TrimSpace(c.Query("limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			f.Limit = n
		}
	}
	f.Category = strings.TrimSpace(c.Query("category"))

	items, err := h.svc.List(c.UserContext(), f)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"data": items, "total": len(items)})
}

func (h *Handler) stats(c *fiber.Ctx) error {
	uid := userID(c)
	if uid == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "not authenticated")
	}
	s, err := h.svc.Stats(c.UserContext(), uid)
	if err != nil {
		return err
	}
	return c.JSON(s)
}

func (h *Handler) markRead(c *fiber.Ctx) error {
	uid := userID(c)
	if err := h.svc.MarkRead(c.UserContext(), c.Params("id"), uid); err != nil {
		if errors.Is(err, ErrNotFound) {
			return fiber.NewError(fiber.StatusNotFound, err.Error())
		}
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) markAllRead(c *fiber.Ctx) error {
	uid := userID(c)
	count, err := h.svc.MarkAllRead(c.UserContext(), uid)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"updated": count})
}

func (h *Handler) delete(c *fiber.Ctx) error {
	uid := userID(c)
	if err := h.svc.Delete(c.UserContext(), c.Params("id"), uid); err != nil {
		if errors.Is(err, ErrNotFound) {
			return fiber.NewError(fiber.StatusNotFound, err.Error())
		}
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}
