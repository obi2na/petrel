package config

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	t.Setenv("CONFIG_DIR", ".")
	if err := os.Setenv("APP_ENV", "test"); err != nil {
		t.Fatalf("TestLoadConfig Failed with %v", err)
	}
	c, _ := LoadConfig("test")

	assert.Equal(t, "test", c.Env)
	assert.Equal(t, "3001", c.Port)
}
