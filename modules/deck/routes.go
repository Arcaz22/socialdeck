package deck

import (
	"github.com/gin-gonic/gin"
	"github.com/socialdeck/backend/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup, h *Handler, validator middleware.TokenValidator) {
	auth := middleware.RequireAuth(validator)

	decks := rg.Group("/decks")
	{
		// Public — tidak perlu login
		decks.GET("", h.ListPublic)

		// Protected — butuh login
		decks.GET("/me", auth, h.ListMine)
		decks.GET("/:id", auth, h.GetDeck)
		decks.POST("", auth, h.CreateDeck)
		decks.PUT("/:id", auth, h.UpdateDeck)
		decks.DELETE("/:id", auth, h.DeleteDeck)

		// Cards
		decks.POST("/:id/cards", auth, h.AddCard)
		decks.DELETE("/:id/cards/:cardId", auth, h.DeleteCard)
	}
}
