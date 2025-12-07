package handlers

import (
	"time"
)

type assignDeliveryRequest struct {
	OrderID string `json:"order_id"`
}

type assignDeliveryResponse struct {
	CourierID        int64     `json:"courier_id"`
	OrderID          string    `json:"order_id"`
	TransportType    string    `json:"transport_type"`
	DeliveryDeadline time.Time `json:"delivery_deadline"`
}

type unassignDeliveryRequest struct {
	OrderID string `json:"order_id"`
}

type unassignDeliveryResponse struct {
	OrderID   string `json:"order_id"`
	Status    string `json:"status"`
	CourierID int64  `json:"courier_id"`
}
