package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"go-learn/models"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/bytedance/gopkg/util/logger"
	_ "github.com/lib/pq"
	_ "github.com/stretchr/testify/assert"
	_ "github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Глобальные переменные для тестовой инфраструктуры
var (
	testProductDB            *sql.DB
	postgresProductContainer testcontainers.Container
)

var productLogger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
	Level: slog.LevelDebug,
}))

// TestMain - точка входа для всех тестов в пакете
func TestMain(m *testing.M) {
	// Настройка тестового окружения
	code, err := setupTestProductDatabase()
	if err != nil {
		fmt.Printf("❌ Ошибка при настройке тестовой БД: %v\n", err)
		os.Exit(1)
	}
	if code != 0 {
		os.Exit(code)
	}

	// Запуск всех тестов
	exitCode := m.Run()

	// Очистка ресурсов
	teardownProductTestDatabase()

	os.Exit(exitCode)
}

// Создаем тестовый контейнер PostgreSQL
func setupTestProductDatabase() (int, error) {
	logger.Debug("🚀 Запуск тестового контейнера PostgreSQL...")

	ctx := context.Background()

	// Описываем PostgreSQL контейнера
	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "testDB",
		},
		WaitingFor: wait.ForAll(
			wait.ForLog("database system is ready to accept connections"),
			wait.ForListeningPort("5432/tcp"),
		).WithDeadline(120 * time.Second),
	}

	// Запускаем контейнер
	var err error
	postgresProductContainer, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		logger.Error("❌ Ошибка создания контейнера", "error", err)
		return 1, err
	}

	// Получаем хост и порт
	host, err := postgresProductContainer.Host(ctx)
	if err != nil {
		logger.Error("❌ Ошибка получения хоста", "error", err)
		return 1, err
	}

	port, err := postgresProductContainer.MappedPort(ctx, "5432")
	if err != nil {
		logger.Error("❌ Ошибка получения порта", "error", err)
		return 1, err
	}

	// Формируем строку подключения
	dns := fmt.Sprintf("host=%s port=%s user=test password=test dbname=testDB sslmode=disable",
		host, port.Port())

	// Подключаемся к БД с повторными попытками
	testProductDB, err = connectWithRetriesProducts(dns, 10, 2*time.Second)
	if err != nil {
		logger.Error("Ошибка подключения к тестовой БД products", "error", err)
		return 1, err
	}

	// Создаем тестовую таблицу
	query := `
		CREATE TABLE IF NOT EXISTS products (
    		id SERIAL PRIMARY KEY,
    		name VARCHAR(255) NOT NULL,
    		price INTEGER NOT NULL CHECK (price > 0)
		);
	`

	_, err = testProductDB.Exec(query)
	if err != nil {
		logger.Error("Ошибка создания таблицы products", "error", err)
		return 1, err
	}

	fmt.Printf("✅ Контейнер запущен. Host: %s, Port: %s\n", host, port.Port())
	return 0, nil
}

// teardownTestDatabase очищает ресурсы после тестов
func teardownProductTestDatabase() {
	fmt.Printf("🧹 Очистка тестовых ресурсов...")

	if testProductDB != nil {
		// Очищаем данные перед закрытием
		_, err := testProductDB.Exec("TRUNCATE TABLE products RESTART IDENTITY CASCADE")
		if err != nil {
			fmt.Printf("⚠️  Ошибка очистки таблицы products: %v\n", err)
		}

		if err := testProductDB.Close(); err != nil {
			fmt.Printf("⚠️  Ошибка закрытия БД: %v\n", err)
		}
	}

	if postgresProductContainer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := postgresProductContainer.Terminate(ctx); err != nil {
			fmt.Printf("⚠️  Ошибка при остановке контейнера: %v\n", err)
		}
		//if removeError := postgresProductContainer.Terminate(context.Background()); removeError != nil {
		//	fmt.Printf("⚠️  Ошибка при принудительном удалении: %v\n", removeError)
		//}
	}

	fmt.Println("✅ Тестовые ресурсы очищены")
}

// Пытается подключиться к БД с повторными попытками
func connectWithRetriesProducts(dns string, maxRetries int, delay time.Duration) (*sql.DB, error) {
	var err error
	var db *sql.DB

	for i := 0; i < maxRetries; i++ {
		db, err = sql.Open("postgres", dns)
		if err != nil {
			time.Sleep(delay)
			continue
		}

		err = db.Ping()
		if err == nil {
			return db, nil
		}

		db.Close()
		time.Sleep(delay)
	}
	return nil, fmt.Errorf("could not connect to %s after %d retries: %w", dns, maxRetries, err)
}

// =========== ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ ===========

