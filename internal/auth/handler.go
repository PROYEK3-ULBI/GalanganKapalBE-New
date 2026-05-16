package auth

import (
	"errors"

	"github.com/gofiber/fiber/v2"
)

// Handler wires authentication endpoints to the Fiber router.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Register attaches auth routes under the given router group.
// The protectedMW must be the JWT middleware to apply to /me.
func (h *Handler) Register(r fiber.Router, protectedMW fiber.Handler) {
	r.Post("/login", h.login)
	r.Get("/me", protectedMW, h.me)
	r.Put("/profile", protectedMW, h.updateProfile)
	r.Post("/password", protectedMW, h.changePassword)
	r.Put("/notification-preferences", protectedMW, h.updateNotifPrefs)
}

func (h *Handler) login(c *fiber.Ctx) error {
	var body LoginRequest
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	resp, err := h.svc.Login(c.UserContext(), body.Email, body.Password)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			return fiber.NewError(fiber.StatusUnauthorized, err.Error())
		case errors.Is(err, ErrInactiveAccount):
			return fiber.NewError(fiber.StatusForbidden, err.Error())
		default:
			return err
		}
	}
	return c.JSON(resp)
}

func (h *Handler) me(c *fiber.Ctx) error {
	userID, ok := c.Locals(ContextUserIDKey).(string)
	if !ok || userID == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "not authenticated")
	}

	user, err := h.svc.Me(c.UserContext(), userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "user not found")
		}
		return err
	}
	return c.JSON(fiber.Map{"user": user})
}

func (h *Handler) updateProfile(c *fiber.Ctx) error {
	userID, _ := c.Locals(ContextUserIDKey).(string)
	if userID == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "not authenticated")
	}
	var req UpdateProfileRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	user, err := h.svc.UpdateProfile(c.UserContext(), userID, req)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "user not found")
		}
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.JSON(fiber.Map{"user": user})
}

func (h *Handler) changePassword(c *fiber.Ctx) error {
	userID, _ := c.Locals(ContextUserIDKey).(string)
	if userID == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "not authenticated")
	}
	var req ChangePasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if err := h.svc.ChangePassword(c.UserContext(), userID, req.CurrentPassword, req.NewPassword); err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			return fiber.NewError(fiber.StatusUnauthorized, "current password is incorrect")
		case errors.Is(err, ErrUserNotFound):
			return fiber.NewError(fiber.StatusNotFound, "user not found")
		default:
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) updateNotifPrefs(c *fiber.Ctx) error {
	userID, _ := c.Locals(ContextUserIDKey).(string)
	if userID == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "not authenticated")
	}
	var req UpdateNotificationPreferencesRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	user, err := h.svc.UpdateNotificationPreferences(c.UserContext(), userID, req.Preferences)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "user not found")
		}
		return err
	}
	return c.JSON(fiber.Map{"user": user})
}
