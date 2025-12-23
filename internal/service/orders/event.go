package orders

import (
	"time"
)

// Event is a single order event
type Event struct {
	OrderID   string
	Status    string
	CreatedAt time.Time
}
