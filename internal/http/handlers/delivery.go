package handlers

import (
	"course-go-avito-Orurh/internal/apperr"
	"errors"
	"net/http"
)

// DeliveryHandler handles HTTP requests for delivery resources.
type DeliveryHandler struct {
	usecase deliveryUsecase
}

// NewDeliveryHandler creates a new DeliveryHandler.
func NewDeliveryHandler(uc deliveryUsecase) *DeliveryHandler {
	return &DeliveryHandler{usecase: uc}
}

// Assign handles the HTTP request for assigning a delivery.
func (h *DeliveryHandler) Assign(w http.ResponseWriter, r *http.Request) {
	var req assignDeliveryRequest
	if ok := decodeJSON(w, r, &req); !ok {
		return
	}

	res, err := h.usecase.Assign(r.Context(), req.OrderID)
	switch {
	case err == nil:
		writeJSON(w, r, http.StatusOK, assignResultToResponse(res))
	case errors.Is(err, apperr.Invalid):
		writeError(w, r, http.StatusBadRequest, "invalid input")
	case errors.Is(err, apperr.Conflict):
		writeError(w, r, http.StatusConflict, "no available couriers")
	default:
		writeError(w, r, http.StatusInternalServerError, "internal error")
	}
}

// Unassign handles the HTTP request for unassigning a delivery.
func (h *DeliveryHandler) Unassign(w http.ResponseWriter, r *http.Request) {
	var req unassignDeliveryRequest
	if ok := decodeJSON(w, r, &req); !ok {
		return
	}

	res, err := h.usecase.Unassign(r.Context(), req.OrderID)
	switch {
	case err == nil:
		writeJSON(w, r, http.StatusOK, unassignResultToResponse(res))
	case errors.Is(err, apperr.Invalid):
		writeError(w, r, http.StatusBadRequest, "invalid input")
	case errors.Is(err, apperr.NotFound):
		writeError(w, r, http.StatusNotFound, "delivery not found")
	default:
		writeError(w, r, http.StatusInternalServerError, "internal error")
	}
}
