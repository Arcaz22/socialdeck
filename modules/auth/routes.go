package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/socialdeck/backend/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup, h *Handler, svc *Service) {
	auth := rg.Group("/auth")
	{
		auth.POST("/register", h.Register)
		auth.POST("/login", h.Login)
		auth.POST("/refresh", h.Refresh)

		// Protected — butuh access token
		auth.POST("/logout", middleware.RequireAuth(svc), h.Logout)
		auth.GET("/me", middleware.RequireAuth(svc), h.Me)
	}
}
