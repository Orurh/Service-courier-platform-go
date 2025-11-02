package config

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/spf13/pflag"
)

// Config stores HTTP service settings.
type Config struct{ Port int }

// Load reads configuration in order: .env (if present) → environment → flags.
func Load() (*Config, error) {
	if err := godotenv.Load(".env"); err != nil {
		log.Printf("warning: .env not loaded: %v", err)
	}

	port := 8080
	if v := os.Getenv("PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			port = p
		}
	}

	pflag.IntVarP(&port, "port", "p", port, "port to listen on")
	pflag.Parse()

	if port <= 0 || port > 65535 {
		return nil, fmt.Errorf("invalid port: %d", port)
	}
	return &Config{Port: port}, nil
}
