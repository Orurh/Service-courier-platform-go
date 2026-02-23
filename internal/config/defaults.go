package config

import "time"

const defaultPort = 8080

var defaultOrdersGateway = OrdersGateway{
	MaxAttempts: 4,
	BaseDelay:   150 * time.Millisecond,
	MaxDelay:    200 * time.Millisecond,
}

var defaultDB = DB{
	Host: "127.0.0.1",
	Port: "5432",
	User: "myuser",
	Pass: "mypassword",
	Name: "test_db",
}

const defaultOrderServiceHost = "localhost:50051"

var defaultDelivery = Delivery{
	AutoReleaseInterval: 10 * time.Second,
}

// DefaultPort returns the default port.
func DefaultPort() int {
	return defaultPort
}

// DefaultOrdersGateway returns the default orders gateway settings.
func DefaultOrdersGateway() OrdersGateway {
	return defaultOrdersGateway
}

// DefaultDB returns the default database settings.
func DefaultDB() DB {
	return defaultDB
}

// DefaultOrderServiceHost returns the default order service host.
func DefaultOrderServiceHost() string {
	return defaultOrderServiceHost
}

// DefaultDelivery returns the default delivery settings.
func DefaultDelivery() Delivery {
	return defaultDelivery
}
