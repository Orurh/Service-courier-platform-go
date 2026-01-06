package router_test

// import (
// 	"io"
// 	"log/slog"
// 	"testing"

// 	"course-go-avito-Orurh/internal/http/handlers"
// 	"course-go-avito-Orurh/internal/http/router"

// 	"github.com/stretchr/testify/require"
// )

// func TestNew_NotNil(t *testing.T) {
// 	t.Parallel()

// 	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
// 	base := handlers.New(logger)

// 	cour := &handlers.CourierHandler{}
// 	del := &handlers.DeliveryHandler{}

// 	require.NotPanics(t, func() {
// 		_ = router.New(base, cour, del)
// 	})
// }
