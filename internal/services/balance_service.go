package services

import (
	"github.com/splitcore/backend/internal/database"
	"github.com/splitcore/backend/internal/models"
)

type BalanceService struct{}

func NewBalanceService() *BalanceService {
	return &BalanceService{}
}

type BalanceResponse struct {
	GroupID     string           `json:"group_id"`
	CurrencyCode string          `json:"currency_code"`
	Balances    []models.Balance `json:"balances"`
}

func (s *BalanceService) GetGroupBalances(groupID string) (*BalanceResponse, error) {
	groupService := NewGroupService()
	group, err := groupService.GetByID(groupID)
	if err != nil {
		return nil, err
	}

	members, err := groupService.GetMembers(groupID)
	if err != nil {
		return nil, err
	}

	// Compute net balance per user
	// net_balance[user] = Σ(paid_by_user) - Σ(share_amount_for_user)
	balanceMap := make(map[string]int64)
	userNames := make(map[string]string)
	userAvatars := make(map[string]string)

	// Initialize all members with 0
	for _, m := range members {
		balanceMap[m.UserID] = 0
		if m.User != nil {
			userNames[m.UserID] = m.User.DisplayName
			userAvatars[m.UserID] = m.User.AvatarURL
		}
		if m.Nickname != "" {
			userNames[m.UserID] = m.Nickname
		}
	}

	// Get all paid amounts
	rows, err := database.DB.Query(
		`SELECT paid_by_user_id, amount_cents FROM expenses
		 WHERE group_id = ? AND deleted_at IS NULL`,
		groupID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var payerID string
		var amountCents int64
		if err := rows.Scan(&payerID, &amountCents); err != nil {
			return nil, err
		}
		balanceMap[payerID] += amountCents
	}

	// Get all split amounts
	splitRows, err := database.DB.Query(
		`SELECT es.user_id, es.share_amount_cents
		 FROM expense_splits es
		 INNER JOIN expenses e ON es.expense_id = e.id
		 WHERE e.group_id = ? AND e.deleted_at IS NULL`,
		groupID,
	)
	if err != nil {
		return nil, err
	}
	defer splitRows.Close()

	for splitRows.Next() {
		var userID string
		var shareCents int64
		if err := splitRows.Scan(&userID, &shareCents); err != nil {
			return nil, err
		}
		balanceMap[userID] -= shareCents
	}

	// Build response
	var balances []models.Balance
	for _, m := range members {
		net := balanceMap[m.UserID]
		status := "settled"
		if net > 0 {
			status = "owed"
		} else if net < 0 {
			status = "owes"
		}

		balances = append(balances, models.Balance{
			UserID:      m.UserID,
			DisplayName: userNames[m.UserID],
			AvatarURL:   userAvatars[m.UserID],
			NetCents:    net,
			Status:      status,
		})
	}

	return &BalanceResponse{
		GroupID:      groupID,
		CurrencyCode: group.CurrencyCode,
		Balances:     balances,
	}, nil
}

func (s *BalanceService) GetUserGroupBalance(groupID, userID string) (*models.Balance, error) {
	resp, err := s.GetGroupBalances(groupID)
	if err != nil {
		return nil, err
	}

	for _, b := range resp.Balances {
		if b.UserID == userID {
			return &b, nil
		}
	}

	return nil, nil
}
