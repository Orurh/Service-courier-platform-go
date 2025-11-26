package router

import (
	"net/http"
	"testing"

	"course-go-avito-Orurh/internal/http/handlers"
)

func TestNew_NotNil(t *testing.T) {
	base := &handlers.Handlers{}
	cour := &handlers.CourierHandler{}
	del := &handlers.DeliveryHandler{}

	var _ http.Handler = New(base, cour, del)
}
