package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort       string
	MigrationsDir string

	PostgresHost     string
	PostgresPort     string
	PostgresDB       string
	PostgresUser     string
	PostgresPassword string

	RedisURL string

	AuthSecretKey              string
	AccessTokenTTLMinutes      int
	RefreshTokenTTLDays        int
	RefreshTokenCookieName     string
	RefreshTokenCookieSecure   bool
	RefreshTokenCookieSameSite string
	RefreshTokenCookiePath     string
}

func Load() *Config {
	_ = godotenv.Load()

	accessTTL, _ := strconv.Atoi(os.Getenv("ACCESS_TOKEN_TTL_MINUTES"))
	refreshTTL, _ := strconv.Atoi(os.Getenv("REFRESH_TOKEN_TTL_DAYS"))
	cookieSecure, _ := strconv.ParseBool(os.Getenv("REFRESH_TOKEN_COOKIE_SECURE"))

	return &Config{
		AppPort:          getEnv("APP_PORT", "8080"),
		MigrationsDir:    getEnv("MIGRATIONS_DIR", "migrations"),
		PostgresHost:     getEnv("POSTGRES_HOST", "localhost"),
		PostgresPort:     getEnv("POSTGRES_PORT", "5432"),
		PostgresDB:       os.Getenv("POSTGRES_DB"),
		PostgresUser:     os.Getenv("POSTGRES_USER"),
		PostgresPassword: os.Getenv("POSTGRES_PASSWORD"),
		RedisURL:         os.Getenv("REDIS_URL"),

		AuthSecretKey:              os.Getenv("AUTH_SECRET_KEY"),
		AccessTokenTTLMinutes:      accessTTL,
		RefreshTokenTTLDays:        refreshTTL,
		RefreshTokenCookieName:     getEnv("REFRESH_TOKEN_COOKIE_NAME", "refresh_token"),
		RefreshTokenCookieSecure:   cookieSecure,
		RefreshTokenCookieSameSite: getEnv("REFRESH_TOKEN_COOKIE_SAMESITE", "lax"),
		RefreshTokenCookiePath:     getEnv("REFRESH_TOKEN_COOKIE_PATH", "/api/v1/auth"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
