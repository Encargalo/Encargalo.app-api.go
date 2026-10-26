package groups

import (
	"Encargalo.app-api.go/internal/auth/domain/middleware/auth"
	"Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/banners/handler"
	"github.com/labstack/echo/v4"
)

type BannersGroup interface {
	Resource(g *echo.Echo)
}

type bannersGroup struct {
	handlerBanners      handler.BannersHandler
	handlerStaffBanners handler.StaffBannersHandler
	middleAuth          auth.AuthMiddleware
}

func NewBannersGroup(handlerBanners handler.BannersHandler, handlerStaffBanners handler.StaffBannersHandler, middleAuth auth.AuthMiddleware) BannersGroup {
	return &bannersGroup{handlerBanners, handlerStaffBanners, middleAuth}
}

func (bg *bannersGroup) Resource(g *echo.Echo) {
	customerRoutes := g.Group("/banners",
		bg.middleAuth.Auth,
		bg.middleAuth.Require(models.Customer),
	)
	customerRoutes.GET("", bg.handlerBanners.ListActiveBanners)

	staffRoutes := g.Group("/banners/staff",
		bg.middleAuth.Auth,
		bg.middleAuth.Require(models.Staff),
	)
	staffRoutes.POST("", bg.handlerStaffBanners.CreateBanner)
	staffRoutes.GET("", bg.handlerStaffBanners.ListBanners)
	staffRoutes.GET("/:id", bg.handlerStaffBanners.GetBanner)
	staffRoutes.PATCH("/:id", bg.handlerStaffBanners.UpdateBanner)
	staffRoutes.DELETE("/:id", bg.handlerStaffBanners.DeleteBanner)
}
