package main

import (
	"context"
	"os/signal"
	"syscall"

	"course-go-avito-Orurh/internal/app"
)

func main() {
	ctxSignals, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	container := app.MustBuildContainer(ctxSignals)
	app.MustRun(container)
}
