package db_test

import (
	"github.com/obi2na/petrel/config"
	"github.com/obi2na/petrel/internal/db"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConnect_Success(t *testing.T) {
	// Set config values manually for test environment (e.g., Docker Postgres)
	config.C.DB.Host = "localhost"
	config.C.DB.Port = "5432"
	config.C.DB.User = "postgres"
	config.C.DB.Password = "secret"
	config.C.DB.DBName = "petrel_test"

	conn, err := db.Connect()
	assert.NoError(t, err, "Expected no error connecting to database")
	assert.NotNil(t, conn, "Expected a valid DB connection")

	err = conn.Ping()
	assert.NoError(t, err, "Expected successful ping to the database")
}
