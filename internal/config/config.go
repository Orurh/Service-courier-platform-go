package config

import (
	"errors"
	"fmt"
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

	OrderService  string
	OrdersGateway OrdersGateway
	Kafka         Kafka
	Pprof         PprofConfig
}

// OrdersGateway stores orders gateway settings.
type OrdersGateway struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

// DB stores database settings.
type DB struct {
	Host string
	Port string
	User string
	Pass string
	Name string
}

// Kafka stores kafka settings.
type Kafka struct {
	Brokers []string
	Topic   string
	GroupID string
}

// Delivery stores delivery-related settings.
type Delivery struct {
	AutoReleaseInterval time.Duration
}

// PprofConfig stores pprof server settings.
type PprofConfig struct {
	Enabled bool
	Addr    string
	User    string
	Pass    string
}

// DSN returns database connection string.
func (d DB) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		d.User, d.Pass, d.Host, d.Port, d.Name)
}

func envOrDefault(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func readSecretFromFile(envKey string) (string, bool, error) {
	path := strings.TrimSpace(os.Getenv(envKey))
	if path == "" {
		return "", false, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", true, fmt.Errorf("read %s: %w", envKey, err)
	}
	val := strings.TrimSpace(string(b))
	return val, true, nil
}

func loadenv() error {
	// возвращаем ошибку на верх, но не выходим
	if err := godotenv.Load(".env"); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("load .env: %w", err)
	}
	return nil
}

func parsePort() (int, error) {
	port := defaultPort
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
			return 0, fmt.Errorf("parse flags: %w", err)
		}
	}
	if port <= 0 || port > 65535 {
		return 0, fmt.Errorf("invalid port: %d", port)
	}
	return port, nil
}

func parseDB() (DB, error) {
	pass := envOrDefault("POSTGRES_PASSWORD", defaultDB.Pass)
	if v, ok, err := readSecretFromFile("POSTGRES_PASSWORD_FILE"); err != nil {
		if !errors.Is(err, os.ErrNotExist) || strings.TrimSpace(pass) == "" {
			return DB{}, err
		}
	} else if ok {
		pass = v
	}

	db := DB{
		Host: envOrDefault("POSTGRES_HOST", defaultDB.Host),
		Port: envOrDefault("POSTGRES_PORT", defaultDB.Port),
		User: envOrDefault("POSTGRES_USER", defaultDB.User),
		Pass: pass,
		Name: envOrDefault("POSTGRES_DB", defaultDB.Name),
	}
	if _, err := strconv.Atoi(db.Port); err != nil {
		return DB{}, fmt.Errorf("invalid POSTGRES_PORT: %q", db.Port)
	}
	return db, nil
}

func parseDelivery() (Delivery, error) {
	intervalStr := envOrDefault("DELIVERY_AUTO_RELEASE_INTERVAL", defaultDelivery.AutoReleaseInterval.String())
	autoReleaseInterval, err := time.ParseDuration(intervalStr)
	if err != nil {
		return Delivery{}, fmt.Errorf("invalid DELIVERY_AUTO_RELEASE_INTERVAL %q: %w", intervalStr, err)
	}
	return Delivery{AutoReleaseInterval: autoReleaseInterval}, nil
}

