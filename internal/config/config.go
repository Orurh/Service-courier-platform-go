package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/pflag"
)

var flagsMu sync.Mutex

// Config stores HTTP service settings.
type Config struct {
	Port     int
	DB       DB
	Delivery Delivery
}

// DB stores database settings.
type DB struct {
	Host string
	Port string
	User string
	Pass string
	Name string
}

// Delivery stores delivery-related settings.
type Delivery struct {
	AutoReleaseInterval time.Duration
}

// DSN returns database connection string.
func (d DB) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		d.User, d.Pass, d.Host, d.Port, d.Name)
}

// envOrDefault returns environment variable value if set, otherwise default.
func envOrDefault(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

// Load reads configuration in order: .env (if present) → environment → flags.
func Load() (*Config, error) {
	flagsMu.Lock()
	defer flagsMu.Unlock()
	if err := godotenv.Load(".env"); err != nil {
		log.Printf("warning: .env not loaded: %v", err)
	}

	port := 8080
	if v := os.Getenv("PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			port = p
		}
	}

	if pflag.CommandLine.Lookup("port") == nil {
		pflag.IntVarP(&port, "port", "p", port, "port to listen on")
	}

	if !pflag.CommandLine.Parsed() {
		if err := pflag.CommandLine.Parse(os.Args[1:]); err != nil {
			return nil, fmt.Errorf("parse flags: %w", err)
		}
	}

	if port <= 0 || port > 65535 {
		return nil, fmt.Errorf("invalid port: %d", port)
	}

	db := DB{
		Host: envOrDefault("POSTGRES_HOST", "127.0.0.1"),
		Port: envOrDefault("POSTGRES_PORT", "5432"),
		User: envOrDefault("POSTGRES_USER", "myuser"),
		Pass: envOrDefault("POSTGRES_PASSWORD", "mypassword"),
		Name: envOrDefault("POSTGRES_DB", "test_db"),
	}
	if _, err := strconv.Atoi(db.Port); err != nil {
		return nil, fmt.Errorf("invalid POSTGRES_PORT: %q", db.Port)
	}

	intervalStr := envOrDefault("DELIVERY_AUTO_RELEASE_INTERVAL", "10s")
	autoReleaseInterval, err := time.ParseDuration(intervalStr)
	if err != nil {
		return nil, fmt.Errorf("invalid DELIVERY_AUTO_RELEASE_INTERVAL %q: %w", intervalStr, err)
	}

	deliveryCfg := Delivery{
		AutoReleaseInterval: autoReleaseInterval,
	}

	return &Config{Port: port, DB: db, Delivery: deliveryCfg}, nil
}
