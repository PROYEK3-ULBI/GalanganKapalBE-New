package server

import (
	"errors"
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PROYEK3-ULBI/sims-backend/internal/activitylog"
	"github.com/PROYEK3-ULBI/sims-backend/internal/auth"
	"github.com/PROYEK3-ULBI/sims-backend/internal/config"
	"github.com/PROYEK3-ULBI/sims-backend/internal/materialrequests"
	"github.com/PROYEK3-ULBI/sims-backend/internal/materials"
	"github.com/PROYEK3-ULBI/sims-backend/internal/notifications"
	"github.com/PROYEK3-ULBI/sims-backend/internal/projects"
	"github.com/PROYEK3-ULBI/sims-backend/internal/purchaseorders"
	"github.com/PROYEK3-ULBI/sims-backend/internal/reports"
	"github.com/PROYEK3-ULBI/sims-backend/internal/support"
	"github.com/PROYEK3-ULBI/sims-backend/internal/tools"
	"github.com/PROYEK3-ULBI/sims-backend/internal/transactions"
	"github.com/PROYEK3-ULBI/sims-backend/internal/users"
	"github.com/PROYEK3-ULBI/sims-backend/internal/vendors"
	"github.com/PROYEK3-ULBI/sims-backend/internal/warehouselocations"
)

// New constructs a Fiber app with all middleware and routes wired in.
func New(cfg *config.Config, pool *pgxpool.Pool) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:               "SIMS Backend",
		DisableStartupMessage: true,
		ErrorHandler:          errorHandler,
	})

	// Global middleware.
	app.Use(recover.New())
	if !cfg.IsProduction() {
		app.Use(logger.New())
	}
	app.Use(cors.New(cors.Config{
		AllowOrigins:     strings.Join(cfg.CorsAllowedOrigins, ","),
		AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization",
		AllowCredentials: false,
	}))

	// Health check.
	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// Wire auth module.
	jwtMgr := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTExpiryHours)
	authRepo := auth.NewRepository(pool)
	authSvc := auth.NewService(authRepo, jwtMgr)
	authHandler := auth.NewHandler(authSvc)

	api := app.Group("/api")
	authMW := auth.Middleware(jwtMgr)
	adminMW := auth.RequireRole("admin")
	// Procurement workflows: admin and supervisor can mutate PO/vendor data.
	procurementMW := auth.RequireRole("admin", "supervisor")

	// Wire activity log module first so other modules can be wired with the adapter.
	actRepo := activitylog.NewRepository(pool)
	actSvc := activitylog.NewService(actRepo)
	actHandler := activitylog.NewHandler(actSvc)
	actHandler.Register(api, authMW, adminMW)

	// Now hook the auth login event into the activity log.
	authSvc.SetActivityLogger(activitylog.NewAuthAdapter(actSvc))

	authHandler.Register(api.Group("/auth"), authMW)

	// Wire materials module.
	matRepo := materials.NewRepository(pool)
	matSvc := materials.NewService(matRepo)
	matSvc.SetActivityLogger(activitylog.NewMaterialsAdapter(actSvc))
	matHandler := materials.NewHandler(matSvc)
	matHandler.Register(api, authMW, adminMW)

	// Wire vendors module.
	venRepo := vendors.NewRepository(pool)
	venSvc := vendors.NewService(venRepo)
	venSvc.SetActivityLogger(activitylog.NewVendorsAdapter(actSvc))
	venHandler := vendors.NewHandler(venSvc)
	venHandler.Register(api, authMW, procurementMW)

	// Wire purchase orders module.
	poRepo := purchaseorders.NewRepository(pool)
	poSvc := purchaseorders.NewService(poRepo)
	poSvc.SetActivityLogger(activitylog.NewPurchaseOrdersAdapter(actSvc))
	poHandler := purchaseorders.NewHandler(poSvc)
	poHandler.Register(api, authMW, procurementMW)

	// Wire projects module.
	projRepo := projects.NewRepository(pool)
	projSvc := projects.NewService(projRepo)
	projSvc.SetActivityLogger(activitylog.NewProjectsAdapter(actSvc))
	projHandler := projects.NewHandler(projSvc)
	projHandler.Register(api, authMW, procurementMW)

	// Wire transactions module (goods receipt / issue / scrap-return / list).
	txRepo := transactions.NewRepository(pool)
	txSvc := transactions.NewService(pool, txRepo)
	txSvc.SetActivityLogger(activitylog.NewTransactionsAdapter(actSvc))
	txHandler := transactions.NewHandler(txSvc)
	txHandler.Register(api, authMW)

	// Wire material requests module (Staff submits, Supervisor approves/rejects).
	mrRepo := materialrequests.NewRepository(pool)

	// Notifications module is wired before MR so the MR service can fire-and-forget
	// notifications via the adapter when requests are created/approved/rejected.
	notifRepo := notifications.NewRepository(pool)
	notifSvc := notifications.NewService(notifRepo)
	notifHandler := notifications.NewHandler(notifSvc)
	notifHandler.Register(api, authMW)

	mrSvc := materialrequests.NewService(mrRepo, notifications.NewMaterialRequestsAdapter(notifSvc))
	mrSvc.SetActivityLogger(activitylog.NewMaterialRequestsAdapter(actSvc))
	mrHandler := materialrequests.NewHandler(mrSvc)
	// Approval action requires admin or supervisor (procurementMW already covers both).
	mrHandler.Register(api, authMW, procurementMW)

	// Wire users management module (admin only).
	usrRepo := users.NewRepository(pool)
	usrSvc := users.NewService(usrRepo)
	usrSvc.SetActivityLogger(activitylog.NewUsersAdapter(actSvc))
	usrHandler := users.NewHandler(usrSvc)
	usrHandler.Register(api, authMW, adminMW)

	// Wire tools module (read+checkout for any user, mutate-catalog for admin).
	tlRepo := tools.NewRepository(pool)
	tlSvc := tools.NewService(tlRepo)
	tlSvc.SetActivityLogger(activitylog.NewToolsAdapter(actSvc))
	tlHandler := tools.NewHandler(tlSvc)
	tlHandler.Register(api, authMW, adminMW)

	// Wire reports module (read-only analytics for any authenticated user).
	rptRepo := reports.NewRepository(pool)
	rptSvc := reports.NewService(rptRepo)
	rptHandler := reports.NewHandler(rptSvc)
	rptHandler.Register(api, authMW)

	// Wire warehouse locations module (admin-only mutate, any-user read).
	wlRepo := warehouselocations.NewRepository(pool)
	wlSvc := warehouselocations.NewService(wlRepo)
	wlHandler := warehouselocations.NewHandler(wlSvc)
	wlHandler.Register(api, authMW, adminMW)

	// Wire support tickets module (any user submit, admin process).
	supRepo := support.NewRepository(pool)
	supSvc := support.NewService(supRepo)
	supHandler := support.NewHandler(supSvc)
	supHandler.Register(api, authMW, adminMW)

	return app
}

// errorHandler converts internal errors into JSON responses.
// Unexpected errors are logged and surfaced as a generic 500 to avoid leaking details.
func errorHandler(c *fiber.Ctx, err error) error {
	var fe *fiber.Error
	if errors.As(err, &fe) {
		return c.Status(fe.Code).JSON(fiber.Map{
			"error":   fe.Message,
			"code":    fe.Code,
			"success": false,
		})
	}
	log.Printf("[error] %s %s: %v", c.Method(), c.OriginalURL(), err)
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"error":   "internal server error",
		"code":    fiber.StatusInternalServerError,
		"success": false,
	})
}
