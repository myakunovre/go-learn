package core

import (
	"context"
	"fmt"
	"go-learn/api/grpc/proto/pb"
	"go-learn/internal/service/core/product"
	"log/slog"
	"net"

	"google.golang.org/grpc"
)

type ProductGRPCServer struct {
	*pb.UnimplementedProductServiceServer
	prodService *product.ProductService
	logger      *slog.Logger
}

func NewProductGRPCServer(prodService *product.ProductService, logger *slog.Logger) *ProductGRPCServer {
	return &ProductGRPCServer{
		prodService: prodService,
		logger:      logger,
	}
}

func (s *ProductGRPCServer) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.ProductResponse, error) {
	id := req.GetProductId()

	prod, err := s.prodService.Get(id)
	if err != nil {
		s.logger.Error("[gRPC ProductService] Error of getting product", "id", id, "error", err)
		return nil, fmt.Errorf("product get failed: %w", err)
	}

	return &pb.ProductResponse{
		Id:     prod.ID,
		Name:   prod.Name,
		Price:  prod.Price,
		Amount: prod.Amount,
	}, nil
}

func (s *ProductGRPCServer) Start(port string) error {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterProductServiceServer(grpcServer, s)

	s.logger.Info("[gRPC ProductService] Started on port " + port)
	return grpcServer.Serve(lis)
}
