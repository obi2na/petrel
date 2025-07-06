package db

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/obi2na/petrel/config"
	"log"
)

var DB *sql.DB

func Connect() (*sql.DB, error) {
	log.Println("connecting to database")

	dbConfigs := config.C.DB

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbConfigs.Host, dbConfigs.Port, dbConfigs.User, dbConfigs.Password, dbConfigs.DBName,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open Database: %w", err)
	}

	log.Println("pinging database")
	//test database connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	log.Println("database connection completed successfully")

	DB = db
	return DB, nil
}
