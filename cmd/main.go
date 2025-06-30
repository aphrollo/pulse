package main

import (
	"github.com/aphrollo/pulse/internal/app"
	"log"
)

func main() {
	api := app.New()

	if err := api.Listen(":3000"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
