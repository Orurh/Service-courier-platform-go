package app

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/dig"

	"course-go-avito-Orurh/internal/logx"
)

func TestWorkerRunner_MustRun_NoPanicOnNil(t *testing.T) {
	r := &WorkerRunner{runFn: func(*dig.Container) error { return nil }}
	require.NotPanics(t, func() { r.MustRun(dig.New()) })
}

func TestWorkerRunner_MustRun_PanicsOnOtherError(t *testing.T) {
	sentinel := errors.New("boom")
	r := &WorkerRunner{runFn: func(*dig.Container) error { return sentinel }}
	require.Panics(t, func() { r.MustRun(dig.New()) })
}

func TestWorkerRun_ReturnsError_WhenConsumerNil(t *testing.T) {
	err := workerRun(context.Background(), nil, logx.Nop(), nil, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "kafka consumer is nil")
}
