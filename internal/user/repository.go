package user

import (
	"database/sql"
	"errors"

	"github.com/sirupsen/logrus"
)

type UserRepository struct{}

type UserRepositoryInterface interface {
	Create(tx *sql.Tx, user *User) (int, error)
	GetByID(db *sql.DB, id int) (*User, error)
	GetByUsername(db *sql.DB, username string) (*User, error)
	UpdatePassword(tx *sql.Tx, id int, hashedPassword string) error
}

func NewUserRepository() UserRepositoryInterface {
	return &UserRepository{}
}

// Create creates a new user in the database
func (r *UserRepository) Create(
	tx *sql.Tx,
	user *User,
) (int, error) {
	query := `
		INSERT INTO users (
			username, password, created_at
		)
		VALUES ($1, $2, NOW())
		RETURNING id
	`

	var id int
	err := tx.QueryRow(
		query,
		user.Username,
		user.Password,
	).Scan(&id)

	if err != nil {
		logrus.WithError(err).Error("Failed to create user")
		return 0, err
	}

	logrus.WithFields(logrus.Fields{
		"user_id":  id,
		"username": user.Username,
	}).Info("User created successfully")

	return id, nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(db *sql.DB, id int) (*User, error) {
	query := `
		SELECT id, username, password, created_at
		FROM users
		WHERE id = $1
	`

	user := &User{}
	err := db.QueryRow(query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Password,
		&user.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			logrus.WithField("user_id", id).Warn("User not found")
			return nil, errors.New("user not found")
		}
		logrus.WithError(err).Error("Failed to get user by ID")
		return nil, err
	}

	return user, nil
}

// GetByUsername retrieves a user by username
func (r *UserRepository) GetByUsername(db *sql.DB, username string) (*User, error) {
	query := `
		SELECT id, username, password, created_at
		FROM users
		WHERE username = $1
	`

	user := &User{}
	err := db.QueryRow(query, username).Scan(
		&user.ID,
		&user.Username,
		&user.Password,
		&user.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			logrus.WithField("username", username).Warn("User not found")
			return nil, errors.New("user not found")
		}
		logrus.WithError(err).Error("Failed to get user by username")
		return nil, err
	}

	return user, nil
}

// UpdatePassword updates user's password
func (r *UserRepository) UpdatePassword(tx *sql.Tx, id int, hashedPassword string) error {
	query := `
		UPDATE users
		SET password = $1
		WHERE id = $2
	`

	result, err := tx.Exec(query, hashedPassword, id)
	if err != nil {
		logrus.WithError(err).Error("Failed to update password")
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return errors.New("user not found")
	}

	logrus.WithField("user_id", id).Info("Password updated successfully")
	return nil
}
