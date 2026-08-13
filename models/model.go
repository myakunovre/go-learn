package models

import "time"

// Product - сущность товара
type Product struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Price  int64  `json:"price"`
	Amount int64  `json:"amount"`
}

// User - сущность товара
type User struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Email        string `json:"email"`
	PasswordHash string `json:"passwordHash"`
}

// Session - сущность сессии
type Session struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// Order - сущность товара
type Order struct {
	ID          int    `json:"id"`
	Description string `json:"description"`
	UserId      int    `json:"userId"`
}
