package product

import (
	"go-learn/internal/service/mocks"
	"go-learn/models"
	"log/slog"
	"os"
	"reflect"
	"testing"

	"go.uber.org/mock/gomock"
)

var logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
	Level: slog.LevelDebug,
}))

func TestProductService_Create(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockProductRepo := mocks.NewMockProductRepository(ctrl)

	mockProductRepo.EXPECT().CreateProduct(gomock.Any(), gomock.Any()).AnyTimes().Return(0, nil)

	type fields struct {
		repo   *mocks.MockProductRepository
		logger *slog.Logger
	}
	type args struct {
		name  string
		price int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int
		wantErr bool
	}{
		{
			name: "success",
			fields: fields{
				repo:   mockProductRepo,
				logger: logger,
			},
			args: args{
				name:  "Телефон",
				price: 1000,
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "success",
			fields: fields{
				repo:   mockProductRepo,
				logger: logger,
			},
			args: args{
				name:  "Телефон",
				price: 1,
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "SuccessShortName",
			fields: fields{
				repo:   mockProductRepo,
				logger: logger,
			},
			args: args{
				name:  "N",
				price: 1000,
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "SuccessTooLongName",
			fields: fields{
				repo:   mockProductRepo,
				logger: logger,
			},
			args: args{
				name:  "Nfkdlfmdsa;fmsalfmasl;fmlas;mlasmsam;asm;salmvlas;mvlasmvla;mva;mv;asmvlamva;s",
				price: 1000,
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "errorEmptyName",
			fields: fields{
				repo:   mockProductRepo,
				logger: logger,
			},
			args: args{
				name:  "",
				price: 1000,
			},
			want:    0,
			wantErr: true,
		},
		{
			name: "errorNegativePrice",
			fields: fields{
				repo:   mockProductRepo,
				logger: logger,
			},
			args: args{
				name:  "123",
				price: -1000,
			},
			want:    0,
			wantErr: true,
		},
		{
			name: "errorZeroPrice",
			fields: fields{
				repo:   mockProductRepo,
				logger: logger,
			},
			args: args{
				name:  "123",
				price: 0,
			},
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &ProductService{
				repo:   tt.fields.repo,
				logger: tt.fields.logger,
			}
			got, err := s.Create(tt.args.name, tt.args.price)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Create() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProductService_Delete(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockProductRepo := mocks.NewMockProductRepository(ctrl)

	mockProductRepo.EXPECT().DeleteProduct(gomock.Any()).AnyTimes().Return(nil)

	type fields struct {
		repo   *mocks.MockProductRepository
		logger *slog.Logger
	}
	type args struct {
		id int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "successId=1",
			fields: fields{
				repo:   mockProductRepo,
				logger: logger,
			},
			args: args{
				id: 1,
			},
			wantErr: false,
		},
		{
			name: "successId=1000",
			fields: fields{
				repo:   mockProductRepo,
				logger: logger,
			},
			args: args{
				id: 1000,
			},
			wantErr: false,
		},
		{
			name: "errorNegativeId=-1",
			fields: fields{
				repo:   mockProductRepo,
				logger: logger,
			},
			args: args{
				id: -1,
			},
			wantErr: true,
		},
		{
			name: "errorNegativeId=-1000",
			fields: fields{
				repo:   mockProductRepo,
				logger: logger,
			},
			args: args{
				id: -1000,
			},
			wantErr: true,
		},
		{
			name: "errorZeroId",
			fields: fields{
				repo:   mockProductRepo,
				logger: logger,
			},
			args: args{
				id: 0,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &ProductService{
				repo:   tt.fields.repo,
				logger: tt.fields.logger,
			}
			if err := s.Delete(tt.args.id); (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProductService_Get(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockProductRepo := mocks.NewMockProductRepository(ctrl)

	mockProductRepo.EXPECT().GetProduct(gomock.Any()).AnyTimes().Return(&models.Product{}, nil)

	type fields struct {
		repo   *mocks.MockProductRepository
		logger *slog.Logger
	}
	type args struct {
		id int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *models.Product
		wantErr bool
	}{
		{
			name: "successId=1",
			fields: fields{
				repo:   mockProductRepo,
				logger: logger,
			},
			args: args{
				id: 1,
			},
			want:    &models.Product{},
			wantErr: false,
		},
		{
			name: "successId=999",
			fields: fields{
				repo:   mockProductRepo,
				logger: logger,
			},
			args: args{
				id: 999,
			},
			want:    &models.Product{},
			wantErr: false,
		},
		{
			name: "errorNegativeId=-1",
			fields: fields{
				repo:   mockProductRepo,
				logger: logger,
			},
			args: args{
				id: -1,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "errorNegativeId=-999",
			fields: fields{
				repo:   mockProductRepo,
				logger: logger,
			},
			args: args{
				id: -999,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "errorZeroId",
			fields: fields{
				repo:   mockProductRepo,
				logger: logger,
			},
			args: args{
				id: 0,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &ProductService{
				repo:   tt.fields.repo,
				logger: tt.fields.logger,
			}
			got, err := s.Get(tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Get() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProductService_GetAllProducts(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockProductRepo := mocks.NewMockProductRepository(ctrl)

	mockProductRepo.EXPECT().GetAllProducts().AnyTimes().Return([]models.Product{}, nil)

	type fields struct {
		repo   *mocks.MockProductRepository
		logger *slog.Logger
	}
	tests := []struct {
		name    string
		fields  fields
		want    []models.Product
		wantErr bool
	}{
		{
			name: "success",
			fields: fields{
				repo:   mockProductRepo,
				logger: logger,
			},
			want:    []models.Product{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &ProductService{
				repo:   tt.fields.repo,
				logger: tt.fields.logger,
			}
			got, err := s.GetAllProducts()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAllProducts() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetAllProducts() got = %v, want %v", got, tt.want)
			}
		})
	}
}
