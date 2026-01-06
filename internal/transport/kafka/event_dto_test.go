package kafka_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"course-go-avito-Orurh/internal/service/orders"
	"course-go-avito-Orurh/internal/transport/kafka"
)

func TestToDomain_TrimsAndCopiesFields(t *testing.T) {
	t.Parallel()

	ts := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)

	dto := kafka.EventDTO{
		OrderID:   "  order-1  ",
		Status:    "  created  ",
		CreatedAt: ts,
	}

	got := kafka.ToDomain(dto)

	require.Equal(t, orders.Event{
		OrderID:   "order-1",
		Status:    "created",
		CreatedAt: ts,
	}, got)
}
