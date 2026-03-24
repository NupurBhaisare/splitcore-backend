package services

import (
	"strconv"
	"strings"

	"github.com/splitcore/backend/internal/database"
	"github.com/splitcore/backend/internal/models"
)

type SearchService struct{}

func NewSearchService() *SearchService {
	return &SearchService{}
}

// SearchExpensesInput defines search parameters.
type SearchExpensesInput struct {
	GroupID   string
	Query     string
	Category  string
	MinAmount int64
	MaxAmount int64
	StartDate string
	EndDate   string
	PayerID   string
	Page      int
	PerPage   int
}

// SearchExpenses searches expenses within a group.
func (s *SearchService) SearchExpenses(in SearchExpensesInput) (*models.SearchExpenseResult, error) {
	if in.Page < 1 {
		in.Page = 1
	}
	if in.PerPage < 1 || in.PerPage > 100 {
		in.PerPage = 20
	}
	offset := (in.Page - 1) * in.PerPage

	// Build WHERE clause
	conditions := []string{"e.group_id = ?", "e.deleted_at IS NULL"}
	args := []interface{}{in.GroupID}

	if in.Query != "" {
		conditions = append(conditions, "(e.title LIKE ? OR e.description LIKE ?)")
		searchTerm := "%" + in.Query + "%"
		args = append(args, searchTerm, searchTerm)
	}

	if in.Category != "" {
		conditions = append(conditions, "e.category = ?")
		args = append(args, in.Category)
	}

	if in.MinAmount > 0 {
		conditions = append(conditions, "e.amount_cents >= ?")
		args = append(args, in.MinAmount)
	}

	if in.MaxAmount > 0 {
		conditions = append(conditions, "e.amount_cents <= ?")
		args = append(args, in.MaxAmount)
	}

	if in.StartDate != "" {
		conditions = append(conditions, "e.expense_date >= ?")
		args = append(args, in.StartDate)
	}

	if in.EndDate != "" {
		conditions = append(conditions, "e.expense_date <= ?")
		args = append(args, in.EndDate)
	}

	if in.PayerID != "" {
		conditions = append(conditions, "e.paid_by_user_id = ?")
		args = append(args, in.PayerID)
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count total
	var total int
	countQuery := "SELECT COUNT(*) FROM expenses e WHERE " + whereClause
	countArgs := args
	database.DB.QueryRow(countQuery, countArgs...).Scan(&total)

	// Fetch results
	query := `SELECT e.id, e.group_id, e.paid_by_user_id, e.title, e.description, e.amount_cents,
	                 e.currency_code, e.category, e.expense_date, e.created_at, e.updated_at,
	                 u.id, u.email, u.display_name, u.avatar_url, u.created_at, u.updated_at
	          FROM expenses e
	          INNER JOIN users u ON e.paid_by_user_id = u.id
	          WHERE ` + whereClause + `
	          ORDER BY e.expense_date DESC, e.created_at DESC
	          LIMIT ? OFFSET ?`
	args = append(args, in.PerPage, offset)

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var expenses []models.Expense
	expenseService := NewExpenseService()

	for rows.Next() {
		var e models.Expense
		var u models.User

		if err := rows.Scan(
			&e.ID, &e.GroupID, &e.PaidByUserID, &e.Title, &e.Description, &e.AmountCents,
			&e.CurrencyCode, &e.Category, &e.ExpenseDate, &e.CreatedAt, &e.UpdatedAt,
			&u.ID, &u.Email, &u.DisplayName, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			continue
		}
		e.PaidBy = &u
		e.Splits, _ = expenseService.GetSplits(e.ID)
		expenses = append(expenses, e)
	}

	if expenses == nil {
		expenses = []models.Expense{}
	}

	return &models.SearchExpenseResult{
		Expenses: expenses,
		Total:    total,
		Page:     in.Page,
		PerPage:  in.PerPage,
	}, nil
}

// GlobalSearch searches across all user's groups.
func (s *SearchService) GlobalSearch(userID, query string, page, perPage int) (*models.SearchExpenseResult, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	searchTerm := "%" + query + "%"

	var total int
	database.DB.QueryRow(
		`SELECT COUNT(*) FROM expenses e
		 INNER JOIN group_members gm ON e.group_id = gm.group_id
		 WHERE gm.user_id = ? AND gm.removed_at IS NULL
		   AND e.deleted_at IS NULL
		   AND (e.title LIKE ? OR e.description LIKE ?)`,
		userID, searchTerm, searchTerm,
	).Scan(&total)

	rows, err := database.DB.Query(
		`SELECT e.id, e.group_id, e.paid_by_user_id, e.title, e.description, e.amount_cents,
		        e.currency_code, e.category, e.expense_date, e.created_at, e.updated_at,
		        u.id, u.email, u.display_name, u.avatar_url, u.created_at, u.updated_at
		 FROM expenses e
		 INNER JOIN group_members gm ON e.group_id = gm.group_id
		 INNER JOIN users u ON e.paid_by_user_id = u.id
		 WHERE gm.user_id = ? AND gm.removed_at IS NULL
		   AND e.deleted_at IS NULL
		   AND (e.title LIKE ? OR e.description LIKE ?)
		 ORDER BY e.expense_date DESC, e.created_at DESC
		 LIMIT ? OFFSET ?`,
		userID, searchTerm, searchTerm, perPage, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var expenses []models.Expense
	expenseService := NewExpenseService()

	for rows.Next() {
		var e models.Expense
		var u models.User

		if err := rows.Scan(
			&e.ID, &e.GroupID, &e.PaidByUserID, &e.Title, &e.Description, &e.AmountCents,
			&e.CurrencyCode, &e.Category, &e.ExpenseDate, &e.CreatedAt, &e.UpdatedAt,
			&u.ID, &u.Email, &u.DisplayName, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			continue
		}
		e.PaidBy = &u
		e.Splits, _ = expenseService.GetSplits(e.ID)
		expenses = append(expenses, e)
	}

	if expenses == nil {
		expenses = []models.Expense{}
	}

	return &models.SearchExpenseResult{
		Expenses: expenses,
		Total:    total,
		Page:     page,
		PerPage:  perPage,
	}, nil
}

// ExportToCSV generates CSV content for a group's expenses.
func (s *SearchService) ExportToCSV(groupID string) (string, error) {
	groupService := NewGroupService()
	_, err := groupService.GetByID(groupID)
	if err != nil {
		return "", err
	}

	members, _ := groupService.GetMembers(groupID)
	memberMap := make(map[string]string)
	for _, m := range members {
		if m.User != nil {
			memberMap[m.UserID] = m.User.DisplayName
		} else {
			memberMap[m.UserID] = m.UserID
		}
	}

	rows, err := database.DB.Query(
		`SELECT e.id, e.title, e.description, e.amount_cents, e.currency_code,
		        e.category, e.expense_date, e.paid_by_user_id,
		        u.display_name
		 FROM expenses e
		 INNER JOIN users u ON e.paid_by_user_id = u.id
		 WHERE e.group_id = ? AND e.deleted_at IS NULL
		 ORDER BY e.expense_date DESC, e.created_at DESC`,
		groupID,
	)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	lines := []string{"Date,Title,Description,Amount,Payer,Category,Split Between"}

	for rows.Next() {
		var id, title, desc, currency, category, payerID, payerName string
		var amountCents int64
		var expenseDate string

		if err := rows.Scan(&id, &title, &desc, &amountCents, &currency, &category, &expenseDate, &payerID, &payerName); err != nil {
			continue
		}

		// Get splits for this expense
		splits, _ := NewExpenseService().GetSplits(id)
		var splitNames []string
		for _, sp := range splits {
			if name, ok := memberMap[sp.UserID]; ok {
				splitNames = append(splitNames, name)
			}
		}
		splitStr := ""
		if len(splitNames) > 0 {
			splitStr = strings.Join(splitNames, "; ")
		}

		amount := strconv.FormatInt(amountCents, 10)
		lines = append(lines, escapeCSV(expenseDate)+","+
			escapeCSV(title)+","+
			escapeCSV(desc)+","+
			amount+","+
			escapeCSV(payerName)+","+
			escapeCSV(category)+","+
			escapeCSV(splitStr))
	}

	return strings.Join(lines, "\n"), nil
}

func escapeCSV(s string) string {
	s = strings.ReplaceAll(s, "\"", "\"\"")
	if strings.ContainsAny(s, ",\n\r") {
		s = "\"" + s + "\""
	}
	return s
}
