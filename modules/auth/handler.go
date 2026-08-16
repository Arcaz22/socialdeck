package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/socialdeck/backend/domain"
	"github.com/socialdeck/backend/internal/config"
)

type Handler struct {
	service *Service
	cfg     *config.Config
}

func NewHandler(service *Service, cfg *config.Config) *Handler {
	return &Handler{service: service, cfg: cfg}
}

// ─── Register ────────────────────────────────────────────────────────────────

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=30"`
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type UserResponse struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

type RegisterResponse struct {
	Message string       `json:"message"`
	User    UserResponse `json:"user"`
}

type AccessTokenResponse struct {
	AccessToken string `json:"access_token"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

type MeResponse struct {
	User UserResponse `json:"user"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// Register godoc
// @Summary Register a new user
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Register request"
// @Success 201 {object} RegisterResponse
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auth/register [post]
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.service.Register(c.Request.Context(), req.Username, req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrEmailTaken):
			c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		case errors.Is(err, domain.ErrUsernameTaken):
			c.JSON(http.StatusConflict, gin.H{"error": "username already taken"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "registration successful",
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
		},
	})
}

// ─── Login ───────────────────────────────────────────────────────────────────

type LoginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Login godoc
// @Summary Login with email and password
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login request"
// @Success 200 {object} AccessTokenResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tokens, err := h.service.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredential) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	h.setRefreshCookie(c, tokens.RefreshToken)

	c.JSON(http.StatusOK, gin.H{
		"access_token": tokens.AccessToken,
	})
}

// ─── Refresh ─────────────────────────────────────────────────────────────────

// Refresh godoc
// @Summary Rotate refresh token and issue a new access token
// @Tags Auth
// @Produce json
// @Success 200 {object} AccessTokenResponse
// @Failure 401 {object} ErrorResponse
// @Router /auth/refresh [post]
func (h *Handler) Refresh(c *gin.Context) {
	refreshToken, err := c.Cookie(h.cfg.RefreshTokenCookieName)
	if err != nil || refreshToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing refresh token"})
		return
	}

	tokens, err := h.service.Refresh(c.Request.Context(), refreshToken)
	if err != nil {
		h.clearRefreshCookie(c)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
		return
	}

	h.setRefreshCookie(c, tokens.RefreshToken)

	c.JSON(http.StatusOK, gin.H{
		"access_token": tokens.AccessToken,
	})
}

// ─── Logout ──────────────────────────────────────────────────────────────────

// Logout godoc
// @Summary Logout the current user
// @Tags Auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} MessageResponse
// @Failure 401 {object} ErrorResponse
// @Router /auth/logout [post]
func (h *Handler) Logout(c *gin.Context) {
	refreshToken, err := c.Cookie(h.cfg.RefreshTokenCookieName)
	if err != nil {
		refreshToken = ""
	}

	accessToken := c.GetString("access_token")
	if err := h.service.LogoutSession(c.Request.Context(), accessToken, refreshToken); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
		return
	}

	h.clearRefreshCookie(c)
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

// ─── Me ──────────────────────────────────────────────────────────────────────

// Me godoc
// @Summary Get current authenticated user
// @Tags Auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} MeResponse
// @Failure 401 {object} ErrorResponse
// @Router /auth/me [get]
func (h *Handler) Me(c *gin.Context) {
	userID := c.GetString("user_id")
	user, err := h.service.GetCurrentUser(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": UserResponse{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			CreatedAt: user.CreatedAt,
		},
	})
}

// ─── Cookie Helpers ──────────────────────────────────────────────────────────

func (h *Handler) setRefreshCookie(c *gin.Context, token string) {
	maxAge := h.cfg.RefreshTokenTTLDays * 24 * 60 * 60
	sameSite := resolveSameSite(h.cfg.RefreshTokenCookieSameSite)

	c.SetSameSite(sameSite)
	c.SetCookie(
		h.cfg.RefreshTokenCookieName,
		token,
		maxAge,
		h.cfg.RefreshTokenCookiePath,
		"",
		h.cfg.RefreshTokenCookieSecure,
		true, // httpOnly — tidak bisa diakses JS
	)
}

func (h *Handler) clearRefreshCookie(c *gin.Context) {
	sameSite := resolveSameSite(h.cfg.RefreshTokenCookieSameSite)
	c.SetSameSite(sameSite)
	c.SetCookie(
		h.cfg.RefreshTokenCookieName,
		"",
		-1,
		h.cfg.RefreshTokenCookiePath,
		"",
		h.cfg.RefreshTokenCookieSecure,
		true,
	)
}

func resolveSameSite(s string) http.SameSite {
	switch s {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}
