package config

import (
	"fmt"
	"github.com/spf13/viper"
	"log"
	"os"
	"strings"
	"sync"
)

type DBConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

type NotionConfig struct {
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	RedirectURI  string `mapstructure:"redirect_uri"`
	StateSecret  string `mapstructure:"state_secret"`
}

type AppConfig struct {
	Env    string       `mapstructure:"env"`
	Port   string       `mapstructure:"port"`
	DB     DBConfig     `mapstructure:"db"`
	Notion NotionConfig `mapstructure:"notion"`
}

var (
	C        AppConfig
	loadOnce sync.Once
	loadErr  error
)

func LoadConfig(env string) (AppConfig, error) {
	loadOnce.Do(func() {
		// Resolve the effective env name
		if env == "" {
			env = os.Getenv("APP_ENV")
		}
		if env == "" {
			env = "local"
		}

		// Resolve the config directory
		configDir := os.Getenv("CONFIG_DIR")
		if configDir == "" {
			// Use /config in Docker; fallback to ./config locally
			if _, err := os.Stat("/config"); err == nil {
				configDir = "/config"
			} else {
				configDir = "./config"
			}
		}

		log.Printf("📁 Using config path: %s", configDir)
		log.Printf("🔧 Loading config for: %s environment", env)

		viper.SetConfigName("config." + env)
		viper.SetConfigType("yaml")
		viper.AddConfigPath(configDir)
		viper.AutomaticEnv()
		viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

		if err := viper.ReadInConfig(); err != nil {
			log.Printf("⚠️ Config file not found (%s): %v", env, err)
			// Not fatal — rely on env vars
		}

		if err := viper.Unmarshal(&C); err != nil {
			loadErr = fmt.Errorf("error unmarshalling config: %w", err)
			return
		}

		log.Printf("✅ Loaded config for: %s", C.Env)
	})

	return C, loadErr
}
