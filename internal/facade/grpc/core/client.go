package core

import (
	"context"
	"fmt"
	"go-learn/api/grpc/proto/pb"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ProductGRPCClient struct {
	client pb.ProductServiceClient
	conn   *grpc.ClientConn
	logger *slog.Logger
}

func NewProductGRPCClient(addr string, logger *slog.Logger) (*ProductGRPCClient, error) {
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to core service (gRPC): %w", err)
	}

	client := pb.NewProductServiceClient(conn)

	return &ProductGRPCClient{
		client: client,
		conn:   conn,
		logger: logger,
	}, nil
}

func (c *ProductGRPCClient) GetProduct(ctx context.Context, productID int64) (*pb.ProductResponse, error) {
	req := &pb.GetProductRequest{
		ProductId: productID,
	}

	resp, err := c.client.GetProduct(ctx, req)
	if err != nil {
		c.logger.Error("failed to get product from core service", "productID", productID, "error", err)
		return nil, fmt.Errorf("failed to get product from core service: %w", err)
	}

	return resp, nil
}

func (c *ProductGRPCClient) Close() error {
	return c.conn.Close()
}
