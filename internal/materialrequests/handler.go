package materialrequests

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

// Register attaches material request endpoints under r.
//   - protectedMW: any authenticated user (read own + create).
//   - approveMW: roles allowed to approve/reject (admin/supervisor).
func (h *Handler) Register(r fiber.Router, protectedMW fiber.Handler, approveMW fiber.Handler) {
	g := r.Group("/material-requests", protectedMW)
	g.Get("/", h.list)
	g.Get("/:id", h.get)
	g.Post("/", h.create)
	g.Post("/:id/approve", approveMW, h.approve)
	g.Post("/:id/reject", approveMW, h.reject)
	g.Delete("/:id", h.delete)
}

func (h *Handler) list(c *fiber.Ctx) error {
	role, _ := c.Locals(auth.ContextUserRoleKey).(string)
	userID, _ := c.Locals(auth.ContextUserIDKey).(string)

	limit := 0
	if v := strings.TrimSpace(c.Query("limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}

	f := ListFilters{
		Status:    strings.TrimSpace(c.Query("status")),
		Type:      strings.TrimSpace(c.Query("type")),
		Priority:  strings.TrimSpace(c.Query("priority")),
		ProjectID: strings.TrimSpace(c.Query("projectId")),
		Limit:     limit,
	}

	// Staff can only see their own requests.
	// Admin and Supervisor can see all (or pass ?requesterId=... to filter).
	switch role {
	case "staff":
		f.RequesterID = userID
	default:
		if v := strings.TrimSpace(c.Query("requesterId")); v != "" {
			f.RequesterID = v
		}
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
	mr, err := h.svc.Get(c.UserContext(), c.Params("id"))
	if err != nil {
		return mapError(err)
	}
	// Staff can only view their own.
	role, _ := c.Locals(auth.ContextUserRoleKey).(string)
	userID, _ := c.Locals(auth.ContextUserIDKey).(string)
	if role == "staff" && mr.RequesterID != userID {
		return fiber.NewError(fiber.StatusForbidden, "cannot view other user's request")
	}
	return c.JSON(mr)
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
	created, err := h.svc.Create(c.UserContext(), req, userID)
	if err != nil {
		return mapError(err)
	}
	return c.Status(fiber.StatusCreated).JSON(created)
}

func (h *Handler) approve(c *fiber.Ctx) error {
	approverID, _ := c.Locals(auth.ContextUserIDKey).(string)
	var req ApprovalRequest
	if err := c.BodyParser(&req); err != nil {
		// Body is optional; empty body is fine.
		req = ApprovalRequest{}
	}
	mr, err := h.svc.Approve(c.UserContext(), c.Params("id"), approverID, req.Notes)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(mr)
}

func (h *Handler) reject(c *fiber.Ctx) error {
	approverID, _ := c.Locals(auth.ContextUserIDKey).(string)
	var req ApprovalRequest
	if err := c.BodyParser(&req); err != nil {
		req = ApprovalRequest{}
	}
	mr, err := h.svc.Reject(c.UserContext(), c.Params("id"), approverID, req.Notes)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(mr)
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
	case errors.Is(err, ErrAlreadyDecided):
		return fiber.NewError(fiber.StatusConflict, err.Error())
	case errors.Is(err, ErrNotPending):
		return fiber.NewError(fiber.StatusConflict, err.Error())
	case errors.Is(err, ErrMaterialNotFound):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	case IsValidation(err):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	default:
		return err
	}
}
