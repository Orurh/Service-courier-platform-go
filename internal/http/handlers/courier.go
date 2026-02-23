package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"course-go-avito-Orurh/internal/apperr"
	"course-go-avito-Orurh/internal/logx"
)

// CourierHandler serves HTTP endpoints for courier resources.
type CourierHandler struct {
	usecase courierUsecase
	logger  logx.Logger
}

// NewCourierHandler wires a CourierUsecase into HTTP handlers.
func NewCourierHandler(logger logx.Logger, uc courierUsecase) *CourierHandler {
	return &CourierHandler{usecase: uc, logger: logger}
}

// GetByID handles GET /courier/{id}.
// @Summary Получить курьера по ID
// @Description Возвращает курьера по идентификатору
// @Tags couriers
// @Produce json
// @Param id path int true "Courier ID"
// @Success 200 {object} courierDTO
// @Failure 400 {object} ErrorResponse "invalid id"
// @Failure 404 {object} ErrorResponse "not found"
// @Failure 500 {object} ErrorResponse "internal error"
// @Router /courier/{id} [get]
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
// @Summary Список курьеров
// @Description Возвращает список курьеров с опциональной пагинацией (limit/offset)
// @Tags couriers
// @Produce json
// @Param limit query int false "Limit" minimum(0)
// @Param offset query int false "Offset" minimum(0)
// @Success 200 {array} courierDTO
// @Failure 400 {object} ErrorResponse "invalid limit/offset"
// @Failure 500 {object} ErrorResponse "internal error"
// @Router /couriers [get]
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
// @Summary Создать курьера
// @Description Создаёт нового курьера
// @Tags couriers
// @Accept json
// @Produce json
// @Param request body createCourierRequest true "Create courier payload"
// @Success 201 {object} IDResponse "created id"
// @Header 201 {string} Location "URL созданного ресурса"
// @Failure 400 {object} ErrorResponse "invalid input"
// @Failure 409 {object} ErrorResponse "phone already exists"
// @Failure 500 {object} ErrorResponse "internal error"
// @Router /courier [post]
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

// Update handles PUT /courier.
// @Summary Обновить курьера
// @Description Частично обновляет данные курьера по телу запроса
// @Tags couriers
// @Accept json
// @Produce json
// @Param request body createCourierRequest true "Create courier payload"
// @Success 200 {object} StatusResponse "status ok"
// @Failure 400 {object} ErrorResponse "invalid input"
// @Failure 404 {object} ErrorResponse "not found"
// @Failure 409 {object} ErrorResponse "phone already exists"
// @Failure 500 {object} ErrorResponse "internal error"
// @Router /courier [post]
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
