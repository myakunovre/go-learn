package order

import (
	"context"
	"fmt"
	"go-learn/api/grpc/proto/pb"

	"golang.org/x/exp/slog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type OrderGRPCClient struct {
	client pb.OrderServiceClient
	conn   *grpc.ClientConn
	logger *slog.Logger
}

func NewOrderGRPCClient(addr string, logger *slog.Logger) (*OrderGRPCClient, error) {
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to order service (gRPC): %w", err)
	}

	client := pb.NewOrderServiceClient(conn)

	return &OrderGRPCClient{
		client: client,
		conn:   conn,
		logger: logger,
	}, nil
}

func (c *OrderGRPCClient) GetBookedProducts(ctx context.Context) ([]*pb.BookedProduct, error) {
	req := &pb.GetBookedProductsRequest{}

	resp, err := c.client.GetBookedProducts(ctx, req)
	if err != nil {
		c.logger.Error("Failed to get booked products from order service", "error", err)
		return nil, fmt.Errorf("failed to get booked products: %w", err)
	}

	return resp.BookedProducts, nil
}

func (c *OrderGRPCClient) Close() error {
	return c.conn.Close()
}
