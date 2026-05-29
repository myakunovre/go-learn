package main

import (
	"database/sql"
	"fmt"
	"go-learn/repo"
	"go-learn/service"
	"log"
	"net/http"
	"strconv"

	_ "github.com/gin-gonic/gin"
	_ "github.com/lib/pq" // Драйвер для Postgres
)

func main() {
	// === ПОДКЛЮЧЕНИЕ К БАЗЕ ДАННЫХ ===
	// Строка подключения к Postgres
	connStr := "host=localhost port=5432 user=user password=password dbname=products_db sslmode=disable"

	// Подключение к БД
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}
	defer db.Close() // Закроем соединение когда программа завершится

	// Проверка соединения
	err = db.Ping()
	if err != nil {
		log.Fatalf("Не удалось подключиться к Postgres: %v", err)
	}
	fmt.Println("✅ Успешно подключено к Postgres!!!")

	// === СОЗДАЕМ ТАБЛИЦУ ТОВАРОВ ===
	createTableProducts(db)

	productRepository := repo.NewProductRepository(db)
	serv := service.NewProductService(productRepository)

	// === НАСТРАИВАЕМ ВЕБ-СЕРВЕР ===
	// Настройка HTTP-эндпоинта
	http.HandleFunc("/product/delete", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json") // ✅ Добавлен заголовок

		// Проверка метода
		if r.Method != http.MethodDelete {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(`{"error":"Method not allowed, use DELETE"}`))
			return
		}

		// Получаем ID из query параметра
		idStr := r.URL.Query().Get("id")
		if idStr == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"id parameter is required"}`))
			return
		}

		// Конвертируем в число
		id, err := strconv.Atoi(idStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"invalid id format"}`))
			return
		}

		// Удаляем
		err = serv.Delete(id)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf(`{"error":"%v"}`, err)))
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Println("Product deleted")
		w.Write([]byte("ok"))
	})

	// Запуск HTTP-сервера
	fmt.Println("🌐 Started to http://localhost:8080")
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}

func createTableProducts(db *sql.DB) {
	query := `
	CREATE TABLE IF NOT EXISTS products (
		id SERIAL PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		price NUMERIC(10, 2) NOT NULL
	);`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatalf("Ошибка создания таблицы: %v", err)
	}

	fmt.Println("✅ Таблица 'products' проверена/создана успешно.")
}
