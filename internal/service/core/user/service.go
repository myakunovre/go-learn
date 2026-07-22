package user

import (
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type UserRepository interface {
	CreateUser(name, email, passwordHash string) (int, error)
}

type UserService struct {
	repo   UserRepository
	logger *slog.Logger
}

var emailRe = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func isValidEmailSimple(email string) bool {
	email = strings.TrimSpace(email)
	return emailRe.MatchString(email)
}

func NewUserService(repo UserRepository, logger *slog.Logger) *UserService {
	return &UserService{
		repo:   repo,
		logger: logger,
	}
}

func (s *UserService) Create(name, email, password string) (int, error) {
	// Проверка email на валидность
	if !isValidEmailSimple(email) {
		s.logger.Error("[UserService] Email is not valid", "email", email)
		return 0, errors.New("invalid email")
	}

	passwordBytes := []byte(password)
	// bcrypt.GenerateFromPassword сам создаст соль и сделает хэш
	passwordHash, err := bcrypt.GenerateFromPassword(passwordBytes, bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("[UserService] Error generate password hash", "error", err)
		return 0, err
	}

	id, err := s.repo.CreateUser(name, email, string(passwordHash))
	if err != nil {
		s.logger.Error("[UserService] Error of creating user", "name", name, "email", email, "error", err)
		return 0, fmt.Errorf("user creation failed: %w", err)
	}

	s.logger.Info("[UserService] ✅ User created successfully", "id", id, "name", name, "email", email)
	return id, nil
}
