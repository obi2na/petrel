package config

import (
	"fmt"
	"github.com/spf13/viper"
	"log"
	"os"
	"strings"
)

type NotionConfig struct {
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	RedirectURI  string `mapstructure:"redirect_uri"`
}

type AppConfig struct {
	Env    string       `mapstructure:"env"`
	Port   string       `mapstructure:"port"`
	Notion NotionConfig `mapstructure:"notion"`
}

var C AppConfig

func LoadConfig() {
	fmt.Println("entering LoadConfig")

	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "local"
	}

	log.Printf("Loading config for: %s environment\n", C.Env)

	viper.SetConfigName("config." + env)
	viper.SetConfigType("yaml")
	viper.AddConfigPath("../../config") //move to root directory before using /config
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file: %#v", err)
	}

	log.Printf("✅ Loaded config for: %s environment\n", C.Env)
}
