package config

import (
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"log"
	"os"
	"sync"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	t.Setenv("CONFIG_DIR", "./testdata")
	tests := []struct {
		name           string
		env            string
		mockInjector   SecretInjector
		errExpected    bool
		expectedErr    string
		expectedResult AppConfig
	}{
		{
			name:           "test LoadConfig without secret injector",
			env:            "test",
			mockInjector:   nil,
			errExpected:    false,
			expectedResult: validEnvExpectedResult(),
		},
		{
			name:           "test LoadConfig with secret injector",
			env:            "test",
			mockInjector:   &mockSecretInjector{},
			errExpected:    false,
			expectedResult: injectedSecretExpectedResult(),
		},
		{
			name:         "test LoadConfig with no env specified",
			env:          "",
			mockInjector: nil,
			errExpected:  true,
			expectedErr:  "Config file not found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			defer resetConfigState()

			if err := os.Setenv("APP_ENV", tc.env); err != nil {
				t.Fatalf("TestLoadConfig Failed with %v", err)
			}

			c, err := LoadConfig(tc.env, tc.mockInjector)
			if tc.errExpected {
				assert.Contains(t, err.Error(), tc.expectedErr)
				return
			}

			//compare expected result to result
			if diff := cmp.Diff(tc.expectedResult, c); diff != "" {
				t.Errorf("unexpected config diff (-want +got):\n%s", diff)
			}
		})
	}

}

func validEnvExpectedResult() AppConfig {
	return AppConfig{
		Env:  "test",
		Port: "3001",
		Notion: NotionConfig{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			RedirectURI:  "http://localhost:3001/auth/notion/callback",
			StateSecret:  "secret-signing-key",
		},
	}
}

func injectedSecretExpectedResult() AppConfig {
	return AppConfig{
		Env:  "test",
		Port: "3001",
		Notion: NotionConfig{
			ClientID:     "injected-client-id",
			ClientSecret: "injected-client-secret",
			RedirectURI:  "http://localhost:3001/auth/notion/callback",
			StateSecret:  "injected-state-secret",
		},
	}
}

func resetConfigState() {
	C = AppConfig{}
	loadOnce = sync.Once{}
	loadErr = nil
}

type mockSecretInjector struct{}

func (m *mockSecretInjector) InjectSecrets(cfg *AppConfig) error {
	log.Println("✅ Mock injector called")
	cfg.Notion.ClientSecret = "injected-client-secret"
	cfg.Notion.ClientID = "injected-client-id"
	cfg.Notion.StateSecret = "injected-state-secret"
	return nil
}
