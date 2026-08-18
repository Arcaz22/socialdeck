package main

import (
	"log"
	"path/filepath"
	"runtime"

	"github.com/gin-gonic/gin"
	docs "github.com/socialdeck/backend/docs"
	"github.com/socialdeck/backend/internal/config"
	"github.com/socialdeck/backend/modules/auth"
	"github.com/socialdeck/backend/modules/deck"
	"github.com/socialdeck/backend/modules/game"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

//go:generate env GOFLAGS=-buildvcs=false go run github.com/swaggo/swag/cmd/swag init -g main.go --output docs --parseGoList=false

// @title Social Deck API
// @version 1.0
// @description Social Deck backend API.
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	// Load config
	cfg := config.Load()

	// Connect Postgres
	db, err := config.NewPostgres(cfg)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer db.Close()

	// Auto migrate
	_, filename, _, _ := runtime.Caller(0)
	root := filepath.Dir(filename)
	migrationsDir := filepath.Join(root, "migrations")

	if err := config.Migrate(db, migrationsDir); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	// Connect Redis
	rdb, err := config.NewRedis(cfg)
	if err != nil {
		log.Fatalf("redis: %v", err)
	}
	defer rdb.Close()

	// Wire up auth
	authRepo := auth.NewRepository(db)
	authService := auth.NewService(authRepo, rdb, cfg)
	authHandler := auth.NewHandler(authService, cfg)

	// Wire up deck
	deckRepo := deck.NewRepository(db)
	deckService := deck.NewService(deckRepo)
	deckHandler := deck.NewHandler(deckService)

	// Wire up game
	gameHub := game.NewHub()
	gameRepo := game.NewRepository(db)
	gameService := game.NewService(gameRepo, rdb)
	gameHandler := game.NewHandler(gameService, gameHub)

	// Router
	r := gin.Default()

	// Swagger docs
	docs.SwaggerInfo.BasePath = "/api/v1"
	r.GET("/docs", func(c *gin.Context) {
		c.Redirect(302, "/docs/index.html")
	})
	r.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := r.Group("/api/v1")
	auth.RegisterRoutes(api, authHandler, authService)
	deck.RegisterRoutes(api, deckHandler, authService)
	game.RegisterRoutes(api, gameHandler, authService)

	log.Printf("server running on :%s", cfg.AppPort)
	log.Printf("api docs available at http://localhost:%s/docs/index.html", cfg.AppPort)
	if err := r.Run(":" + cfg.AppPort); err != nil {
		log.Fatalf("server: %v", err)
	}
}
