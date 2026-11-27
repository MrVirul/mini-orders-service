package handler

import (
	"database/sql"
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"github.com/MrVirul/mini-orders-service/internal/database"
	"github.com/MrVirul/mini-orders-service/internal/httpx"
	"github.com/MrVirul/mini-orders-service/internal/middleware"
	"github.com/MrVirul/mini-orders-service/internal/service"
)

type MenuHandler struct{}

func NewMenuHandler() *MenuHandler { return &MenuHandler{} }

func (h *MenuHandler) List(c *fiber.Ctx) error {
	t, ok := middleware.TenantFrom(c)
	if !ok || t.DB == nil {
		return httpx.Error(c, fiber.StatusInternalServerError,
			"internal_error", "Tenant missing from request context")
	}

	items, err := h.service(t.DB).List(c.UserContext())
	if err != nil {
		slog.Error("list menu items failed", "tenant_code", t.Code, "error", err)
		return httpx.Error(c, fiber.StatusInternalServerError,
			"internal_error", "Could not load menu")
	}
	return c.JSON(fiber.Map{"items": items})
}

func (h *MenuHandler) service(db *sql.DB) *service.MenuService {
	return service.NewMenuService(database.NewMenuRepository(db))
}
