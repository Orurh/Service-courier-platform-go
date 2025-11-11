package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"course-go-avito-Orurh/internal/apperr"
	"course-go-avito-Orurh/internal/domain"
)

// CourierHandler serves HTTP endpoints for courier resources.
type CourierHandler struct{ uc CourierUsecase }

// NewCourierHandler wires a CourierUsecase into HTTP handlers.
func NewCourierHandler(uc CourierUsecase) *CourierHandler { return &CourierHandler{uc: uc} }

// GetByID handles GET /courier/{id}.
func (h *CourierHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := idFromURL(r, "id")
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid id")
		return
	}

	c, err := h.uc.Get(r.Context(), id)
	switch {
	case err == nil:
		writeJSON(w, r, http.StatusOK, c)
	case errors.Is(err, apperr.NotFound):
		writeError(w, r, http.StatusNotFound, "not found")
	default:
		writeError(w, r, http.StatusInternalServerError, "internal error")
	}
}

// List handles GET /couriers.
func (h *CourierHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var (
		limitPtr, offsetPtr *int
	)
	if s := q.Get("limit"); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil || v < 0 {
			writeError(w, r, http.StatusBadRequest, "invalid limit")
			return
		}
		limitPtr = &v
	}
	if s := q.Get("offset"); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil || v < 0 {
			writeError(w, r, http.StatusBadRequest, "invalid offset")
			return
		}
		offsetPtr = &v
	}

	list, err := h.uc.List(r.Context(), limitPtr, offsetPtr)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, r, http.StatusOK, list)
}

// Create handles POST /courier.
func (h *CourierHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req domain.Courier
	if ok := decodeJSON(w, r, &req); !ok {
		return
	}
	id, err := h.uc.Create(r.Context(), &req)
	switch {
	case err == nil:
		w.Header().Set("Location", "/courier/"+strconv.FormatInt(id, 10))
		writeJSON(w, r, http.StatusCreated, map[string]any{"id": id})
	case errors.Is(err, apperr.Invalid):
		writeError(w, r, http.StatusBadRequest, "invalid input")
	case errors.Is(err, apperr.Conflict):
		writeError(w, r, http.StatusConflict, "phone already exists")
	default:
		writeError(w, r, http.StatusInternalServerError, "internal error")
	}
}

// Update handles PUT /courier with partial updates from the request body.
func (h *CourierHandler) Update(w http.ResponseWriter, r *http.Request) {
	var req domain.PartialCourierUpdate
	if ok := decodeJSON(w, r, &req); !ok {
		return
	}
	_, err := h.uc.UpdatePartial(r.Context(), req)
	switch {
	case err == nil:
		writeJSON(w, r, http.StatusOK, map[string]string{"status": "ok"})
	case errors.Is(err, apperr.Invalid):
		writeError(w, r, http.StatusBadRequest, "invalid input")
	case errors.Is(err, apperr.Conflict):
		writeError(w, r, http.StatusConflict, "phone already exists")
	case errors.Is(err, apperr.NotFound):
		writeError(w, r, http.StatusNotFound, "not found")
	default:
		writeError(w, r, http.StatusInternalServerError, "internal error")
	}
}
