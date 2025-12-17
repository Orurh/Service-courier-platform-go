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

// GetByID fetches an order by ID from the orders service.
func (g *GRPCGateway) GetByID(ctx context.Context, id string) (*Order, error) {
	resp, err := g.client.GetOrderById(ctx, &ordersproto.GetOrderByIdRequest{Id: id})
	if err != nil {
		return nil, fmt.Errorf("order gateway: GetOrderById: %w", err)
	}
	ord := resp.Order
	if ord == nil {
		return nil, nil
	}
	var createdAt time.Time
	if ord.CreatedAt != nil {
		createdAt = ord.CreatedAt.AsTime()
	}
	return &Order{
		ID:        ord.Id,
		Status:    ord.Status,
		CreatedAt: createdAt,
	}, nil
}

// ListFrom список заказов через gRPC, не используется
func (g *GRPCGateway) ListFrom(ctx context.Context, from time.Time) ([]Order, error) {
	req := &ordersproto.GetOrdersRequest{
		From: timestamppb.New(from.UTC()),
	}
	resp, err := g.client.GetOrders(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("order gateway: GetOrders: %w", err)
	}
	orders := make([]Order, 0, len(resp.Orders))
	for _, o := range resp.Orders {
		if o == nil {
			continue
		}
		ord := Order{
			ID:        o.Id,
			Status:    o.Status,
			CreatedAt: o.CreatedAt.AsTime(),
		}

		orders = append(orders, ord)
	}

	return orders, nil
}

// через http
// type Order struct {
// 	ID        string
// 	CreatedAt time.Time
// }

// type orderResponse struct {
//     ID        string    `json:"id"`
//     CreatedAt time.Time `json:"created_at"`
// }

// type Gateway interface {
// 	ListFrom(ctx context.Context, from time.Time) ([]Order, error)
// }

// type HTTPGateway struct {
//     baseURL string
//     client  *http.Client
// }

// func NewHTTPGateway(baseURL string, client *http.Client) *HTTPGateway {
//     if client == nil {
//         client = &http.Client{Timeout: 5 * time.Second}
//     }
//     return &HTTPGateway{
//         baseURL: baseURL,
//         client:  client,
//     }
// }

// func (g *HTTPGateway) ListFrom(ctx context.Context, from time.Time) ([]Order, error) {
// 	u, err := url.Parse(g.baseURL)
// 	if err != nil {
// 		return nil, fmt.Errorf("gateway: invalid base url: %w", err)
// 	}
// 	u.Path = strings.TrimRight(u.Path, "/") + "/public/api/v1/orders"

// 	q := u.Query()
// 	q.Set("from", from.UTC().Format(time.RFC3339Nano))
// 	u.RawQuery = q.Encode()

// 	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
// 	if err != nil {
// 		return nil, fmt.Errorf("gateway: new request: %w", err)
// 	}
// 	req.Header.Set("Accept", "application/json")

// 	resp, err := g.client.Do(req)
// 	if err != nil {
// 		return nil, fmt.Errorf("gateway: do request: %w", err)
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode != http.StatusOK {
// 		return nil, fmt.Errorf("gateway: unexpected status %d", resp.StatusCode)
// 	}

// 	var external []orderResponse
// 	if err := json.NewDecoder(resp.Body).Decode(&external); err != nil {
// 		return nil, fmt.Errorf("gateway: decode response: %w", err)
// 	}

//     orders := make([]Order, 0, len(external))
//     for _, eo := range external {
//         orders = append(orders, Order(eo))
//     }
//     return orders, nil

// }
