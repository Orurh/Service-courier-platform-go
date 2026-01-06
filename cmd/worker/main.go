package main

import (
	"context"
	"os/signal"
	"syscall"

	"course-go-avito-Orurh/internal/app"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	container := app.MustBuildWorkerContainer(ctx)
	app.NewWorkerRunner().MustRun(container)
}
