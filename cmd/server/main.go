package main

import (
	"database/sql"
	"log"
	"os"

	"config-manager/src/handlers"
	"config-manager/src/services"
	"config-manager/src/storage"

	_ "config-manager/docs"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/file"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/mattn/go-sqlite3"
	"github.com/swaggo/echo-swagger"
)

func main() {
	// Get database path from environment or use default
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data/config.db"
	}

	// Ensure data directory exists
	if err := os.MkdirAll("./data", 0755); err != nil {
		log.Fatal("Failed to create data directory:", err)
	}

	// Open database connection
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Failed to close database: %v", err)
		}
	}()

	// Run database migrations
	if err := runMigrations(dbPath); err != nil {
		log.Fatal("Failed to run migrations:", err)
	}

	// Initialize services
	validationService, err := services.NewValidationService()
	if err != nil {
		log.Fatal("Failed to create validation service:", err)
	}

	sqliteStore := storage.NewSQLiteStore(db)
	configService := services.NewConfigService(sqliteStore, validationService)
	configHandler := handlers.NewConfigHandler(configService)

	// Create Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Swagger UI endpoint
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	// Health check endpoint
	e.GET("/health", func(c echo.Context) error {
		// Test database connection
		if err := db.Ping(); err != nil {
			return c.JSON(503, map[string]string{
				"status":   "error",
				"database": "disconnected",
			})
		}

		return c.JSON(200, map[string]string{
			"status":   "ok",
			"database": "connected",
		})
	})

	// API routes
	api := e.Group("/api/v1")

	// Configuration endpoints
	api.POST("/configs", configHandler.CreateConfig)
	api.PUT("/configs/:name", configHandler.UpdateConfig)
	api.POST("/configs/:name/rollback", configHandler.RollbackConfig)
	api.GET("/configs/:name", configHandler.GetLatestConfig)
	api.GET("/configs/:name/versions/:version", configHandler.GetConfigVersion)
	api.GET("/configs/:name/versions", configHandler.ListVersions)

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Start server
	log.Printf("Starting server on port %s", port)
	if err := e.Start(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

// runMigrations applies database migrations using golang-migrate
func runMigrations(dbPath string) error {
	// Create database file if it doesn't exist
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		file, err := os.Create(dbPath)
		if err != nil {
			return err
		}
		if err := file.Close(); err != nil {
			return err
		}
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Failed to close migration database: %v", err)
		}
	}()

	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return err
	}

	// Use file source for migrations
	fileSource, err := (&file.File{}).Open("file://migrations")
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance("file", fileSource, "sqlite3", driver)
	if err != nil {
		return err
	}

	// Apply migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	log.Println("Database migrations applied successfully")
	return nil
}
