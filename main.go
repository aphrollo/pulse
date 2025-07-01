package main

import (
	"log"

	"github.com/joho/godotenv"

	"github.com/aphrollo/pulse/app"
	db "github.com/aphrollo/pulse/storage"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found or failed to load")
	}

	if err := db.Connect(); err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer db.Close()

	api := app.New()

	if err := api.Listen(":3000"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
