package services

import (
	"sort"

	"github.com/nupurbhaisare/splitcore-backend/internal/database"
	"github.com/nupurbhaisare/splitcore-backend/internal/models"
)

// DebtService handles debt simplification.
type DebtService struct{}

// DebtService entry — wraps BalanceService for raw balances.
type rawBalanceEntry struct {
	userID      string
	displayName string
	avatarURL   string
	netCents    int64
	status      string
}

func NewDebtService() *DebtService {
	return &DebtService{}
}

// GetSimplifiedDebts returns the simplified debt graph for a group.
// It computes raw balances, subtracts past settlements, then nets creditors vs debtors.
func (s *DebtService) GetSimplifiedDebts(groupID string) (*models.DebtResponse, error) {
	groupService := NewGroupService()
	group, err := groupService.GetByID(groupID)
	if err != nil {
		return nil, err
	}

	members, err := groupService.GetMembers(groupID)
	if err != nil {
		return nil, err
	}

	// Build balance map from raw balances
	balanceMap := make(map[string]int64)
	userNames := make(map[string]string)
	userAvatars := make(map[string]string)

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

	// Add paid amounts
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

	// Subtract split amounts
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

	// Subtract settled amounts
	settlementRows, err := database.DB.Query(
		`SELECT from_user_id, to_user_id, amount_cents FROM settlements
		 WHERE group_id = ?`,
		groupID,
	)
	if err != nil {
		return nil, err
	}
	defer settlementRows.Close()

	for settlementRows.Next() {
		var fromID, toID string
		var amountCents int64
		if err := settlementRows.Scan(&fromID, &toID, &amountCents); err != nil {
			return nil, err
		}
		// from_user paid the amount, so their net owed decreases
		balanceMap[fromID] -= amountCents
		// to_user received the amount, so their net owed increases
		balanceMap[toID] += amountCents
	}

	// Build balances slice
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

	// Netting algorithm: produce minimum transactions
	debts := s.netBalances(balanceMap)

	return &models.DebtResponse{
		GroupID:      groupID,
		CurrencyCode: group.CurrencyCode,
		Balances:     balances,
		Debts:        debts,
	}, nil
}

// netBalances takes a map of userID -> net balance and returns minimum settlement transactions.
// Positive = owed money, Negative = owes money.
func (s *DebtService) netBalances(balanceMap map[string]int64) []models.SimplifiedDebt {
	// Separate into creditors (positive) and debtors (negative)
	type entry struct {
		userID   string
		amount   int64
	}

	var creditors []entry
	var debtors []entry

	for userID, balance := range balanceMap {
		if balance > 0 {
			creditors = append(creditors, entry{userID: userID, amount: balance})
		} else if balance < 0 {
			debtors = append(debtors, entry{userID: userID, amount: -balance}) // store as positive
		}
	}

	// Sort descending by amount for both
	sort.Slice(creditors, func(i, j int) bool { return creditors[i].amount > creditors[j].amount })
	sort.Slice(debtors, func(i, j int) bool { return debtors[i].amount > debtors[j].amount })

	var debts []models.SimplifiedDebt

	i, j := 0, 0
	for i < len(debtors) && j < len(creditors) {
		debtor := &debtors[i]
		creditor := &creditors[j]

		if debtor.amount == 0 {
			i++
			continue
		}
		if creditor.amount == 0 {
			j++
			continue
		}

		transfer := debtor.amount
		if creditor.amount < transfer {
			transfer = creditor.amount
		}

		if transfer > 0 {
			debts = append(debts, models.SimplifiedDebt{
				FromUserID:  debtor.userID,
				ToUserID:    creditor.userID,
				AmountCents: transfer,
			})
		}

		debtor.amount -= transfer
		creditor.amount -= transfer

		if debtor.amount == 0 {
			i++
		}
		if creditor.amount == 0 {
			j++
		}
	}

	return debts
}
