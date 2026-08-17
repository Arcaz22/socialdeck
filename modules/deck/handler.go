package deck

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/socialdeck/backend/domain"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type DeckListResponse struct {
	Decks []domain.Deck `json:"decks"`
}

type DeckDetailResponse struct {
	Deck  *domain.Deck  `json:"deck"`
	Cards []domain.Card `json:"cards"`
}

type DeckResponse struct {
	Deck *domain.Deck `json:"deck"`
}

type CardResponse struct {
	Card *domain.Card `json:"card"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// ─── GET /decks ───────────────────────────────────────────────────────────────

// ListPublic godoc
// @Summary List public decks
// @Description Get all public decks, including system decks and public user decks.
// @Tags Decks
// @Produce json
// @Success 200 {object} DeckListResponse
// @Failure 500 {object} ErrorResponse
// @Router /decks [get]
func (h *Handler) ListPublic(c *gin.Context) {
	decks, err := h.service.ListPublic(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"decks": decks})
}

// ─── GET /decks/me ────────────────────────────────────────────────────────────

// ListMine godoc
// @Summary List my decks
// @Description Get all decks owned by the authenticated user.
// @Tags Decks
// @Produce json
// @Security BearerAuth
// @Success 200 {object} DeckListResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /decks/me [get]
func (h *Handler) ListMine(c *gin.Context) {
	userID := c.GetString("user_id")
	decks, err := h.service.ListMyDecks(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"decks": decks})
}

// ─── GET /decks/:id ───────────────────────────────────────────────────────────

// GetDeck godoc
// @Summary Get deck detail
// @Description Get a deck and its cards. Private decks are only available to their owner.
// @Tags Decks
// @Produce json
// @Security BearerAuth
// @Param id path string true "Deck ID"
// @Success 200 {object} DeckDetailResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /decks/{id} [get]
func (h *Handler) GetDeck(c *gin.Context) {
	userID := c.GetString("user_id")
	deckID := c.Param("id")

	deck, cards, err := h.service.GetDeck(c.Request.Context(), deckID, userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"deck":  deck,
		"cards": cards,
	})
}

// ─── POST /decks ──────────────────────────────────────────────────────────────

type CreateDeckRequest struct {
	Name     string `json:"name"      binding:"required,min=1,max=100"`
	Mode     string `json:"mode"      binding:"required"`
	IsPublic bool   `json:"is_public"`
}

// CreateDeck godoc
// @Summary Create a deck
// @Description Create a new deck owned by the authenticated user.
// @Tags Decks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateDeckRequest true "Create deck request"
// @Success 201 {object} DeckResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /decks [post]
func (h *Handler) CreateDeck(c *gin.Context) {
	var req CreateDeckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("user_id")
	deck, err := h.service.CreateDeck(c.Request.Context(), req.Name, req.Mode, req.IsPublic, userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"deck": deck})
}

// ─── PUT /decks/:id ───────────────────────────────────────────────────────────

type UpdateDeckRequest struct {
	Name     string `json:"name"      binding:"required,min=1,max=100"`
	Mode     string `json:"mode"      binding:"required"`
	IsPublic bool   `json:"is_public"`
}

// UpdateDeck godoc
// @Summary Update a deck
// @Description Update name, mode, and visibility for a deck owned by the authenticated user.
// @Tags Decks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Deck ID"
// @Param request body UpdateDeckRequest true "Update deck request"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /decks/{id} [put]
func (h *Handler) UpdateDeck(c *gin.Context) {
	var req UpdateDeckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("user_id")
	deckID := c.Param("id")

	if err := h.service.UpdateDeck(c.Request.Context(), deckID, req.Name, req.Mode, req.IsPublic, userID); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deck updated"})
}

// ─── DELETE /decks/:id ────────────────────────────────────────────────────────

// DeleteDeck godoc
// @Summary Delete a deck
// @Description Delete a deck owned by the authenticated user.
// @Tags Decks
// @Produce json
// @Security BearerAuth
// @Param id path string true "Deck ID"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /decks/{id} [delete]
func (h *Handler) DeleteDeck(c *gin.Context) {
	userID := c.GetString("user_id")
	deckID := c.Param("id")

	if err := h.service.DeleteDeck(c.Request.Context(), deckID, userID); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deck deleted"})
}

// ─── POST /decks/:id/cards ────────────────────────────────────────────────────

type AddCardRequest struct {
	Type    string `json:"type"    binding:"required"`
	Content string `json:"content" binding:"required,min=1"`
}

// AddCard godoc
// @Summary Add a card to a deck
// @Description Add a truth or dare card to a deck owned by the authenticated user.
// @Tags Decks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Deck ID"
// @Param request body AddCardRequest true "Add card request"
// @Success 201 {object} CardResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /decks/{id}/cards [post]
func (h *Handler) AddCard(c *gin.Context) {
	var req AddCardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("user_id")
	deckID := c.Param("id")

	card, err := h.service.AddCard(c.Request.Context(), deckID, req.Type, req.Content, userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"card": card})
}

// ─── DELETE /decks/:id/cards/:cardId ─────────────────────────────────────────

// DeleteCard godoc
// @Summary Delete a card from a deck
// @Description Delete a card from a deck owned by the authenticated user.
// @Tags Decks
// @Produce json
// @Security BearerAuth
// @Param id path string true "Deck ID"
// @Param cardId path string true "Card ID"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /decks/{id}/cards/{cardId} [delete]
func (h *Handler) DeleteCard(c *gin.Context) {
	userID := c.GetString("user_id")
	deckID := c.Param("id")
	cardID := c.Param("cardId")

	if err := h.service.DeleteCard(c.Request.Context(), deckID, cardID, userID); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "card deleted"})
}

// ─── Error Helper ─────────────────────────────────────────────────────────────

func (h *Handler) handleError(c *gin.Context, err error) {
	var invalidInput domain.ErrInvalidInput
	switch {
	case errors.Is(err, domain.ErrDeckNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrDeckForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrCardNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.As(err, &invalidInput):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
