package order

import (
	"context"
	"fmt"
	"go-learn/api/grpc/proto/pb"
	"go-learn/internal/service/order"
	"log/slog"
	"net"

	"google.golang.org/grpc"
)

type OrderGRPCServer struct {
	pb.UnimplementedOrderServiceServer
	orderService      *order.OrderService
	orderCacheService *order.OrderCacheService
	logger            *slog.Logger
}

func NewProductGRPCServer(orderService *order.OrderService, orderCacheService *order.OrderCacheService, logger *slog.Logger) *OrderGRPCServer {
	return &OrderGRPCServer{
		orderService:      orderService,
		orderCacheService: orderCacheService,
		logger:            logger,
	}
}

func (s *OrderGRPCServer) GetBookedProducts(context context.Context, req *pb.GetBookedProductsRequest) (*pb.BookedProductsResponse, error) {

	bookedProducts, err := s.orderCacheService.GetAllProducts(context)
	if err != nil {
		s.logger.Error("[gRPC OrderService] Error of getting booked core", "error", err)
		return nil, fmt.Errorf("core get failed: %w", err)
	}

	// Преобразуем полученный слайс bookedProducts в слайс *pb.BookedProduct
	var result []*pb.BookedProduct
	for _, bookedProduct := range bookedProducts {
		prod := &pb.BookedProduct{
			ProductId: bookedProduct.ProductID,
			Amount:    int32(bookedProduct.Quantity),
		}
		result = append(result, prod)
	}

	return &pb.BookedProductsResponse{
		BookedProducts: result,
	}, nil
}

func (s *OrderGRPCServer) Start(port string) error {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterOrderServiceServer(grpcServer, s)

	s.logger.Info("[gRPC ProductService] Started on port " + port)
	return grpcServer.Serve(lis)
}
