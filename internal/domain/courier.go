package domain

// Courier represents a delivery courier.
type Courier struct {
	ID     int64
	Name   string
	Phone  string
	Status string
}

// PartialCourierUpdate carries optional fields to update a courier.
// A nil field means “do not change” that attribute.
type PartialCourierUpdate struct {
	ID     int64
	Name   *string
	Phone  *string
	Status *string
}
