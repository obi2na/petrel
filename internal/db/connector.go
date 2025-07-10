package db

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/obi2na/petrel/config"
	"log"
)

func Connect() (*pgxpool.Pool, error) {
	log.Println("connecting to database")
	dbConfigs := config.C.DB
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s",
		dbConfigs.User, dbConfigs.Password, dbConfigs.Host, dbConfigs.Port, dbConfigs.DBName,
	)

	dbpool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to create pgx pool: %w", err)
	}

	log.Println("pinging database")
	if err := dbpool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping pgx pool: %w", err)
	}
	log.Println("pinged database successfully")

	return dbpool, nil
}
