package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"github.com/socialdeck/backend/domain"
	"github.com/socialdeck/backend/internal/config"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo  *Repository
	redis *redis.Client
	cfg   *config.Config
}

func NewService(repo *Repository, redis *redis.Client, cfg *config.Config) *Service {
	return &Service{repo: repo, redis: redis, cfg: cfg}
}

// ─── Register ────────────────────────────────────────────────────────────────

func (s *Service) Register(ctx context.Context, username, email, password string) (*domain.User, error) {
	emailTaken, err := s.repo.EmailExists(ctx, email)
	if err != nil {
		return nil, err
	}
	if emailTaken {
		return nil, domain.ErrEmailTaken
	}

	usernameTaken, err := s.repo.UsernameExists(ctx, username)
	if err != nil {
		return nil, err
	}
	if usernameTaken {
		return nil, domain.ErrUsernameTaken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	return s.repo.Create(ctx, username, email, string(hash))
}

// ─── Login ───────────────────────────────────────────────────────────────────

func (s *Service) Login(ctx context.Context, email, password string) (*domain.TokenPair, error) {
	user, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, domain.ErrInvalidCredential
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, domain.ErrInvalidCredential
	}

	return s.generateTokenPair(ctx, user.ID)
}

// ─── Refresh ─────────────────────────────────────────────────────────────────

func (s *Service) Refresh(ctx context.Context, refreshToken string) (*domain.TokenPair, error) {
	// Cek refresh token di Redis
	key := s.refreshKey(refreshToken)
	userID, err := s.redis.Get(ctx, key).Result()
	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	// Pastikan user masih ada
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil || user == nil {
		return nil, domain.ErrInvalidToken
	}

	// Hapus refresh token lama (rotation)
	s.redis.Del(ctx, key)

	// Buat token pair baru
	return s.generateTokenPair(ctx, user.ID)
}

// ─── Logout ──────────────────────────────────────────────────────────────────

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	return s.redis.Del(ctx, s.refreshKey(refreshToken)).Err()
}

func (s *Service) LogoutSession(ctx context.Context, accessToken, refreshToken string) error {
	if refreshToken != "" {
		if err := s.Logout(ctx, refreshToken); err != nil {
			return err
		}
	}

	if accessToken == "" {
		return nil
	}

	return s.RevokeAccessToken(ctx, accessToken)
}

func (s *Service) GetCurrentUser(ctx context.Context, userID string) (*domain.User, error) {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, domain.ErrUserNotFound
	}
	return user, nil
}

// ─── Validate Access Token ───────────────────────────────────────────────────

func (s *Service) ValidateAccessToken(ctx context.Context, tokenStr string) (string, error) {
	claims, err := s.parseAccessToken(tokenStr)
	if err != nil {
		return "", err
	}

	jti, ok := claims["jti"].(string)
	if !ok || jti == "" {
		return "", domain.ErrInvalidToken
	}

	revoked, err := s.redis.Exists(ctx, s.accessDenylistKey(jti)).Result()
	if err != nil {
		return "", domain.ErrInvalidToken
	}
	if revoked > 0 {
		return "", domain.ErrInvalidToken
	}

	userID, ok := claims["sub"].(string)
	if !ok || userID == "" {
		return "", domain.ErrInvalidToken
	}

	return userID, nil
}

func (s *Service) RevokeAccessToken(ctx context.Context, tokenStr string) error {
	claims, err := s.parseAccessToken(tokenStr)
	if err != nil {
		return domain.ErrInvalidToken
	}

	jti, ok := claims["jti"].(string)
	if !ok || jti == "" {
		return domain.ErrInvalidToken
	}

	exp, err := claims.GetExpirationTime()
	if err != nil || exp == nil {
		return domain.ErrInvalidToken
	}

	ttl := time.Until(exp.Time)
	if ttl <= 0 {
		return nil
	}

	return s.redis.Set(ctx, s.accessDenylistKey(jti), "revoked", ttl).Err()
}

func (s *Service) parseAccessToken(tokenStr string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(s.cfg.AuthSecretKey), nil
	})

	if err != nil || !token.Valid {
		return nil, domain.ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, domain.ErrInvalidToken
	}

	return claims, nil
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func (s *Service) generateTokenPair(ctx context.Context, userID string) (*domain.TokenPair, error) {
	// Access token
	accessToken, err := s.generateAccessToken(userID)
	if err != nil {
		return nil, err
	}

	// Refresh token
	refreshToken, err := s.generateRefreshToken()
	if err != nil {
		return nil, err
	}

	// Simpan refresh token di Redis
	ttl := time.Duration(s.cfg.RefreshTokenTTLDays) * 24 * time.Hour
	if err := s.redis.Set(ctx, s.refreshKey(refreshToken), userID, ttl).Err(); err != nil {
		return nil, err
	}

	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *Service) generateAccessToken(userID string) (string, error) {
	ttl := time.Duration(s.cfg.AccessTokenTTLMinutes) * time.Minute
	jti, err := s.generateRandomToken(16)
	if err != nil {
		return "", err
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"sub": userID,
		"jti": jti,
		"exp": now.Add(ttl).Unix(),
		"iat": now.Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).
		SignedString([]byte(s.cfg.AuthSecretKey))
}

func (s *Service) generateRefreshToken() (string, error) {
	return s.generateRandomToken(32)
}

func (s *Service) generateRandomToken(size int) (string, error) {
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *Service) refreshKey(token string) string {
	return fmt.Sprintf("refresh:%s", token)
}

func (s *Service) accessDenylistKey(jti string) string {
	return fmt.Sprintf("access_denylist:%s", jti)
}
