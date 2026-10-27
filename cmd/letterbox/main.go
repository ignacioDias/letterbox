package main

import (
	"context"
	"letterbox/internal/cache"
	"letterbox/internal/database"
	"letterbox/internal/repositories"
	"letterbox/internal/router"
	"letterbox/internal/server"
	"log"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	db, err := sqlx.Connect("pgx", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Successfully connected to PostgreSQL")

	database := database.NewDatabase(db)
	if err := database.Init(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	go cleanupExpiredSessions(database.SessionRepository)

	cache := cache.NewCache()

	defer cache.CloseClient()

	router := router.NewRouter(database, cache)
	server := server.NewServer(port, router)
	server.StartServer(router)
}

func cleanupExpiredSessions(repo repositories.SessionRepository) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		if err := repo.DeleteExpiredSessions(context.Background()); err != nil {
			log.Printf("Failed to delete expired sessions: %v", err)
		}
		<-ticker.C
	}
}
