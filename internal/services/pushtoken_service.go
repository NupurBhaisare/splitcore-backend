package services

import (
	"time"

	"github.com/google/uuid"
	"github.com/splitcore/backend/internal/database"
	"github.com/splitcore/backend/internal/models"
)

// PushTokenService handles device push token management.
type PushTokenService struct{}

func NewPushTokenService() *PushTokenService {
	return &PushTokenService{}
}

// Register registers or updates a push token for a user.
func (s *PushTokenService) Register(userID, deviceToken, platform string) (*models.PushToken, error) {
	now := time.Now()

	// Upsert: update existing or insert new
	id := uuid.New().String()
	_, err := database.DB.Exec(
		`INSERT INTO push_tokens (id, user_id, device_token, platform, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(user_id, device_token) DO UPDATE SET
			updated_at = excluded.updated_at,
			deleted_at = NULL`,
		id, userID, deviceToken, platform, now, now,
	)
	if err != nil {
		return nil, err
	}

	return s.GetByToken(deviceToken)
}

// GetByToken retrieves a push token record by device token.
func (s *PushTokenService) GetByToken(deviceToken string) (*models.PushToken, error) {
	row := database.DB.QueryRow(
		`SELECT id, user_id, device_token, platform, created_at, updated_at
		 FROM push_tokens WHERE device_token = ? AND deleted_at IS NULL`,
		deviceToken,
	)

	var token models.PushToken
	err := row.Scan(&token.ID, &token.UserID, &token.DeviceToken, &token.Platform, &token.CreatedAt, &token.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &token, nil
}

// Unregister removes (soft-deletes) a push token.
func (s *PushTokenService) Unregister(deviceToken, userID string) error {
	now := time.Now()
	_, err := database.DB.Exec(
		`UPDATE push_tokens SET deleted_at = ? WHERE device_token = ? AND user_id = ?`,
		now, deviceToken, userID,
	)
	return err
}

// GetByUser returns all active push tokens for a user.
func (s *PushTokenService) GetByUser(userID string) ([]models.PushToken, error) {
	rows, err := database.DB.Query(
		`SELECT id, user_id, device_token, platform, created_at, updated_at
		 FROM push_tokens WHERE user_id = ? AND deleted_at IS NULL`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []models.PushToken
	for rows.Next() {
		var token models.PushToken
		err := rows.Scan(&token.ID, &token.UserID, &token.DeviceToken, &token.Platform, &token.CreatedAt, &token.UpdatedAt)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}

	return tokens, nil
}
