package services

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/splitcore/backend/internal/database"
	"github.com/splitcore/backend/internal/models"
)

type ExpenseService struct{}

func NewExpenseService() *ExpenseService {
	return &ExpenseService{}
}

type CreateExpenseInput struct {
	GroupID      string
	PaidByUserID string
	Title        string
	Description  string
	AmountCents  int64
	CurrencyCode string
	Category     string
	ExpenseDate  time.Time
	SplitUserIDs []string // Users to split between (if empty, split among all group members)
	SplitType   string    // equal, percentage, exact, shares (default: equal)
	Splits       []SplitInput // Individual split data for percentage/exact/shares
}

// SplitInput represents a single user's split with optional percentage or share count.
type SplitInput struct {
	UserID      string
	ShareCents  int64
	Percentage  float64 // for percentage splits
	ShareCount  int     // for shares splits
}

func (s *ExpenseService) Create(in CreateExpenseInput) (*models.Expense, error) {
	groupService := NewGroupService()

	// Verify group exists and user is a member
	if _, err := groupService.GetByID(in.GroupID); err != nil {
		return nil, err
	}
	if !groupService.IsMember(in.GroupID, in.PaidByUserID) {
		return nil, errors.New("user is not a member of this group")
	}

	if in.CurrencyCode == "" {
		in.CurrencyCode = "USD"
	}
	if in.Category == "" {
		in.Category = "other"
	}
	if in.ExpenseDate.IsZero() {
		in.ExpenseDate = time.Now()
	}

	expense := &models.Expense{
		ID:           uuid.New().String(),
		GroupID:      in.GroupID,
		PaidByUserID: in.PaidByUserID,
		Title:        in.Title,
		Description:  in.Description,
		AmountCents:  in.AmountCents,
		CurrencyCode: in.CurrencyCode,
		Category:     in.Category,
		ExpenseDate:  in.ExpenseDate,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	tx, err := database.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`INSERT INTO expenses (id, group_id, paid_by_user_id, title, description, amount_cents, currency_code, category, expense_date, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		expense.ID, expense.GroupID, expense.PaidByUserID, expense.Title, expense.Description,
		expense.AmountCents, expense.CurrencyCode, expense.Category, expense.ExpenseDate,
		expense.CreatedAt, expense.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Determine split users
	splitUsers := in.SplitUserIDs
	if len(splitUsers) == 0 && len(in.Splits) == 0 {
		// Split among all group members
		members, err := groupService.GetMembers(in.GroupID)
		if err != nil {
			return nil, err
		}
		for _, m := range members {
			splitUsers = append(splitUsers, m.UserID)
		}
	}

	// Determine split type
	splitType := in.SplitType
	if splitType == "" {
		splitType = "equal"
	}

	// Handle rich splits (percentage, exact, shares)
	if len(in.Splits) > 0 {
		for _, sp := range in.Splits {
			shareCents := sp.ShareCents
			percentage := sp.Percentage
			shareCount := sp.ShareCount

			split := models.ExpenseSplit{
				ID:               uuid.New().String(),
				ExpenseID:        expense.ID,
				UserID:           sp.UserID,
				ShareAmountCents: shareCents,
				SplitType:        splitType,
				Percentage:       percentage,
				ShareCount:       shareCount,
				CreatedAt:        time.Now(),
			}

			_, err = tx.Exec(
				`INSERT INTO expense_splits (id, expense_id, user_id, share_amount_cents, split_type, created_at)
				 VALUES (?, ?, ?, ?, ?, ?)`,
				split.ID, split.ExpenseID, split.UserID, split.ShareAmountCents, split.SplitType, split.CreatedAt,
			)
			if err != nil {
				return nil, err
			}
		}
	} else {
		// Create equal splits (default behavior)
		splitCount := len(splitUsers)
		if splitCount == 0 {
			return nil, errors.New("no users to split with")
		}

		splitAmount := expense.AmountCents / int64(splitCount)
		remainder := expense.AmountCents % int64(splitCount)

		for i, userID := range splitUsers {
			shareAmount := splitAmount
			// Distribute remainder to first users
			if i < int(remainder) {
				shareAmount++
			}

			split := models.ExpenseSplit{
				ID:               uuid.New().String(),
				ExpenseID:        expense.ID,
				UserID:           userID,
				ShareAmountCents: shareAmount,
				SplitType:        splitType,
				CreatedAt:        time.Now(),
			}

			_, err = tx.Exec(
				`INSERT INTO expense_splits (id, expense_id, user_id, share_amount_cents, split_type, created_at)
				 VALUES (?, ?, ?, ?, ?, ?)`,
				split.ID, split.ExpenseID, split.UserID, split.ShareAmountCents, split.SplitType, split.CreatedAt,
			)
			if err != nil {
				return nil, err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// Reload with splits
	return s.GetByID(expense.ID, in.GroupID)
}

func (s *ExpenseService) GetByID(id, groupID string) (*models.Expense, error) {
	expense := &models.Expense{}
	var deletedAt sql.NullTime
	err := database.DB.QueryRow(
		`SELECT id, group_id, paid_by_user_id, title, description, amount_cents, currency_code,
		        category, expense_date, created_at, updated_at, deleted_at
		 FROM expenses WHERE id = ? AND deleted_at IS NULL`,
		id,
	).Scan(&expense.ID, &expense.GroupID, &expense.PaidByUserID, &expense.Title, &expense.Description,
		&expense.AmountCents, &expense.CurrencyCode, &expense.Category, &expense.ExpenseDate,
		&expense.CreatedAt, &expense.UpdatedAt, &deletedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("expense not found")
	}
	if err != nil {
		return nil, err
	}
	if deletedAt.Valid {
		expense.DeletedAt = &deletedAt.Time
	}

	// Load payer info
	userService := NewUserService()
	payer, err := userService.GetByID(expense.PaidByUserID)
	if err == nil {
		expense.PaidBy = payer
	}

	// Load splits
	splits, err := s.GetSplits(expense.ID)
	if err == nil {
		expense.Splits = splits
	}

	return expense, nil
}

func (s *ExpenseService) GetSplits(expenseID string) ([]models.ExpenseSplit, error) {
	rows, err := database.DB.Query(
		`SELECT es.id, es.expense_id, es.user_id, es.share_amount_cents, es.split_type, es.created_at,
		        COALESCE(es.percentage, 0), COALESCE(es.share_count, 0),
		        u.id, u.email, u.display_name, u.avatar_url, u.created_at, u.updated_at
		 FROM expense_splits es
		 INNER JOIN users u ON es.user_id = u.id
		 WHERE es.expense_id = ?`,
		expenseID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var splits []models.ExpenseSplit
	for rows.Next() {
		var sp models.ExpenseSplit
		var u models.User
		if err := rows.Scan(&sp.ID, &sp.ExpenseID, &sp.UserID, &sp.ShareAmountCents, &sp.SplitType, &sp.CreatedAt,
			&sp.Percentage, &sp.ShareCount,
			&u.ID, &u.Email, &u.DisplayName, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		sp.User = &u
		splits = append(splits, sp)
	}

	return splits, nil
}

func (s *ExpenseService) GetByGroup(groupID string) ([]models.Expense, error) {
	rows, err := database.DB.Query(
		`SELECT e.id, e.group_id, e.paid_by_user_id, e.title, e.description, e.amount_cents,
		        e.currency_code, e.category, e.expense_date, e.created_at, e.updated_at,
		        u.id, u.email, u.display_name, u.avatar_url, u.created_at, u.updated_at
		 FROM expenses e
		 INNER JOIN users u ON e.paid_by_user_id = u.id
		 WHERE e.group_id = ? AND e.deleted_at IS NULL
		 ORDER BY e.expense_date DESC, e.created_at DESC`,
		groupID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var expenses []models.Expense
	for rows.Next() {
		var e models.Expense
		var u models.User
		if err := rows.Scan(&e.ID, &e.GroupID, &e.PaidByUserID, &e.Title, &e.Description, &e.AmountCents,
			&e.CurrencyCode, &e.Category, &e.ExpenseDate, &e.CreatedAt, &e.UpdatedAt,
			&u.ID, &u.Email, &u.DisplayName, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		e.PaidBy = &u
		// Load splits for each expense
		splits, _ := s.GetSplits(e.ID)
		e.Splits = splits
		expenses = append(expenses, e)
	}

	return expenses, nil
}

func (s *ExpenseService) Update(id, groupID, userID, title, description, category string, amountCents int64, splitUserIDs []string) (*models.Expense, error) {
	expense, err := s.GetByID(id, groupID)
	if err != nil {
		return nil, err
	}

	// Only the payer can update
	if expense.PaidByUserID != userID {
		return nil, errors.New("only the payer can update this expense")
	}

	if title != "" {
		expense.Title = title
	}
	if description != "" {
		expense.Description = description
	}
	if category != "" {
		expense.Category = category
	}
	if amountCents > 0 {
		expense.AmountCents = amountCents
	}
	expense.UpdatedAt = time.Now()

	tx, err := database.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`UPDATE expenses SET title = ?, description = ?, category = ?, amount_cents = ?, updated_at = ?
		 WHERE id = ? AND deleted_at IS NULL`,
		expense.Title, expense.Description, expense.Category, expense.AmountCents, expense.UpdatedAt, id,
	)
	if err != nil {
		return nil, err
	}

	// If split users changed, recreate splits
	if len(splitUserIDs) > 0 {
		// Delete existing splits
		_, err = tx.Exec(`DELETE FROM expense_splits WHERE expense_id = ?`, id)
		if err != nil {
			return nil, err
		}

		// Create new splits
		splitCount := len(splitUserIDs)
		splitAmount := expense.AmountCents / int64(splitCount)
		remainder := expense.AmountCents % int64(splitCount)

		for i, splitUserID := range splitUserIDs {
			shareAmount := splitAmount
			if i < int(remainder) {
				shareAmount++
			}

			_, err = tx.Exec(
				`INSERT INTO expense_splits (id, expense_id, user_id, share_amount_cents, split_type, created_at)
				 VALUES (?, ?, ?, ?, ?, ?)`,
				uuid.New().String(), id, splitUserID, shareAmount, "equal", time.Now(),
			)
			if err != nil {
				return nil, err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return s.GetByID(id, groupID)
}

func (s *ExpenseService) Delete(id, groupID, userID string) error {
	expense, err := s.GetByID(id, groupID)
	if err != nil {
		return err
	}

	groupService := NewGroupService()

	// Only payer or group owner can delete
	if expense.PaidByUserID != userID && !groupService.IsOwner(groupID, userID) {
		return errors.New("only the payer or group owner can delete this expense")
	}

	_, err = database.DB.Exec(
		`UPDATE expenses SET deleted_at = ?, updated_at = ? WHERE id = ?`,
		time.Now(), time.Now(), id,
	)
	return err
}
