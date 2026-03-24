package services

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/nupurbhaisare/splitcore-backend/internal/database"
	"github.com/nupurbhaisare/splitcore-backend/internal/models"
	"github.com/nupurbhaisare/splitcore-backend/pkg/utils"
)

type UserService struct{}

func NewUserService() *UserService {
	return &UserService{}
}

func (s *UserService) Create(email, password, displayName string) (*models.User, error) {
	// Check if user already exists
	var existingID string
	err := database.DB.QueryRow("SELECT id FROM users WHERE email = ? AND deleted_at IS NULL", email).Scan(&existingID)
	if err == nil {
		return nil, errors.New("user with this email already exists")
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	hash, err := utils.HashPassword(password)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		ID:           uuid.New().String(),
		Email:        email,
		PasswordHash: hash,
		DisplayName:  displayName,
		AvatarURL:    "",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if displayName == "" {
		// Extract name from email
		parts := splitEmail(email)
		user.DisplayName = parts
	}

	_, err = database.DB.Exec(
		`INSERT INTO users (id, email, password_hash, display_name, avatar_url, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		user.ID, user.Email, user.PasswordHash, user.DisplayName, user.AvatarURL, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) Authenticate(email, password string) (*models.User, error) {
	user := &models.User{}
	err := database.DB.QueryRow(
		`SELECT id, email, password_hash, display_name, avatar_url, created_at, updated_at
		 FROM users WHERE email = ? AND deleted_at IS NULL`,
		email,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.DisplayName, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("invalid email or password")
	}
	if err != nil {
		return nil, err
	}

	if !utils.CheckPassword(password, user.PasswordHash) {
		return nil, errors.New("invalid email or password")
	}

	return user, nil
}

func (s *UserService) GetByID(id string) (*models.User, error) {
	user := &models.User{}
	err := database.DB.QueryRow(
		`SELECT id, email, display_name, avatar_url, created_at, updated_at
		 FROM users WHERE id = ? AND deleted_at IS NULL`,
		id,
	).Scan(&user.ID, &user.Email, &user.DisplayName, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("user not found")
	}
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) Update(id, displayName, avatarURL string) (*models.User, error) {
	_, err := database.DB.Exec(
		`UPDATE users SET display_name = ?, avatar_url = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`,
		displayName, avatarURL, time.Now(), id,
	)
	if err != nil {
		return nil, err
	}

	return s.GetByID(id)
}

func (s *UserService) GetByEmail(email string) (*models.User, error) {
	user := &models.User{}
	err := database.DB.QueryRow(
		`SELECT id, email, display_name, avatar_url, created_at, updated_at
		 FROM users WHERE email = ? AND deleted_at IS NULL`,
		email,
	).Scan(&user.ID, &user.Email, &user.DisplayName, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("user not found")
	}
	if err != nil {
		return nil, err
	}

	return user, nil
}

func splitEmail(email string) string {
	// Simple: just return the part before @
	for i := 0; i < len(email); i++ {
		if email[i] == '@' {
			return email[:i]
		}
	}
	return email
}
