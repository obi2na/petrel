package main

import (
	"flag"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/obi2na/petrel/config"
	"github.com/obi2na/petrel/internal/handler"

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

	router := gin.Default()
	handler.RegisterRoutes(router)

	log.Printf("Starting Petrel on port %s... \n", c.Port)
	err = router.Run()
	if err != nil {
		log.Printf("Starting Petrel on port %s... \n", c.Port)
	}
}