// Очищает таблицу и возвращает репозиторий
func createTestProduct(t *testing.T, name string, price int) (int, *ProductRepository) {
	t.Helper()

	// Очищает таблицу перед каждым тестом
	_, err := testProductDB.Exec("TRUNCATE TABLE products RESTART IDENTITY CASCADE")
	if err != nil {
		t.Fatalf("Failed to clean table products: %v", err)
	}

	repo := &ProductRepository{
		db:     testProductDB,
		logger: productLogger,
	}

	id, err := repo.CreateProduct(name, price)
	if err != nil {
		t.Fatalf("Failed to create product: %v", err)
	}

	return id, repo
}

// =========== ТЕСТЫ ===========

func TestProductRepository_CreateProduct(t *testing.T) {
	type args struct {
		name  string
		price int
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Create simple product",
			args: args{
				name:  " Test Product",
				price: 100,
			},
			wantErr: false,
		},
		{
			name: "Create product with zero price",
			args: args{
				name:  "Free Product",
				price: 0,
			},
			wantErr: false,
		},
		{
			name: "Create product with blank name",
			args: args{
				name:  "",
				price: 100,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, repo := createTestProduct(t, tt.args.name, tt.args.price)

			if id <= 0 {
				t.Errorf("CreateProduct() id = %v, want > 0", id)
			}

			// Проверяем, что продукт реально создался
			product, err := repo.GetProduct(id)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetProduct() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if product.Name != tt.args.name {
				t.Errorf("CreateProduct() got product name %v, want %v", product.Name, tt.args.name)
			}

			if product.Price != tt.args.price {
				t.Errorf("CreateProduct() got product price %v, want %v", product.Price, tt.args.price)
			}
		})
	}
}

func TestProductRepository_GetProduct(t *testing.T) {
	type args struct {
		id int
	}
	tests := []struct {
		name    string
		args    args
		want    *models.Product
		wantErr bool
	}{
		{
			name: "Get existing product",
			args: args{
				id: 0, // Будет заменен при создании
			},
			want: &models.Product{
				Name:  "Test Product",
				Price: 100,
			},
			wantErr: false,
		},
		{
			name: "Get non-existing product",
			args: args{
				id: 99999,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var repo *ProductRepository
			var targetID int

			if tt.args.id == 0 {
				id, r := createTestProduct(t, tt.want.Name, tt.want.Price)
				repo = r
				targetID = id
			} else {
				_, repo = createTestProduct(t, "dummy", 1)
				targetID = tt.args.id
			}

			got, err := repo.GetProduct(targetID)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetProduct() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if got.Name != tt.want.Name {
					t.Errorf("GetProduct() got product name %v, want %v", got.Name, tt.want.Name)
				}
				if got.Price != tt.want.Price {
					t.Errorf("GetProduct() got product price %v, want %v", got.Price, tt.want.Price)
				}
			}
		})
	}
}

func TestProductRepository_GetAllProducts(t *testing.T) {
	t.Run("Get all products from empty table", func(t *testing.T) {
		_, repo := createTestProduct(t, "dummy", 1)
		// Очищаем таблицу
		testProductDB.Exec("TRUNCATE TABLE products RESTART IDENTITY CASCADE")

		products, err := repo.GetAllProducts()
		if err != nil {
			t.Errorf("GetAllProducts() error = %v, want nil", err)
		}

		if len(products) != 0 {
			t.Errorf("GetAllProducts() = %v, want %v products", len(products), 0)
		}
	})

	t.Run("Get all products with data", func(t *testing.T) {
		_, repo := createTestProduct(t, "Product 1", 100)
		repo.CreateProduct("Product 2", 200)
		repo.CreateProduct("Product 3", 300)

		products, err := repo.GetAllProducts()
		if err != nil {
			t.Errorf("GetAllProducts() error = %v, want nil", err)
		}

		if len(products) != 3 {
			t.Errorf("GetAllProducts() got %d products, want 3", len(products))
		}
	})
}

func TestProductRepository_DeleteProduct(t *testing.T) {
	type args struct {
		id int
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Delete product Success",
			args: args{
				id: 0,
			},
			wantErr: false,
		},
		{
			name: "Delete product with non-existing product",
			args: args{
				id: 1000,
			},
			wantErr: true,
		},
		{
			name: "Try to delete product with negative ID",
			args: args{
				id: -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var repo *ProductRepository
			var targetID int

			if tt.args.id == 0 {
				id, r := createTestProduct(t, "To Delete", 100)
				repo = r
				targetID = id
			} else {
				_, repo = createTestProduct(t, "dummy", 1)
				targetID = tt.args.id
			}

			err := repo.DeleteProduct(targetID)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteProduct() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				_, err = repo.GetProduct(targetID)
				if err == nil {
					t.Errorf("DeleteProduct() product still exists")
				}
			}
		})
	}
}
