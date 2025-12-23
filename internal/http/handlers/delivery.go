package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"course-go-avito-Orurh/internal/apperr"
)

// DeliveryHandler handles HTTP requests for delivery resources.
type DeliveryHandler struct {
	usecase deliveryUsecase
	logger  *slog.Logger
}

// NewDeliveryHandler creates a new DeliveryHandler.
func NewDeliveryHandler(logger *slog.Logger, uc deliveryUsecase) *DeliveryHandler {
	if logger == nil {
		panic("delivery_handler: logger is nil")
	}
	return &DeliveryHandler{usecase: uc, logger: logger}
}

// Assign handles the HTTP request for assigning a delivery.
func (h *DeliveryHandler) Assign(w http.ResponseWriter, r *http.Request) {
	var req assignDeliveryRequest
	if ok := decodeJSON(h.logger, w, r, &req); !ok {
		return
	}

	res, err := h.usecase.Assign(r.Context(), req.OrderID)
	switch {
	case err == nil:
		writeJSON(h.logger, w, r, http.StatusOK, assignResultToResponse(res))
	case errors.Is(err, apperr.ErrInvalid):
		writeError(h.logger, w, r, http.StatusBadRequest, "invalid input")
	case errors.Is(err, apperr.ErrConflict):
		writeError(h.logger, w, r, http.StatusConflict, "no available couriers")
	default:
		writeError(h.logger, w, r, http.StatusInternalServerError, "internal error")
	}
}

// Unassign handles the HTTP request for unassigning a delivery.
func (h *DeliveryHandler) Unassign(w http.ResponseWriter, r *http.Request) {
	var req unassignDeliveryRequest
	if ok := decodeJSON(h.logger, w, r, &req); !ok {
		return
	}

	res, err := h.usecase.Unassign(r.Context(), req.OrderID)
	switch {
	case err == nil:
		writeJSON(h.logger, w, r, http.StatusOK, unassignResultToResponse(res))
	case errors.Is(err, apperr.ErrInvalid):
		writeError(h.logger, w, r, http.StatusBadRequest, "invalid input")
	case errors.Is(err, apperr.ErrNotFound):
		writeError(h.logger, w, r, http.StatusNotFound, "delivery not found")
	default:
		writeError(h.logger, w, r, http.StatusInternalServerError, "internal error")
	}
}
