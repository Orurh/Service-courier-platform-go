package handlers

import "course-go-avito-Orurh/internal/domain"

type courierDTO struct {
	ID            int64                       `json:"id"`
	Name          string                      `json:"name"`
	Phone         string                      `json:"phone"`
	Status        domain.CourierStatus        `json:"status"`
	TransportType domain.CourierTransportType `json:"transport_type"`
}

type createCourierRequest struct {
	Name          string                      `json:"name"`
	Phone         string                      `json:"phone"`
	Status        domain.CourierStatus        `json:"status"`
	TransportType domain.CourierTransportType `json:"transport_type"`
}

type updateCourierRequest struct {
	ID            int64                        `json:"id"`
	Name          *string                      `json:"name,omitempty"`
	Phone         *string                      `json:"phone,omitempty"`
	Status        *domain.CourierStatus        `json:"status,omitempty"`
	TransportType *domain.CourierTransportType `json:"transport_type,omitempty"`
}
