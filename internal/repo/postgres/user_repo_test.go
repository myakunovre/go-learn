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
	testUserDB            *sql.DB
	postgresUserContainer testcontainers.Container
)

var userLogger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
	Level: slog.LevelDebug,
}))

// TestMain - точка входа для всех тестов в пакете
func TestMain(m *testing.M) {
	// Настройка тестового окружения
	code, err := setupTestUserDatabase()
	if err != nil {
		fmt.Printf("❌ Ошибка при настройке тестовой БД users: %v\n", err)
		os.Exit(1)
	}
	if code != 0 {
		os.Exit(code)
	}

	// Запуск всех тестов
	exitCode := m.Run()

	// Очистка ресурсов
	teardownUserTestDatabase()

	os.Exit(exitCode)
}

// Создаем тестовый контейнер PostgreSQL
func setupTestUserDatabase() (int, error) {
	logger.Debug("🚀 Запуск тестового контейнера для пользователей...")

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
	postgresUserContainer, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		logger.Error("❌ Ошибка создания контейнера", "error", err)
		return 1, err
	}

	// Получаем хост и порт
	host, err := postgresUserContainer.Host(ctx)
	if err != nil {
		logger.Error("❌ Ошибка получения хоста", "error", err)
		return 1, err
	}

	port, err := postgresUserContainer.MappedPort(ctx, "5432")
	if err != nil {
		logger.Error("❌ Ошибка получения порта", "error", err)
		return 1, err
	}

	// Формируем строку подключения
	dns := fmt.Sprintf("host=%s port=%s user=test password=test dbname=testDB sslmode=disable",
		host, port.Port())

	// Подключаемся к БД с повторными попытками
	testUserDB, err = connectWithRetriesUsers(dns, 10, 2*time.Second)
	if err != nil {
		logger.Error("Ошибка подключения к тестовой БД users", "error", err)
		return 1, err
	}

	// Создаем тестовую таблицу
	query := `
		CREATE TABLE IF NOT EXISTS users (
    		id SERIAL PRIMARY KEY,
    		name VARCHAR(100) NOT NULL,
       		email VARCHAR(100) NOT NULL UNIQUE,
        	password_hash VARCHAR(100) NOT NULL
		);
	`

	_, err = testUserDB.Exec(query)
	if err != nil {
		logger.Error("Ошибка создания таблицы users", "error", err)
		return 1, err
	}

	fmt.Printf("✅ Контейнер запущен. Host: %s, Port: %s\n", host, port.Port())
	return 0, nil
}

// teardownTestDatabase очищает ресурсы после тестов
func teardownUserTestDatabase() {
	fmt.Printf("🧹 Очистка тестовых ресурсов...")

	if testUserDB != nil {
		// Очищаем данные перед закрытием
		_, err := testUserDB.Exec("TRUNCATE TABLE users RESTART IDENTITY CASCADE")
		if err != nil {
			fmt.Printf("⚠️  Ошибка очистки таблицы users: %v\n", err)
		}

		if err := testUserDB.Close(); err != nil {
			fmt.Printf("⚠️  Ошибка закрытия БД: %v\n", err)
		}
	}

	if postgresUserContainer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := postgresUserContainer.Terminate(ctx); err != nil {
			fmt.Printf("⚠️  Ошибка при остановке контейнера: %v\n", err)
		}
		//if removeError := postgresUserContainer.Terminate(context.Background()); removeError != nil {
		//	fmt.Printf("⚠️  Ошибка при принудительном удалении: %v\n", removeError)
		//}
	}

	fmt.Println("✅ Тестовые ресурсы очищены")
}

// Пытается подключиться к БД с повторными попытками
func connectWithRetriesUsers(dns string, maxRetries int, delay time.Duration) (*sql.DB, error) {
	var db *sql.DB
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		var err error
		db, err = sql.Open("postgres", dns)
		if err != nil {
			lastErr = err
			time.Sleep(delay)
			continue
		}

		err = db.Ping()
		if err == nil {
			return db, nil
		}
		lastErr = err

		_ = db.Close()
		time.Sleep(delay)
	}
	return nil, fmt.Errorf("could not connect to %s after %d retries: %w", dns, maxRetries, lastErr)
}

// =========== ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ ===========

// Очищает таблицу и возвращает репозиторий
func createTestUser(t *testing.T, name, email, password string) (int, *UserRepository) {
	t.Helper()

	// Очищает таблицу перед каждым тестом
	_, err := testUserDB.Exec("TRUNCATE TABLE users RESTART IDENTITY CASCADE")
	if err != nil {
		t.Fatalf("Failed to clean table: %v", err)
	}

	repo := &UserRepository{
		db:     testUserDB,
		logger: userLogger,
	}

	id, err := repo.CreateUser(name, email, password)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	return id, repo
}

// =========== ТЕСТЫ ===========

func TestUserRepository_CreateUser(t *testing.T) {
	type args struct {
		name     string
		email    string
		password string
	}
	tests := []struct {
		name string
		args args
		//want    int
		//wantErr assert.ErrorAssertionFunc
		wantErr bool
	}{
		{
			name: "Create simple user",
			args: args{
				name:     "testUser",
				email:    "testUser@test.ru",
				password: "test",
			},
			wantErr: false,
		},
		{
			name: "Create user with empty name",
			args: args{
				name:     "",
				email:    "testUser@test.ru",
				password: "testes-test",
			},
			wantErr: false,
		},
		{
			name: "Create user with empty email",
			args: args{
				name:     "testUser",
				email:    "",
				password: "testes-test",
			},
			wantErr: false,
		},
		{
			name: "Create user with empty password",
			args: args{
				name:     "testUser",
				email:    "testUser@test.ru",
				password: "",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, _ := createTestUser(t, tt.args.name, tt.args.email, tt.args.password)

			if id <= 0 {
				t.Errorf("CreateUser() id = %v, want > 0", id)
			}

			// Проверяем, что продукт реально создался
			var user models.User
			err := testUserDB.QueryRow(
				"SELECT id, name, email, password_hash FROM users WHERE id = $1", id,
			).Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if user.Name != tt.args.name {
				t.Errorf("GetUser() got user name %v, want %v", user.Name, tt.args.name)
			}

			if user.Email != tt.args.email {
				t.Errorf("GetUser() got user email %v, want %v", user.Email, tt.args.email)
			}
		})
	}
}
