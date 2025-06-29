package config

import (
	"fmt"
	"github.com/spf13/viper"
	"log"
	"os"
	"strings"
	"sync"
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

var (
	C        AppConfig
	loadOnce sync.Once
	loadErr  error
)

func LoadConfig(env string) (AppConfig, error) {
	loadOnce.Do(func() {
		if err := os.Setenv("APP_ENV", env); err != nil {
			loadErr = err
			return
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
			loadErr = fmt.Errorf("unknown environment: %s", env)
			return
		}

		log.Printf("Loading config for: %s environment\n", env)

		viper.SetConfigName("config." + env)
		viper.SetConfigType("yaml")
		viper.AddConfigPath(configPath)
		viper.AutomaticEnv()
		viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

		if err := viper.ReadInConfig(); err != nil {
			loadErr = err
			return
		}

		if err := viper.Unmarshal(&C); err != nil {
			loadErr = err
			return
		}

		log.Printf("✅ Loaded config for: %s environment\n", C.Env)
	})

	return C, loadErr
}
