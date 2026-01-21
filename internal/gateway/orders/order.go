package order

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	ordersproto "course-go-avito-Orurh/internal/proto"
)

// Order represents an order from the orders service.
type Order struct {
	ID        string
	Status    string
	CreatedAt time.Time
}

// GRPCGateway is an orders gateway backed by gRPC.
type GRPCGateway struct {
	client ordersproto.OrdersServiceClient
}

// NewGRPCGateway creates an orders gateway backed by gRPC.
func NewGRPCGateway(client ordersproto.OrdersServiceClient) *GRPCGateway {
	if client == nil {
		return nil
	}
	return &GRPCGateway{client: client}
}

func mapProtoOrder(o *ordersproto.Order) Order {
	var createdAt time.Time
	if ts := o.GetCreatedAt(); ts != nil {
		createdAt = ts.AsTime()
	}
	return Order{
		ID:        o.GetId(),
		Status:    o.GetStatus(),
		CreatedAt: createdAt,
	}
}

// GetByID fetches an order by ID from the orders service.
func (g *GRPCGateway) GetByID(ctx context.Context, id string) (*Order, error) {
	resp, err := g.client.GetOrderByID(ctx, &ordersproto.GetOrderByIDRequest{Id: id})
	if err != nil {
		return nil, fmt.Errorf("order gateway: GetOrderByID: %w", err)
	}
	if resp.Order == nil {
		return nil, nil
	}
	ord := mapProtoOrder(resp.GetOrder())
	return &ord, nil
}

// ListFrom fetches orders from the orders service.
func (g *GRPCGateway) ListFrom(ctx context.Context, from time.Time) ([]Order, error) {
	req := &ordersproto.GetOrdersRequest{
		From: timestamppb.New(from.UTC()),
	}
	resp, err := g.client.GetOrders(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("order gateway: GetOrders: %w", err)
	}
	orders := make([]Order, 0, len(resp.Orders))
	for _, o := range resp.GetOrders() {
		if o == nil {
			continue
		}
		orders = append(orders, mapProtoOrder(o))
	}

	return orders, nil
}
