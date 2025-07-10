package main

import (
	"flag"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/obi2na/petrel/config"
	"github.com/obi2na/petrel/internal/api"
	"github.com/obi2na/petrel/internal/db"
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
		log.Fatalf("Failed to load config : %v", err)
	}

	//connect to DB
	dbConn, err := db.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer dbConn.Close()
	log.Println("DB connection successful")

	logger.Init() //initialize singleton logger for application

	router := gin.Default()
	router.Use(middleware.RequestIDMiddleware()) //add Logger middleware to router. ensures request context has requestID
	router.Use(middleware.CORSMiddleware())      //add cors middleware to allow requests from frontend origin
	api.RegisterRoutes(router, dbConn)           //add handlers to router

	log.Printf("Starting Petrel on port %s... \n", c.Port)
	err = router.Run()
	if err != nil {
		log.Printf("Starting Petrel on port %s... \n", c.Port)
	}
}
