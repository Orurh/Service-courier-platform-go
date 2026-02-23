package main

import (
	"context"
	"os/signal"
	"syscall"

	_ "course-go-avito-Orurh/docs" 
	"course-go-avito-Orurh/internal/app"
)

// @title Service Courier API
// @version 1.0
// @description HTTP API for courier and delivery management
// @BasePath /
// @schemes http
func main() {
	ctxSignals, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	container := app.MustBuildContainer(ctxSignals)
	runner := app.NewRunner()
	runner.MustRun(container)
}
