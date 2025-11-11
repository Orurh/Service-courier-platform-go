package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"course-go-avito-Orurh/internal/app"
)

func main() {
	ctxSignals, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	container, err := app.BuildContainer(ctxSignals)
	if err != nil {
		log.Fatalf("failed to build container: %v", err)
	}

	if err := app.Run(container); err != nil {
		log.Fatalf("run error: %v", err)
	}
}
