package user

import (
	"database/sql"
	"fmt"
	"log/slog"
)

type UserRepository struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewUserRepository(db *sql.DB, logger *slog.Logger) *UserRepository {
	return &UserRepository{
		db:     db,
		logger: logger,
	}
}

func (r *UserRepository) CreateUser(name, email, passwordHash string) (int, error) {
	r.logger.Debug("[UserRepository] Creating user", "name", name, "email", email)

	var id int
	err := r.db.QueryRow(
		"INSERT INTO users (name, email, password_hash) VALUES ($1, $2, $3) RETURNING id", name, email, passwordHash,
	).Scan(&id)

	if err != nil {
		r.logger.Error("[UserRepository] Failed to create user", "name", name, "email", email, "err", err)
		return 0, fmt.Errorf("failed to create user: %w", err)
	}

	r.logger.Info("[UserRepository] User created successfully", "id", id, "name", name, "email", email)
	return id, nil
}
