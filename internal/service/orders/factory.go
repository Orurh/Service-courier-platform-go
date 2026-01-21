package orders

import (
	"context"
	"strings"
)

type actionFunc func(context.Context, Event) error

type actionFactory struct {
	byStatus map[string]actionFunc
}

func newActionFactory(onCreated, onCanceled, onCompleted actionFunc) *actionFactory {
	return &actionFactory{
		byStatus: map[string]actionFunc{
			"created":   onCreated,
			"canceled":  onCanceled,
			"deleted":   onCanceled,
			"completed": onCompleted,
		},
	}
}

func (f *actionFactory) get(status string) (actionFunc, bool) {
	status = strings.ToLower(strings.TrimSpace(status))
	fn, ok := f.byStatus[status]
	return fn, ok
}
