package main

import (
	"letterbox/internal/database"
	"letterbox/internal/router"
	"letterbox/internal/server"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	port := os.Getenv("PORT")
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
	router := router.NewRouter(database)
	server := server.NewServer(port, router)
	server.StartServer(router)
}
