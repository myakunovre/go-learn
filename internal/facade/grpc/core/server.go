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
	pb.UnimplementedProductServiceServer
	prodService *product.ProductService
	logger      *slog.Logger
}

func NewProductGRPCServer(prodService *product.ProductService, logger *slog.Logger) *ProductGRPCServer {
	return &ProductGRPCServer{
		prodService: prodService,
		logger:      logger,
	}
}

func (s *ProductGRPCServer) GetProducts(ctx context.Context, req *pb.GetProductRequest) (*pb.ProductResponse, error) {
	IDs := req.GetProductIds()

	products, err := s.prodService.GetProductsByIDs(IDs)
	if err != nil {
		s.logger.Error("[gRPC ProductService] Error of getting products", "id", IDs, "error", err)
		return nil, fmt.Errorf("core get failed: %w", err)
	}

	var result []*pb.Product
	for _, p := range products {
		prod := &pb.Product{
			Id:     p.ID,
			Name:   p.Name,
			Price:  p.Price,
			Amount: p.Amount,
		}
		result = append(result, prod)
	}

	return &pb.ProductResponse{
		Products: result,
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
