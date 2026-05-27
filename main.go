package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	_ "github.com/lib/pq" // Драйвер для Postgres
)

func main() {
	// === ПОДКЛЮЧЕНИЕ К БАЗЕ ДАННЫХ ===
	// Строка подключения к Postgres
	connStr := "host=localhost port=5432 user=user password=password dbname=products_db sslmode=disable"

	// Подключение к БД
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Ошибка настройки БД: %v", err)
	}
	defer db.Close() // Закроем соединение когда программа завершится

	// Проверка соединения
	err = db.Ping()
	if err != nil {
		log.Fatalf("Не удалось подключиться к Postgres: %v", err)
	}
	fmt.Println("✅ Успешно подключено к Postgres!!!")

	// === СОЗДАЕМ ТАБЛИЦУ ТОВАРОВ ===
	query := `
	CREATE TABLE IF NOT EXISTS products (
		id SERIAL PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		price NUMERIC(10, 2) NOT NULL
	);`

	_, err = db.Exec(query)
	if err != nil {
		log.Fatalf("Ошибка создания таблицы: %v", err)
	}
	fmt.Println("✅ Таблица 'products' проверена/создана успешно.")

	// Настройка HTTP-эндпоинта
	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Привет! Сервер работает, базы активны."))
	})

	// Запуск HTTP-сервера
	fmt.Println("Сервер запущен на http://localhost:8080/hello")
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}
