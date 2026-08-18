package game

import (
	"github.com/gin-gonic/gin"
	"github.com/socialdeck/backend/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup, h *Handler, validator middleware.TokenValidator) {
	auth := middleware.RequireAuth(validator)

	game := rg.Group("/game")
	{
		game.POST("/rooms", auth, h.CreateRoom)
		game.POST("/rooms/join", auth, h.JoinRoom)
		game.GET("/rooms/:id/state", auth, h.GetRoomState)
		game.POST("/rooms/:id/leave", auth, h.LeaveRoom)
		game.POST("/rooms/:id/ws-ticket", auth, h.CreateWebSocketTicket)

		game.GET("/rooms/:id/ws", h.WebSocket)
	}
}
