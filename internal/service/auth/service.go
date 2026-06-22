package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"go-learn/models"

	"golang.org/x/crypto/bcrypt"
)

type AuthRepository interface {
	GetUserByEmail(email string) (*models.User, error)
	CreateSession(session *models.Session) error
	GetSessionByToken(token string) (*models.Session, error)
	DeleteSession(token string) error
}

type SessionCache interface {
	SetSession(ctx context.Context, token string, userID int, ttl time.Duration) error
	GetSession(ctx context.Context, token string) (int, error)
	DeleteSession(ctx context.Context, token string) error
}

type AuthService struct {
	repo   AuthRepository
	cache  SessionCache
	logger *slog.Logger
}

func NewAuthService(repo AuthRepository, cache SessionCache, logger *slog.Logger) *AuthService {
	return &AuthService{repo: repo, cache: cache, logger: logger}
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, error) {
	user, err := s.repo.GetUserByEmail(email)
	if err != nil {
		return "", fmt.Errorf("invalid credentials: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", errors.New("invalid credentials")
	}

	// Генерируем токен
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)

	session := &models.Session{
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour), // Токен живёт 24 часа
	}

	// Сохраняем в БД
	if err := s.repo.CreateSession(session); err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}

	// Кэшируем с TTL 30 секунд
	if err := s.cache.SetSession(ctx, token, user.ID, 30*time.Second); err != nil {
		s.logger.Warn("Failed to cache session", "error", err)
	}

	s.logger.Info("User logged in", "user_id", user.ID, "email", email)
	return token, nil
}

func (s *AuthService) Authenticate(ctx context.Context, token string) (int, error) {
	// Сначала проверяем кэш
	userID, err := s.cache.GetSession(ctx, token)
	if err == nil {
		return userID, nil
	}

	// Если нет в кэше, проверяем БД
	session, err := s.repo.GetSessionByToken(token)
	if err != nil {
		return 0, errors.New("invalid or expired token")
	}

	if time.Now().After(session.ExpiresAt) {
		s.repo.DeleteSession(token)
		return 0, errors.New("token expired")
	}

	// Обновляем кэш
	ttl := time.Until(session.ExpiresAt)
	if ttl > 30*time.Second {
		ttl = 30 * time.Second
	}
	s.cache.SetSession(ctx, token, session.UserID, ttl)

	return session.UserID, nil
}

func (s *AuthService) Logout(ctx context.Context, token string) error {
	if err := s.repo.DeleteSession(token); err != nil {
		return err
	}
	return s.cache.DeleteSession(ctx, token)
}
