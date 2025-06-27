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

func LoadConfig(env string) (AppConfig, error) {
	var c AppConfig

	if err := os.Setenv("APP_ENV", env); err != nil {
		return c, err
	}

	if env == "" {
		env = "local"
	}

	configPaths := map[string]string{ //move to root directory before using /config
		"local": "../../config",
		"test":  "../config",
	}

	configPath, ok := configPaths[env]
	if !ok {
		return c, fmt.Errorf("unknown environment: %s", env)
	}

	log.Printf("Loading config for: %s environment\n", env)

	viper.SetConfigName("config." + env)
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configPath)
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		return c, err
	}

	if err := viper.Unmarshal(&c); err != nil {
		return c, err
	}

	log.Printf("✅ Loaded config for: %s environment\n", c.Env)
	return c, nil
}
