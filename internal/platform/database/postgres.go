package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
}

func NewPostgresDB(cfg Config) (*sql.DB, error) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)

	var db *sql.DB
	var err error
	maxRetries := 10

	for i := 1; i <= maxRetries; i++ {
		log.Printf("Connecting to database (Attempt %d/%d)...", i, maxRetries)
		db, err = sql.Open("postgres", connStr)
		if err == nil {
			err = db.Ping()
		}

		if err == nil {
			log.Println("Database connected successfully!")
			return db, nil
		}

		log.Printf("Database not ready yet. Waiting 2 seconds...")
		time.Sleep(2 * time.Second)
	}

	return nil, fmt.Errorf("gagal konek database: %v", err)
}
