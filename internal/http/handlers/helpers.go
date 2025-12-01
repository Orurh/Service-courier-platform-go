package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func reqID(ctx context.Context) string {
	if id := middleware.GetReqID(ctx); id != "" {
		return id
	}
	return "-"
}

func writeJSON(w http.ResponseWriter, r *http.Request, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		log.Printf("req_id=%s json encode error: %v", reqID(r.Context()), err)
	}
}

type errResponse struct {
	Error string `json:"error"`
}

func writeError(w http.ResponseWriter, r *http.Request, status int, msg string) {
	log.Printf("req_id=%s http_error status=%d msg=%q", reqID(r.Context()), status, msg)
	writeJSON(w, r, status, errResponse{Error: msg})
}

const (
	bodyLimit = 1 << 20
)

func decodeJSON[T any](w http.ResponseWriter, r *http.Request, dst *T) bool {
	r.Body = http.MaxBytesReader(w, r.Body, bodyLimit)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return false
	}
	if err := dec.Decode(new(struct{})); err != io.EOF {
		writeError(w, r, http.StatusBadRequest, "invalid json: trailing data")
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
