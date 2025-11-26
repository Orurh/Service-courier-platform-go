package domain

type (
	// CourierStatus represents the status of a courier.
	CourierStatus string
	// CourierTransportType represents the transport type of a courier.
	CourierTransportType string
)

// Courier represents a delivery courier.
type Courier struct {
	ID            int64
	Name          string
	Phone         string
	Status        CourierStatus
	TransportType CourierTransportType
}

// PartialCourierUpdate carries optional fields to update a courier.
// A nil field means “do not change” that attribute.
type PartialCourierUpdate struct {
	ID            int64
	Name          *string
	Phone         *string
	Status        *CourierStatus
	TransportType *CourierTransportType
}
