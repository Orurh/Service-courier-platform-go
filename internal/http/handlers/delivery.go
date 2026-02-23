package handlers

import (
	"errors"
	"net/http"

	"course-go-avito-Orurh/internal/apperr"
	"course-go-avito-Orurh/internal/logx"
)

// DeliveryHandler handles HTTP requests for delivery resources.
type DeliveryHandler struct {
	usecase deliveryUsecase
	logger  logx.Logger
}

// NewDeliveryHandler creates a new DeliveryHandler.
func NewDeliveryHandler(logger logx.Logger, uc deliveryUsecase) *DeliveryHandler {
	return &DeliveryHandler{usecase: uc, logger: logger}
}

// Assign handles POST /delivery/assign.
// @Summary Назначить доставку
// @Description Назначает курьера на заказ по order_id
// @Tags deliveries
// @Accept json
// @Produce json
// @Param request body assignDeliveryRequest true "Assign delivery payload"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse "invalid input"
// @Failure 409 {object} ErrorResponse "no available couriers"
// @Failure 500 {object} ErrorResponse "internal error"
// @Router /delivery/assign [post]
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

// Unassign handles POST /delivery/unassign.
// @Summary Снять назначение доставки
// @Description Снимает назначение курьера с заказа по order_id
// @Tags deliveries
// @Accept json
// @Produce json
// @Param request body unassignDeliveryRequest true "Unassign delivery payload"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse "invalid id"
// @Failure 404 {object} ErrorResponse "delivery not found"
// @Failure 500 {object} ErrorResponse "internal error"
// @Router /delivery/unassign [post]
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
