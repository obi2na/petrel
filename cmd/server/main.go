package main

import (
	"flag"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/obi2na/petrel/config"
	"github.com/obi2na/petrel/internal/handler"
	"github.com/obi2na/petrel/internal/logger"
	"github.com/obi2na/petrel/internal/middleware"

	"log"
)

func main() {
	var env string
	flag.StringVar(&env, "env", "", "environment name")
	flag.Parse()

	_ = godotenv.Load() //load .env if present
	c, err := config.LoadConfig(env)
	if err != nil {
		log.Fatalf(err.Error())
	}

	logger.Init()

	router := gin.Default()
	router.Use(middleware.RequestIDMiddleware()) //add Logger middleware to router
	handler.RegisterRoutes(router)               //add handlers to router

	log.Printf("Starting Petrel on port %s... \n", c.Port)
	err = router.Run()
	if err != nil {
		log.Printf("Starting Petrel on port %s... \n", c.Port)
	}
}
