package handlers

import "course-go-avito-Orurh/internal/domain"

type courierDTO struct {
	ID            int64                       `json:"id" example:"1"`
	Name          string                      `json:"name" example:"Иван"`
	Phone         string                      `json:"phone" example:"+79991234567"`
	Status        domain.CourierStatus        `json:"status" example:"active"`
	TransportType domain.CourierTransportType `json:"transport_type" example:"bike"`
}

type createCourierRequest struct {
	Name          string                      `json:"name" example:"Иван"`
	Phone         string                      `json:"phone" example:"+79991234567"`
	Status        domain.CourierStatus        `json:"status" example:"active"`
	TransportType domain.CourierTransportType `json:"transport_type" example:"bike"`
}

type updateCourierRequest struct {
	ID            int64                        `json:"id" example:"1"`
	Name          *string                      `json:"name,omitempty" example:"Иван"`
	Phone         *string                      `json:"phone,omitempty" example:"+79991234567"`
	Status        *domain.CourierStatus        `json:"status,omitempty" example:"active"`
	TransportType *domain.CourierTransportType `json:"transport_type,omitempty" example:"bike"`
}

// IDResponse contains created/returned entity identifier.
type IDResponse struct {
	ID int64 `json:"id" example:"1"`
}

// StatusResponse contains operation status.
type StatusResponse struct {
	Status string `json:"status" example:"ok"`
}

// ErrorResponse contains error message for failed requests.
type ErrorResponse struct {
	Error string `json:"error" example:"invalid input"`
}