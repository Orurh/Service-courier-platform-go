package domain

// Courier represents a delivery courier.
type Courier struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Phone  string `json:"phone"`
	Status string `json:"status"`
}

// PartialCourierUpdate carries optional fields to update a courier.
// A nil field means “do not change” that attribute.
type PartialCourierUpdate struct {
	ID     int64   `json:"id"`
	Name   *string `json:"name,omitempty"`
	Phone  *string `json:"phone,omitempty"`
	Status *string `json:"status,omitempty"`
}