func parseOrdersGateway() (orderService string, cfg OrdersGateway, err error) {
	orderService = envOrDefault("ORDER_SERVICE_HOST", defaultOrderServiceHost)

	maxAttemptsStr := envOrDefault("ORDER_GATEWAY_MAX_ATTEMPTS", strconv.Itoa(defaultOrdersGateway.MaxAttempts))
	maxAttempts, err := strconv.Atoi(maxAttemptsStr)
	if err != nil || maxAttempts < 1 || maxAttempts > 10 {
		return "", OrdersGateway{}, fmt.Errorf("invalid ORDER_GATEWAY_MAX_ATTEMPTS %q", maxAttemptsStr)
	}

	baseDelayStr := envOrDefault("ORDER_GATEWAY_BASE_DELAY", defaultOrdersGateway.BaseDelay.String())
	baseDelay, err := time.ParseDuration(baseDelayStr)
	if err != nil || baseDelay <= 0 {
		return "", OrdersGateway{}, fmt.Errorf("invalid ORDER_GATEWAY_BASE_DELAY %q: %w", baseDelayStr, err)
	}

	maxDelayStr := envOrDefault("ORDER_GATEWAY_MAX_DELAY", defaultOrdersGateway.MaxDelay.String())
	maxDelay, err := time.ParseDuration(maxDelayStr)
	if err != nil || maxDelay <= 0 {
		return "", OrdersGateway{}, fmt.Errorf("invalid ORDER_GATEWAY_MAX_DELAY %q: %w", maxDelayStr, err)
	}
	if maxDelay < baseDelay {
		return "", OrdersGateway{}, fmt.Errorf(
			"invalid ORDER_GATEWAY_MAX_DELAY %q: must be >= ORDER_GATEWAY_BASE_DELAY %q",
			maxDelayStr, baseDelayStr,
		)
	}

	return orderService, OrdersGateway{
		MaxAttempts: maxAttempts,
		BaseDelay:   baseDelay,
		MaxDelay:    maxDelay,
	}, nil
}

func parsePprof() (PprofConfig, error) {
	enabledStr := envOrDefault("PPROF_ENABLED", "false")
	enabled, err := strconv.ParseBool(enabledStr)
	if err != nil {
		return PprofConfig{}, fmt.Errorf("invalid PPROF_ENABLED %q: %w", enabledStr, err)
	}

	addr := strings.TrimSpace(envOrDefault("PPROF_ADDR", "127.0.0.1:6060"))
	if enabled && addr == "" {
		addr = "127.0.0.1:6060"
	}

	return PprofConfig{
		Enabled: enabled,
		Addr:    addr,
		User:    strings.TrimSpace(os.Getenv("PPROF_USER")),
		Pass:    strings.TrimSpace(os.Getenv("PPROF_PASS")),
	}, nil
}

// Load reads configuration in order: .env (if present) → environment → flags.
func Load() (*Config, error) {
	flagsMu.Lock()
	defer flagsMu.Unlock()
	if err := loadenv(); err != nil {
		return nil, err
	}

	port, err := parsePort()
	if err != nil {
		return nil, err
	}
	db, err := parseDB()
	if err != nil {
		return nil, err
	}
	deliveryCfg, err := parseDelivery()
	if err != nil {
		return nil, err
	}
	orderHost, gwCfg, err := parseOrdersGateway()
	if err != nil {
		return nil, err
	}
	kafkaCfg, err := loadKafka()
	if err != nil {
		return nil, err
	}

	pprofCfg, err := parsePprof()
	if err != nil {
		return nil, err
	}

	return &Config{Port: port, DB: db, Delivery: deliveryCfg, OrderService: orderHost, OrdersGateway: gwCfg, Kafka: kafkaCfg, Pprof: pprofCfg}, nil
}

func loadKafka() (Kafka, error) {
	brokersCSV := envOrDefault("KAFKA_BROKERS", "kafka:9092")
	raw := strings.Split(brokersCSV, ",")
	brokers := make([]string, 0, len(raw))
	for _, b := range raw {
		b = strings.TrimSpace(b)
		if b != "" {
			brokers = append(brokers, b)
		}
	}
	if len(brokers) == 0 {
		return Kafka{}, fmt.Errorf("invalid KAFKA_BROKERS: %q", brokersCSV)
	}

	cfg := Kafka{
		Brokers: brokers,
		Topic:   envOrDefault("KAFKA_ORDER_TOPIC", "order.status.changed"),
		GroupID: envOrDefault("KAFKA_GROUP_ID", "service-courier"),
	}

	return cfg, nil
}
