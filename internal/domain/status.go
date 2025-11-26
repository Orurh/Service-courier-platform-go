package domain

import "regexp"

// List of possible courier statuses
const (
	StatusAvailable CourierStatus = "available"
	StatusBusy      CourierStatus = "busy"
	StatusPaused    CourierStatus = "paused"
)

// List of possible courier transport types
const (
	TransportTypeFoot    CourierTransportType = "on_foot"
	TransportTypeScooter CourierTransportType = "scooter"
	TransportTypeCar     CourierTransportType = "car"
)

// List of allowed statuses
var allowedStatuses = [...]CourierStatus{
	StatusAvailable, StatusBusy, StatusPaused,
}

var allowedTransportTypes = [...]CourierTransportType{
	TransportTypeFoot, TransportTypeScooter, TransportTypeCar,
}

// Valid checks if the CourierStatus is valid
func (s CourierStatus) Valid() bool {
	for _, v := range allowedStatuses {
		if s == v {
			return true
		}
	}
	return false
}

// Valid checks if the CourierTransportType is valid
func (t CourierTransportType) Valid() bool {
	for _, v := range allowedTransportTypes {
		if t == v {
			return true
		}
	}
	return false
}

// rePhone is a regex to validate phone numbers
var rePhone = regexp.MustCompile(`^\+[0-9]{11}$`)

// ValidatePhone validates the phone number format
func ValidatePhone(s string) bool {
	return rePhone.MatchString(s)
}
