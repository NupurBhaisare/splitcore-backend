package services

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/nupurbhaisare/splitcore-backend/internal/database"
	"github.com/nupurbhaisare/splitcore-backend/internal/models"
)

var ErrNotFound = errors.New("record not found")

// SettlementService handles settlement CRUD operations.
type SettlementService struct{}

// CreateSettlementInput is the input for creating a settlement.
type CreateSettlementInput struct {
	GroupID         string
	FromUserID      string
	ToUserID        string
	AmountCents     int64
	CurrencyCode    string
	Note            string
	PaymentMethod   string
	CreatedByUserID string
}

func NewSettlementService() *SettlementService {
	return &SettlementService{}
}

// Create records a new settlement between two users.
func (s *SettlementService) Create(in CreateSettlementInput) (*models.Settlement, error) {
	id := uuid.New().String()
	settledAt := time.Now()
	createdAt := settledAt

	if in.PaymentMethod == "" {
		in.PaymentMethod = "cash"
	}
	if in.CurrencyCode == "" {
		in.CurrencyCode = "USD"
	}

	_, err := database.DB.Exec(
		`INSERT INTO settlements
		(id, group_id, from_user_id, to_user_id, amount_cents, currency_code,
		 settled_at, created_by_user_id, note, payment_method, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, in.GroupID, in.FromUserID, in.ToUserID, in.AmountCents, in.CurrencyCode,
		settledAt, in.CreatedByUserID, in.Note, in.PaymentMethod, createdAt,
	)
	if err != nil {
		return nil, err
	}

	return s.GetByID(id, in.GroupID)
}

// GetByID retrieves a settlement by ID.
func (s *SettlementService) GetByID(id, groupID string) (*models.Settlement, error) {
	row := database.DB.QueryRow(
		`SELECT s.id, s.group_id, s.from_user_id, s.to_user_id, s.amount_cents,
		        s.currency_code, s.settled_at, s.created_by_user_id, s.note,
		        s.payment_method, s.created_at,
		        u1.display_name, u1.email, u1.avatar_url,
		        u2.display_name, u2.email, u2.avatar_url
		 FROM settlements s
		 LEFT JOIN users u1 ON s.from_user_id = u1.id
		 LEFT JOIN users u2 ON s.to_user_id = u2.id
		 WHERE s.id = ? AND s.group_id = ?`,
		id, groupID,
	)

	var settlement models.Settlement
	var fromName, fromEmail, fromAvatar string
	var toName, toEmail, toAvatar string

	err := row.Scan(
		&settlement.ID, &settlement.GroupID, &settlement.FromUserID, &settlement.ToUserID,
		&settlement.AmountCents, &settlement.CurrencyCode, &settlement.SettledAt,
		&settlement.CreatedByUserID, &settlement.Note, &settlement.PaymentMethod,
		&settlement.CreatedAt,
		&fromName, &fromEmail, &fromAvatar,
		&toName, &toEmail, &toAvatar,
	)
	if err != nil {
		return nil, err
	}

	if fromName != "" {
		settlement.FromUser = &models.User{
			ID:          settlement.FromUserID,
			DisplayName: fromName,
			Email:       fromEmail,
			AvatarURL:   fromAvatar,
		}
	}
	if toName != "" {
		settlement.ToUser = &models.User{
			ID:          settlement.ToUserID,
			DisplayName: toName,
			Email:       toEmail,
			AvatarURL:   toAvatar,
		}
	}

	return &settlement, nil
}

// GetByGroup returns all settlements for a group, ordered by settled_at DESC.
func (s *SettlementService) GetByGroup(groupID string) ([]models.Settlement, error) {
	rows, err := database.DB.Query(
		`SELECT s.id, s.group_id, s.from_user_id, s.to_user_id, s.amount_cents,
		        s.currency_code, s.settled_at, s.created_by_user_id, s.note,
		        s.payment_method, s.created_at,
		        u1.display_name, u1.email, u1.avatar_url,
		        u2.display_name, u2.email, u2.avatar_url
		 FROM settlements s
		 LEFT JOIN users u1 ON s.from_user_id = u1.id
		 LEFT JOIN users u2 ON s.to_user_id = u2.id
		 WHERE s.group_id = ?
		 ORDER BY s.settled_at DESC`,
		groupID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var settlements []models.Settlement
	for rows.Next() {
		var settlement models.Settlement
		var fromName, fromEmail, fromAvatar string
		var toName, toEmail, toAvatar string

		err := rows.Scan(
			&settlement.ID, &settlement.GroupID, &settlement.FromUserID, &settlement.ToUserID,
			&settlement.AmountCents, &settlement.CurrencyCode, &settlement.SettledAt,
			&settlement.CreatedByUserID, &settlement.Note, &settlement.PaymentMethod,
			&settlement.CreatedAt,
			&fromName, &fromEmail, &fromAvatar,
			&toName, &toEmail, &toAvatar,
		)
		if err != nil {
			return nil, err
		}

		if fromName != "" {
			set := settlement
			set.FromUser = &models.User{
				ID:          settlement.FromUserID,
				DisplayName: fromName,
				Email:       fromEmail,
				AvatarURL:   fromAvatar,
			}
			settlements = append(settlements, set)
		} else {
			settlements = append(settlements, settlement)
		}

		if toName != "" {
			settlements[len(settlements)-1].ToUser = &models.User{
				ID:          settlement.ToUserID,
				DisplayName: toName,
				Email:       toEmail,
				AvatarURL:   toAvatar,
			}
		}
	}

	return settlements, nil
}

// Delete removes a settlement (only the creator can delete it).
func (s *SettlementService) Delete(settlementID, groupID, userID string) error {
	result, err := database.DB.Exec(
		`DELETE FROM settlements WHERE id = ? AND group_id = ? AND created_by_user_id = ?`,
		settlementID, groupID, userID,
	)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
