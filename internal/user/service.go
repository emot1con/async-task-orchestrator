package user

import (
	"database/sql"
	"errors"
	"task_handler/internal/auth"

	"github.com/sirupsen/logrus"
)

type UserService struct {
	repo UserRepositoryInterface
	db   *sql.DB
}

type UserServiceInterface interface {
	CreateUser(username, password string) (int, error)
	LoginUser(username, password, jwtSecret string) (*auth.TokenPair, error)
	GetUserByID(id int) (*User, error)
	GetUserByUsername(username string) (*User, error)
}

func NewUserService(repo UserRepositoryInterface, db *sql.DB) UserServiceInterface {
	return &UserService{
		repo: repo,
		db:   db,
	}
}

// CreateUser creates a new user with hashed password
func (s *UserService) CreateUser(username, password string) (int, error) {
	// Get user by username
	userData, err := s.repo.GetByUsername(s.db, username)
	if err == nil && userData != nil {
		return 0, errors.New("username already exists")
	}

	// Hash password
	hashedPassword, err := auth.GeneratePasswordHash(password)
	if err != nil {
		return 0, errors.New("failed to hash password")
	}

	user := &User{
		Username: username,
		Password: hashedPassword,
	}

	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		logrus.WithError(err).Error("Failed to begin transaction")
		return 0, err
	}
	defer tx.Rollback()

	// Create user
	id, err := s.repo.Create(tx, user)
	if err != nil {
		return 0, err
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		logrus.WithError(err).Error("Failed to commit transaction")
		return 0, err
	}

	return id, nil
}

// ValidateCredentials validates username and password, returns user if valid
func (s *UserService) LoginUser(username, password, jwtSecret string) (*auth.TokenPair, error) {
	// Get user by username
	user, err := s.repo.GetByUsername(s.db, username)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	if err := auth.ComparePasswordHash([]byte(user.Password), password); err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Generate token pair
	tokens, err := auth.GenerateTokenPair(user.ID, jwtSecret)
	if err != nil {
		return nil, err
	}

	return tokens, nil
}

// GetUserByID retrieves user by ID
func (s *UserService) GetUserByID(id int) (*User, error) {
	return s.repo.GetByID(s.db, id)
}

// GetUserByUsername retrieves user by username
func (s *UserService) GetUserByUsername(username string) (*User, error) {
	return s.repo.GetByUsername(s.db, username)
}
