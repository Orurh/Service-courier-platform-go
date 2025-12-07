package router_test

import (
	"net/http"
	"testing"

	"course-go-avito-Orurh/internal/http/handlers"
	"course-go-avito-Orurh/internal/http/router"
)

func TestNew_NotNil(t *testing.T) {
	base := &handlers.Handlers{}
	cour := &handlers.CourierHandler{}
	del := &handlers.DeliveryHandler{}

	var _ http.Handler = router.New(base, cour, del)
}
