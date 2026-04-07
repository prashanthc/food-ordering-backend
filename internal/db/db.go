package db

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"food-ordering/internal/config"

	_ "github.com/lib/pq"
)

func Connect(cfg *config.Config) *sql.DB {
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to open database connection: %v", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	for i := 0; i < 10; i++ {
		if err := db.Ping(); err == nil {
			break
		}
		log.Printf("Waiting for database... attempt %d", i+1)
		time.Sleep(2 * time.Second)
	}

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	return db
}

func Migrate(db *sql.DB) error {
	statements := []string{
		`CREATE EXTENSION IF NOT EXISTS "pgcrypto"`,
		`CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email VARCHAR(255) UNIQUE NOT NULL,
			name VARCHAR(255) NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS products (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			price DECIMAL(10,2) NOT NULL,
			category VARCHAR(100) NOT NULL,
			image_url TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS orders (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID REFERENCES users(id) ON DELETE CASCADE,
			idempotency_key VARCHAR(64),
			coupon_code VARCHAR(20),
			total_amount DECIMAL(10,2) NOT NULL,
			discount DECIMAL(10,2) DEFAULT 0,
			final_amount DECIMAL(10,2) NOT NULL,
			status VARCHAR(50) DEFAULT 'confirmed',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			CONSTRAINT uq_idempotency UNIQUE (user_id, idempotency_key)
		)`,
		`CREATE TABLE IF NOT EXISTS order_items (
			id SERIAL PRIMARY KEY,
			order_id UUID REFERENCES orders(id) ON DELETE CASCADE,
			product_id INTEGER REFERENCES products(id),
			quantity INTEGER NOT NULL,
			price DECIMAL(10,2) NOT NULL
		)`,
	}

	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("migration error: %w", err)
		}
	}

	return nil
}
