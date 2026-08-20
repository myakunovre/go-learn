package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"go-learn/models"
	"log/slog"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type AuthRepository interface {
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	CreateSession(ctx context.Context, session *models.Session) error
	GetSessionByToken(ctx context.Context, token string) (*models.Session, error)
	DeleteSession(ctx context.Context, token string) error
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
	dbCtx, dbCancel := context.WithTimeout(ctx, 5*time.Second)
	defer dbCancel()

	user, err := s.repo.GetUserByEmail(dbCtx, email)
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
		UserID:    int(user.ID),
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour), // Токен живёт 24 часа
	}

	// Сохраняем в БД
	if err := s.repo.CreateSession(dbCtx, session); err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}

	cacheCtx, cacheCancel := context.WithTimeout(ctx, 2*time.Second)
	defer cacheCancel()

	// Кэшируем с TTL 30 секунд
	if err := s.cache.SetSession(cacheCtx, token, int(user.ID), 30*time.Second); err != nil {
		s.logger.Warn("Failed to redis session", "error", err)
	}

	s.logger.Info("User logged in", "user_id", user.ID, "email", email)
	return token, nil
}

func (s *AuthService) Authenticate(ctx context.Context, token string) (int, error) {
	// Сначала проверяем кэш
	cacheCtx, cacheCancel := context.WithTimeout(ctx, 2*time.Second)
	defer cacheCancel()

	userID, err := s.cache.GetSession(cacheCtx, token)
	if err == nil {
		return userID, nil
	}

	// Если нет в кэше, проверяем БД
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	session, err := s.repo.GetSessionByToken(dbCtx, token)
	if err != nil {
		return 0, errors.New("invalid or expired token")
	}

	if time.Now().After(session.ExpiresAt) {
		s.repo.DeleteSession(dbCtx, token)
		return 0, errors.New("token expired")
	}

	// Обновляем кэш
	ttl := time.Until(session.ExpiresAt)
	if ttl > 30*time.Second {
		ttl = 30 * time.Second
	}

	cacheCtx2, cacheCancel2 := context.WithTimeout(ctx, 2*time.Second)
	defer cacheCancel2()

	s.cache.SetSession(cacheCtx2, token, session.UserID, ttl)

	return session.UserID, nil
}

func (s *AuthService) Logout(ctx context.Context, token string) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := s.repo.DeleteSession(dbCtx, token); err != nil {
		return err
	}

	cacheCtx, cacheCancel := context.WithTimeout(ctx, 2*time.Second)
	defer cacheCancel()

	return s.cache.DeleteSession(cacheCtx, token)
}
