package handlers

import "course-go-avito-Orurh/internal/domain"

func assignResultToResponse(result domain.AssignResult) assignDeliveryResponse {
	return assignDeliveryResponse{
		CourierID:        result.CourierID,
		OrderID:          result.OrderID,
		TransportType:    string(result.TransportType),
		DeliveryDeadline: result.Deadline,
	}
}

func unassignResultToResponse(result domain.UnassignResult) unassignDeliveryResponse {
	return unassignDeliveryResponse{
		OrderID:   result.OrderID,
		Status:    result.Status,
		CourierID: result.CourierID,
	}
}
