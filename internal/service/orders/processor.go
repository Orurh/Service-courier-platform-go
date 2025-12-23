package orders

import (
	"context"
	"errors"

	"course-go-avito-Orurh/internal/apperr"
	"course-go-avito-Orurh/internal/domain"
	"course-go-avito-Orurh/internal/repository"
	"course-go-avito-Orurh/internal/service/delivery"
)

// Processor processes orders events
type Processor struct {
	delivery DeliveryPort
	repo     TxRunner
	factory  *actionFactory
}

// NewProcessorWithDeps creates a Processor from interfaces (handy for tests).
func NewProcessorWithDeps(deliverySvc DeliveryPort, repo TxRunner) *Processor {
	return newProcessor(deliverySvc, repo)
}

type repoAdapter struct {
	r *repository.DeliveryRepo
}

// WithTx opens a transaction and executes fn within it.
func (a repoAdapter) WithTx(ctx context.Context, fn func(tx delivery.TxRepository) error) error {
	return a.r.WithTx(ctx, fn)
}

// NewProcessor creates a new orders.Processor
func NewProcessor(deliverySvc *delivery.Service, repo *repository.DeliveryRepo) *Processor {
	return newProcessor(deliverySvc, repoAdapter{r: repo})
}

func newProcessor(deliverySvc DeliveryPort, repo TxRunner) *Processor {
	p := &Processor{
		delivery: deliverySvc,
		repo:     repo,
	}
	p.factory = newActionFactory(p.onCreated, p.onCanceled, p.onCompleted)
	return p
}

// Handle processes a single orders.Event
func (p *Processor) Handle(ctx context.Context, e Event) error {
	if p.factory == nil {
		return nil
	}
	fn, ok := p.factory.get(e.Status)
	if !ok {
		return nil
	}
	return fn(ctx, e)
}

func (p *Processor) onCreated(ctx context.Context, e Event) error {
	_, err := p.delivery.Assign(ctx, e.OrderID)
	if errors.Is(err, apperr.ErrConflict) {
		return nil
	}
	return err
}

func (p *Processor) onCanceled(ctx context.Context, e Event) error {
	_, err := p.delivery.Unassign(ctx, e.OrderID)
	if errors.Is(err, apperr.ErrNotFound) {
		return nil
	}
	return err
}

func (p *Processor) onCompleted(ctx context.Context, e Event) error {
	return p.repo.WithTx(ctx, func(tx delivery.TxRepository) error {
		d, err := tx.GetByOrderID(ctx, e.OrderID)
		if err != nil {
			return err
		}
		if d == nil {
			return nil
		}
		return tx.UpdateCourierStatus(ctx, d.CourierID, domain.StatusAvailable)
	})
}
