package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var DB *pgxpool.Pool

func ConnectDatabase() {
	loadEnv(".env", "backend/.env", "../.env")

	connString := os.Getenv("DATABASE_URL")
	if connString == "" {
		// Set these environment variables in your terminal or .env.
		// The fallback password is for local development only.
		host := getEnv("DB_HOST", "localhost")
		port := getEnv("DB_PORT", "5432")
		user := getEnv("DB_USER", "postgres")
		password := getEnv("DB_PASSWORD", "Adrian123")
		dbname := getEnv("DB_NAME", "hardware_sme_restore")

		connString = fmt.Sprintf(
			"postgres://%s:%s@%s:%s/%s",
			user,
			password,
			host,
			port,
			dbname,
		)
	}

	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		log.Fatal("Unable to configure PostgreSQL:", err)
	}

	// Allow multiple API requests to use different connections.
	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	DB, err = pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		log.Fatal("Unable to connect to PostgreSQL:", err)
	}

	if err := DB.Ping(ctx); err != nil {
		DB.Close()
		log.Fatal("PostgreSQL is not responding:", err)
	}

	if _, err := DB.Exec(
		ctx,
		`ALTER TABLE sales ALTER COLUMN customer_id DROP NOT NULL`,
	); err != nil {
		DB.Close()
		log.Fatal("Unable to enable walk-in sales:", err)
	}

	fmt.Println("Connected to PostgreSQL successfully!")
}

func CloseDatabase() {
	if DB != nil {
		DB.Close()
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)

	if value == "" {
		return fallback
	}

	return value
}

// loadEnv loads environment variables from key-value .env files if present.
func loadEnv(files ...string) {
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				k := strings.TrimSpace(parts[0])
				v := strings.TrimSpace(parts[1])
				v = strings.Trim(v, `"'`)
				if os.Getenv(k) == "" {
					os.Setenv(k, v)
				}
			}
		}
	}
}