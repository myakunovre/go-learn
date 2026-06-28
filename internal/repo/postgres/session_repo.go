package postgres

import (
	"database/sql"
	"fmt"
	"go-learn/models"
	"log/slog"
)

type AuthRepository struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewAuthRepository(db *sql.DB, logger *slog.Logger) *AuthRepository {
	return &AuthRepository{db: db, logger: logger}
}

func (r *AuthRepository) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	err := r.db.QueryRow(
		"SELECT id, name, email, password_hash FROM users WHERE email = $1",
		email,
	).Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

func (r *AuthRepository) CreateSession(session *models.Session) error {
	return r.db.QueryRow(
		"INSERT INTO sessions (user_id, token, expires_at) VALUES ($1, $2, $3) RETURNING id",
		session.UserID, session.Token, session.ExpiresAt,
	).Scan(&session.ID)
}

func (r *AuthRepository) GetSessionByToken(token string) (*models.Session, error) {
	var session models.Session
	err := r.db.QueryRow(
		"SELECT id, user_id, token, expires_at, created_at FROM sessions WHERE token = $1",
		token,
	).Scan(&session.ID, &session.UserID, &session.Token, &session.ExpiresAt, &session.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	return &session, nil
}

func (r *AuthRepository) DeleteSession(token string) error {
	_, err := r.db.Exec("DELETE FROM sessions WHERE token = $1", token)
	return err
}
