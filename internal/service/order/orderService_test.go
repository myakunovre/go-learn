package order

import (
	"context"
	"go-learn/internal/service/mocks"
	"log/slog"
	"os"
	"testing"

	"go.uber.org/mock/gomock"
)

var logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
	Level: slog.LevelDebug,
}))

func TestOrderService_BuyProduct(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockOrderRepo := mocks.NewMockOrderRepository(ctrl)
	mockOrderRepo.EXPECT().IncrementOrder(gomock.Any(), gomock.Any()).AnyTimes().Return((int64(0)), nil)

	type fields struct {
		repo   OrderRepository
		logger *slog.Logger
	}
	type args struct {
		ctx       context.Context
		productID int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int64
		wantErr bool
	}{
		{
			name: "success",
			fields: fields{
				repo:   mockOrderRepo,
				logger: logger,
			},
			args: args{
				ctx:       context.Background(),
				productID: 1,
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "success",
			fields: fields{
				repo:   mockOrderRepo,
				logger: logger,
			},
			args: args{
				ctx:       context.Background(),
				productID: 9999,
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "errorNegativeId",
			fields: fields{
				repo:   mockOrderRepo,
				logger: logger,
			},
			args: args{
				ctx:       context.Background(),
				productID: -1,
			},
			want:    0,
			wantErr: true,
		},
		{
			name: "errorZeroId",
			fields: fields{
				repo:   mockOrderRepo,
				logger: logger,
			},
			args: args{
				ctx:       context.Background(),
				productID: 0,
			},
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &OrderService{
				repo:   tt.fields.repo,
				logger: tt.fields.logger,
			}
			got, err := s.BuyProduct(tt.args.ctx, tt.args.productID)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuyProduct() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("BuyProduct() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOrderService_GetOrderCount(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockOrderRepo := mocks.NewMockOrderRepository(ctrl)
	mockOrderRepo.EXPECT().GetOrder(gomock.Any(), gomock.Any()).AnyTimes().Return((int64(0)), nil)

	type fields struct {
		repo   OrderRepository
		logger *slog.Logger
	}
	type args struct {
		ctx       context.Context
		productID int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int64
		wantErr bool
	}{
		{
			name: "success",
			fields: fields{
				repo:   mockOrderRepo,
				logger: logger,
			},
			args: args{
				ctx:       context.Background(),
				productID: 1,
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "success",
			fields: fields{
				repo:   mockOrderRepo,
				logger: logger,
			},
			args: args{
				ctx:       context.Background(),
				productID: 1000,
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "errorNegativeId",
			fields: fields{
				repo:   mockOrderRepo,
				logger: logger,
			},
			args: args{
				ctx:       context.Background(),
				productID: -1,
			},
			want:    0,
			wantErr: true,
		},
		{
			name: "errorZeroId",
			fields: fields{
				repo:   mockOrderRepo,
				logger: logger,
			},
			args: args{
				ctx:       context.Background(),
				productID: 0,
			},
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &OrderService{
				repo:   tt.fields.repo,
				logger: tt.fields.logger,
			}
			got, err := s.GetOrderCount(tt.args.ctx, tt.args.productID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetOrderCount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetOrderCount() got = %v, want %v", got, tt.want)
			}
		})
	}
}
