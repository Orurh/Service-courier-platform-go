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
			"created": onCreated,
			// добавил что бы забрать заказ если курьер освободиться
			// пока не придумал как правильнее
			// "pending":    p.onCreated,
			// "confirmed":  p.onCreated,
			// "cooking":    p.onCreated,
			// "delivering": p.onCreated,
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
