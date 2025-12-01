package delivery

import (
	"fmt"
	"time"

	"course-go-avito-Orurh/internal/domain"
)

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
