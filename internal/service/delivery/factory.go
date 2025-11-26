//go:generate mockgen -source=factory.go -destination=timefactory_mocks_test.go -package=delivery \
package delivery

import (
	"fmt"
	"time"

	"course-go-avito-Orurh/internal/domain"
)

// TimeFactory is a factory for calculating delivery deadline.
type TimeFactory interface {
	Deadline(transport domain.CourierTransportType, now time.Time) (time.Time, error)
}

type defaultTimeFactory struct{}

// NewTimeFactory - creates a new TimeFactory.
func NewTimeFactory() TimeFactory {
	return defaultTimeFactory{}
}

// Deadline returns the delivery deadline based on the transport type and the current time.
func (defaultTimeFactory) Deadline(transport domain.CourierTransportType, now time.Time) (time.Time, error) {
	switch transport {
	case domain.TransportTypeFoot:
		return now.Add(30 * time.Minute), nil
	case domain.TransportTypeScooter:
		return now.Add(15 * time.Minute), nil
	case domain.TransportTypeCar:
		return now.Add(5 * time.Minute), nil
	default:
		return time.Time{}, fmt.Errorf("unknown transport type: %s", transport)
	}
}
