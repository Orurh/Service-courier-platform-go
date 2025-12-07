package domain

import "time"

// Delivery - struct representing a delivery assignment.
type Delivery struct {
	ID         int64
	CourierID  int64
	OrderID    string
	AssignedAt time.Time
	Deadline   time.Time
}

// AssignResult - struct representing the result of assigning a delivery.
type AssignResult struct {
	CourierID     int64
	OrderID       string
	TransportType CourierTransportType
	Deadline      time.Time
}

// UnassignResult - struct representing the result of unassigning a delivery.
type UnassignResult struct {
	CourierID int64
	OrderID   string
	Status    string
}
