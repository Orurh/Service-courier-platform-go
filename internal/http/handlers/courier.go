package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"course-go-avito-Orurh/internal/apperr"
)

// CourierHandler serves HTTP endpoints for courier resources.
type CourierHandler struct {
	usecase courierUsecase
	logger  *slog.Logger
}

// NewCourierHandler wires a CourierUsecase into HTTP handlers.
func NewCourierHandler(logger *slog.Logger, uc courierUsecase) *CourierHandler {
	if logger == nil {
		panic("courier_handler: logger is nil")
	}
	return &CourierHandler{usecase: uc, logger: logger}
}

// GetByID handles GET /courier/{id}.
func (h *CourierHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := idFromURL(r, "id")
	if err != nil {
		writeError(h.logger, w, r, http.StatusBadRequest, "invalid id")
		return
	}

	c, err := h.usecase.Get(r.Context(), id)
	switch {
	case err == nil:
		writeJSON(h.logger, w, r, http.StatusOK, modelToResponse(*c))
	case errors.Is(err, apperr.ErrNotFound):
		writeError(h.logger, w, r, http.StatusNotFound, "not found")
	default:
		writeError(h.logger, w, r, http.StatusInternalServerError, "internal error")
	}
}

// List handles GET /couriers.
func (h *CourierHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var limitPtr, offsetPtr *int
	if s := q.Get("limit"); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil || v < 0 {
			writeError(h.logger, w, r, http.StatusBadRequest, "invalid limit")
			return
		}
		limitPtr = &v
	}
	if s := q.Get("offset"); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil || v < 0 {
			writeError(h.logger, w, r, http.StatusBadRequest, "invalid offset")
			return
		}
		offsetPtr = &v
	}

	list, err := h.usecase.List(r.Context(), limitPtr, offsetPtr)
	if err != nil {
		writeError(h.logger, w, r, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(h.logger, w, r, http.StatusOK, modelsToResponse(list))
}

// Create handles POST /courier.
func (h *CourierHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createCourierRequest
	if ok := decodeJSON(h.logger, w, r, &req); !ok {
		return
	}
	id, err := h.usecase.Create(r.Context(), req.toModel())
	switch {
	case err == nil:
		w.Header().Set("Location", "/courier/"+strconv.FormatInt(id, 10))
		writeJSON(h.logger, w, r, http.StatusCreated, map[string]any{"id": id})
	case errors.Is(err, apperr.ErrInvalid):
		writeError(h.logger, w, r, http.StatusBadRequest, "invalid input")
	case errors.Is(err, apperr.ErrConflict):
		writeError(h.logger, w, r, http.StatusConflict, "phone already exists")
	default:
		writeError(h.logger, w, r, http.StatusInternalServerError, "internal error")
	}
}

// Update handles PUT /courier with partial updates from the request body.
func (h *CourierHandler) Update(w http.ResponseWriter, r *http.Request) {
	var req updateCourierRequest
	if ok := decodeJSON(h.logger, w, r, &req); !ok {
		return
	}
	_, err := h.usecase.UpdatePartial(r.Context(), req.toModel())
	switch {
	case err == nil:
		writeJSON(h.logger, w, r, http.StatusOK, map[string]string{"status": "ok"})
	case errors.Is(err, apperr.ErrInvalid):
		writeError(h.logger, w, r, http.StatusBadRequest, "invalid input")
	case errors.Is(err, apperr.ErrConflict):
		writeError(h.logger, w, r, http.StatusConflict, "phone already exists")
	case errors.Is(err, apperr.ErrNotFound):
		writeError(h.logger, w, r, http.StatusNotFound, "not found")
	default:
		writeError(h.logger, w, r, http.StatusInternalServerError, "internal error")
	}
}
