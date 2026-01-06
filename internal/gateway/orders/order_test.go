package order_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	ordersgw "course-go-avito-Orurh/internal/gateway/orders"
	ordersproto "course-go-avito-Orurh/internal/proto"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type stubOrdersClient struct {
	getOrderByIdFn func(ctx context.Context, in *ordersproto.GetOrderByIdRequest, opts ...grpc.CallOption) (*ordersproto.GetOrderByIdResponse, error)
	getOrdersFn    func(ctx context.Context, in *ordersproto.GetOrdersRequest, opts ...grpc.CallOption) (*ordersproto.GetOrdersResponse, error)
}

func (s stubOrdersClient) GetOrderById(ctx context.Context, in *ordersproto.GetOrderByIdRequest, opts ...grpc.CallOption) (*ordersproto.GetOrderByIdResponse, error) {
	if s.getOrderByIdFn == nil {
		panic("GetOrderById not expected")
	}
	return s.getOrderByIdFn(ctx, in, opts...)
}

func (s stubOrdersClient) GetOrders(ctx context.Context, in *ordersproto.GetOrdersRequest, opts ...grpc.CallOption) (*ordersproto.GetOrdersResponse, error) {
	if s.getOrdersFn == nil {
		panic("GetOrders not expected")
	}
	return s.getOrdersFn(ctx, in, opts...)
}

func TestNewGRPCGateway_NilClient_ReturnsNil(t *testing.T) {
	gw := ordersgw.NewGRPCGateway(nil)
	require.Nil(t, gw)
}

func TestGRPCGateway_GetByID_ErrorWrapped(t *testing.T) {
	wantErr := errors.New("boom")

	client := stubOrdersClient{
		getOrderByIdFn: func(ctx context.Context, in *ordersproto.GetOrderByIdRequest, _ ...grpc.CallOption) (*ordersproto.GetOrderByIdResponse, error) {
			require.Equal(t, "order-1", in.GetId())
			return nil, wantErr
		},
	}
	gw := ordersgw.NewGRPCGateway(client)
	require.NotNil(t, gw)

	ord, err := gw.GetByID(context.Background(), "order-1")
	require.Nil(t, ord)
	require.ErrorIs(t, err, wantErr)
	require.True(t, strings.Contains(err.Error(), "order gateway: GetOrderById"))
}

func TestGRPCGateway_GetByID_NoOrder_ReturnsNil(t *testing.T) {
	client := stubOrdersClient{
		getOrderByIdFn: func(ctx context.Context, in *ordersproto.GetOrderByIdRequest, _ ...grpc.CallOption) (*ordersproto.GetOrderByIdResponse, error) {
			return &ordersproto.GetOrderByIdResponse{Order: nil}, nil
		},
	}
	gw := ordersgw.NewGRPCGateway(client)

	ord, err := gw.GetByID(context.Background(), "order-1")
	require.NoError(t, err)
	require.Nil(t, ord)
}

func TestGRPCGateway_GetByID_MapsFields(t *testing.T) {
	wantTime := time.Date(2025, 1, 2, 3, 4, 5, 123, time.UTC)

	client := stubOrdersClient{
		getOrderByIdFn: func(ctx context.Context, in *ordersproto.GetOrderByIdRequest, _ ...grpc.CallOption) (*ordersproto.GetOrderByIdResponse, error) {
			return &ordersproto.GetOrderByIdResponse{
				Order: &ordersproto.Order{
					Id:        "order-1",
					Status:    "CREATED",
					CreatedAt: timestamppb.New(wantTime),
				},
			}, nil
		},
	}
	gw := ordersgw.NewGRPCGateway(client)

	ord, err := gw.GetByID(context.Background(), "order-1")
	require.NoError(t, err)
	require.NotNil(t, ord)

	require.Equal(t, "order-1", ord.ID)
	require.Equal(t, "CREATED", ord.Status)
	require.True(t, ord.CreatedAt.Equal(wantTime))
}

func TestGRPCGateway_GetByID_CreatedAtNil_MapsZeroTime(t *testing.T) {
	client := stubOrdersClient{
		getOrderByIdFn: func(ctx context.Context, in *ordersproto.GetOrderByIdRequest, _ ...grpc.CallOption) (*ordersproto.GetOrderByIdResponse, error) {
			return &ordersproto.GetOrderByIdResponse{
				Order: &ordersproto.Order{
					Id:        "order-1",
					Status:    "CREATED",
					CreatedAt: nil,
				},
			}, nil
		},
	}
	gw := ordersgw.NewGRPCGateway(client)

	ord, err := gw.GetByID(context.Background(), "order-1")
	require.NoError(t, err)
	require.NotNil(t, ord)
	require.True(t, ord.CreatedAt.IsZero())
}

func TestGRPCGateway_ListFrom_SendsUTCFrom_AndMaps(t *testing.T) {
	from := time.Date(2025, 6, 7, 8, 9, 10, 11, time.FixedZone("X", 3*3600))
	wantFromUTC := from.UTC()

	o1Time := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	var capturedFrom time.Time

	client := stubOrdersClient{
		getOrdersFn: func(ctx context.Context, in *ordersproto.GetOrdersRequest, _ ...grpc.CallOption) (*ordersproto.GetOrdersResponse, error) {
			require.NotNil(t, in.GetFrom())
			capturedFrom = in.GetFrom().AsTime()

			return &ordersproto.GetOrdersResponse{
				Orders: []*ordersproto.Order{
					nil, // должен быть пропущен
					{Id: "o1", Status: "A", CreatedAt: timestamppb.New(o1Time)},
					{Id: "o2", Status: "B", CreatedAt: nil},
				},
			}, nil
		},
	}
	gw := ordersgw.NewGRPCGateway(client)

	list, err := gw.ListFrom(context.Background(), from)
	require.NoError(t, err)

	require.True(t, capturedFrom.Equal(wantFromUTC), "expected From in UTC")
	require.Len(t, list, 2)

	require.Equal(t, "o1", list[0].ID)
	require.True(t, list[0].CreatedAt.Equal(o1Time))

	require.Equal(t, "o2", list[1].ID)
	require.True(t, list[1].CreatedAt.IsZero())
}

func TestGRPCGateway_ListFrom_ErrorWrapped(t *testing.T) {
	wantErr := errors.New("boom")

	client := stubOrdersClient{
		getOrdersFn: func(ctx context.Context, in *ordersproto.GetOrdersRequest, _ ...grpc.CallOption) (*ordersproto.GetOrdersResponse, error) {
			return nil, wantErr
		},
	}
	gw := ordersgw.NewGRPCGateway(client)

	_, err := gw.ListFrom(context.Background(), time.Now())
	require.ErrorIs(t, err, wantErr)
	require.True(t, strings.Contains(err.Error(), "order gateway: GetOrders"))
}
