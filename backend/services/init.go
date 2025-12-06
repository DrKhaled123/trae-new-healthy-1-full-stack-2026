package services

import (
	"log"
)

// InitializeServices initializes all services
func InitializeServices() {
	log.Println("Initializing services...")
	InitializeStorage()
	log.Println("Services initialized successfully")
}
