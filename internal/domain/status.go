package domain

import "regexp"

// CourierStatus - type for courier status
type CourierStatus string

// List of possible courier statuses
const (
	StatusAvailable CourierStatus = "available"
	StatusBusy      CourierStatus = "busy"
	StatusPaused    CourierStatus = "paused"
)

// List of allowed statuses
var allowedStatuses = [...]CourierStatus{
	StatusAvailable, StatusBusy, StatusPaused,
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

// rePhone is a regex to validate phone numbers
var rePhone = regexp.MustCompile(`^\+[0-9]{11}$`)

// ValidatePhone validates the phone number format
func ValidatePhone(s string) bool {
	return rePhone.MatchString(s)
}
