package metrics

import "github.com/prometheus/client_golang/prometheus"

// NewRateLimitExceededTotal returns a Prometheus counter for the number of rejected HTTP requests due to rate limiting
func NewRateLimitExceededTotal() prometheus.Counter {
	return prometheus.NewCounter(prometheus.CounterOpts{
		Name: "rate_limit_exceeded_total",
		Help: "Total number of rejected HTTP requests due to rate limiting",
	})
}

// NewGatewayRetriesTotal returns a Prometheus counter for the number of retry attempts performed by gateways
func NewGatewayRetriesTotal() prometheus.Counter {
	return prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gateway_retries_total",
		Help: "Total number of retry attempts performed by gateways",
	})
}
