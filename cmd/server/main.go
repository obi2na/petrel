package main

import (
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/obi2na/petrel/config"
	"github.com/obi2na/petrel/internal/handler"

	"log"
)

func main() {
	_ = godotenv.Load() //load .env if present
	config.LoadConfig()

	router := gin.Default()
	handler.RegisterRoutes(router)

	log.Printf("Starting Petrel on port %s... \n", config.C.Port)
	err := router.Run()
	if err != nil {
		log.Printf("Starting Petrel on port %s... \n", config.C.Port)
	}
}
