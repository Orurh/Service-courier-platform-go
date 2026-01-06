package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"course-go-avito-Orurh/internal/logx"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func reqID(ctx context.Context) string {
	if id := middleware.GetReqID(ctx); id != "" {
		return id
	}
	return "-"
}

// логгер обязательный
func mustLogger(logger logx.Logger) logx.Logger {
	return logger
}

func writeJSON(logger logx.Logger, w http.ResponseWriter, r *http.Request, status int, v any) {
	logger = mustLogger(logger)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		logger.Error("json encode error", logx.String("req_id", reqID(r.Context())), logx.Any("err", err))
	}
}

type errResponse struct {
	Error string `json:"error"`
}

func writeError(logger logx.Logger, w http.ResponseWriter, r *http.Request, status int, msg string) {
	logger = mustLogger(logger)
	logger.Warn("http_error", logx.String("req_id", reqID(r.Context())), logx.Int("status", status), logx.String("msg", msg))
	writeJSON(logger, w, r, status, errResponse{Error: msg})
}

const (
	bodyLimit = 1 << 20
)

func decodeJSON[T any](logger logx.Logger, w http.ResponseWriter, r *http.Request, dst *T) bool {
	logger = mustLogger(logger)
	r.Body = http.MaxBytesReader(w, r.Body, bodyLimit)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		var mbe *http.MaxBytesError
		// добавил логирование о превышении лимита тела
		if errors.As(err, &mbe) {
			logger.Warn("body too large", logx.String("req_id", reqID(r.Context())), logx.Int64("limit_bytes",
				int64(mbe.Limit)), logx.Int64("content_length", r.ContentLength), logx.Any("err", err))
			writeError(logger, w, r, http.StatusRequestEntityTooLarge, "body too large")
			return false
		}

		logger.Warn("json decode error",
			logx.String("req_id", reqID(r.Context())),
			logx.Any("err", err),
		)
		writeError(logger, w, r, http.StatusBadRequest, "invalid json")
		return false
	}
	if err := dec.Decode(new(struct{})); err != io.EOF {
		logger.Warn("json trailing data", logx.String("req_id", reqID(r.Context())), logx.Any("err", err))
		writeError(logger, w, r, http.StatusBadRequest, "invalid json: trailing data")
		return false
	}
	return true
}

func idFromURL(r *http.Request, name string) (int64, error) {
	idStr := chi.URLParam(r, name)
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid id")
	}
	return id, nil
}
