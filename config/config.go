package config

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/spf13/viper"
	"google.golang.org/api/option"
)

type CORSConfig struct {
	AllowOrigins []string `mapstructure:"allow_origins"`
}

type Auth0Config struct {
	Domain       string `mapstructure:"domain"`
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	Connection   string `mapstructure:"connection"`
	RedirectURI  string `mapstructure:"redirect_uri"`
	StateSecret  string `mapstructure:"state_secret"`
}

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
	Auth0  Auth0Config  `mapstructure:"auth0"`
	CORS   CORSConfig   `mapstructure:"cors"`
}

var (
	C        AppConfig
	loadOnce sync.Once
	loadErr  error
)

func InitConfig(env string) (AppConfig, error) {
	// Resolve the effective env name
	if env == "" {
		env = os.Getenv("APP_ENV")
	}
	if env == "" {
		env = "local"
	}

	if env != "local" && env != "test" {
		return LoadConfig(env, &GCPSecretInjector{})
	}

	return LoadConfig(env, nil)
}

func LoadConfig(env string, injector SecretInjector) (AppConfig, error) {

	loadOnce.Do(func() {

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
			loadErr = fmt.Errorf("⚠️ Config file not found (%s): %v", env, err)
			return
		}

		if err := viper.Unmarshal(&C); err != nil {
			loadErr = fmt.Errorf("error unmarshalling config: %w", err)
			return
		}

		// 🔒 Load secrets from GCP Secret Manager
		if env != "local" && injector != nil {
			log.Println("🔐 Fetching secrets from Secret Manager...")
			if err := injector.InjectSecrets(&C); err != nil {
				loadErr = fmt.Errorf("failed to load secrets: %w", err)
				return
			}
			log.Println("✅ Secrets fetched successfully from Secret Manager")
		}

		log.Printf("✅ Loaded config for: %s", C.Env)
	})

	return C, loadErr
}

type SecretInjector interface {
	InjectSecrets(*AppConfig) error
}

type GCPSecretInjector struct{}

func (g *GCPSecretInjector) InjectSecrets(cfg *AppConfig) error {
	//create gcp secret manager client
	log.Println("Loading Secret Manager Client")
	ctx := context.Background()
	client, err := secretmanager.NewClient(ctx, option.WithCredentialsFile(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")))
	if err != nil {
		return fmt.Errorf("failed to create Secret Manager client: %w", err)
	}
	defer client.Close()
	log.Println("Secret Manager Client loaded successfully")

	projectId := os.Getenv("GCP_PROJECT_ID")
	if projectId == "" {
		return fmt.Errorf("GCP_PROJECT_ID environment variable is not set")
	}

	//Helper to fetch secrets
	get := func(secretID string) (string, error) {
		resourceName := fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectId, secretID)
		req := &secretmanagerpb.AccessSecretVersionRequest{
			Name: resourceName,
		}

		result, err := client.AccessSecretVersion(ctx, req)
		if err != nil {
			return "", fmt.Errorf("failed to access secret: %w", err)
		}
		return string(result.Payload.Data), nil
	}

	// Inject secrets
	var secrets = map[string]*string{
		"notion-client-id":    &cfg.Notion.ClientID,
		"notion-oauth-secret": &cfg.Notion.ClientSecret,
		"notion-state-secret": &cfg.Notion.StateSecret,
		"petrel-db-password":  &cfg.DB.Password,
		"petrel-db-name":      &cfg.DB.DBName,
		"auth0-client-secret": &cfg.Auth0.ClientSecret,
		"auth0-domain":        &cfg.Auth0.Domain,
		"auth0-client-id":     &cfg.Auth0.ClientID,
		"auth0-state-secret":  &cfg.Auth0.StateSecret,
	}

	for secretID, target := range secrets {
		val, err := get(secretID)
		if err != nil {
			return err
		}
		*target = val //dereference and place value
	}

	return nil
}
