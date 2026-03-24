package services

import (
	"fmt"

	"github.com/splitcore/backend/internal/database"
	"github.com/splitcore/backend/internal/models"
)

// SummaryService handles monthly summary calculations.
type SummaryService struct{}

func NewSummaryService() *SummaryService {
	return &SummaryService{}
}

// GetGroupSummary returns monthly breakdown for a specific group and year/month.
func (s *SummaryService) GetGroupSummary(groupID, userID string, year, month int) (*models.MonthlySummary, error) {
	groupService := NewGroupService()
	currencyService := NewCurrencyService()

	group, err := groupService.GetByID(groupID)
	if err != nil {
		return nil, err
	}

	summary := &models.MonthlySummary{
		UserID:       userID,
		GroupID:      groupID,
		Year:         year,
		Month:        month,
		CurrencyCode: group.CurrencyCode,
	}

	// Get user's paid expenses for this group in this month
	var totalPaid int64
	rows, err := database.DB.Query(
		`SELECT COALESCE(SUM(e.amount_cents), 0)
		 FROM expenses e
		 WHERE e.group_id = ? AND e.paid_by_user_id = ?
		   AND e.deleted_at IS NULL
		   AND strftime('%Y', e.expense_date) = ?
		   AND strftime('%m', e.expense_date) = ?`,
		groupID, userID, formatYear(year), formatMonth(month),
	)
	if err == nil {
		defer rows.Close()
		if rows.Next() {
			rows.Scan(&totalPaid)
		}
	}

	// Convert to group currency if needed
	if group.CurrencyCode != "USD" {
		totalPaid, _ = currencyService.ConvertCurrency(totalPaid, "USD", group.CurrencyCode)
	}
	summary.TotalPaidCents = totalPaid

	// Get user's owed (share) expenses for this month
	var totalOwed int64
	owedRows, err := database.DB.Query(
		`SELECT COALESCE(SUM(es.share_amount_cents), 0)
		 FROM expense_splits es
		 INNER JOIN expenses e ON es.expense_id = e.id
		 WHERE e.group_id = ? AND es.user_id = ?
		   AND e.deleted_at IS NULL
		   AND strftime('%Y', e.expense_date) = ?
		   AND strftime('%m', e.expense_date) = ?`,
		groupID, userID, formatYear(year), formatMonth(month),
	)
	if err == nil {
		defer owedRows.Close()
		if owedRows.Next() {
			owedRows.Scan(&totalOwed)
		}
	}

	if group.CurrencyCode != "USD" {
		totalOwed, _ = currencyService.ConvertCurrency(totalOwed, "USD", group.CurrencyCode)
	}
	summary.TotalOwedCents = totalOwed

	// Get settled amounts
	var totalSettled int64
	settledRows, err := database.DB.Query(
		`SELECT COALESCE(SUM(s.amount_cents), 0)
		 FROM settlements s
		 WHERE s.group_id = ? AND (s.from_user_id = ? OR s.to_user_id = ?)
		   AND strftime('%Y', s.settled_at) = ?
		   AND strftime('%m', s.settled_at) = ?`,
		groupID, userID, userID, formatYear(year), formatMonth(month),
	)
	if err == nil {
		defer settledRows.Close()
		if settledRows.Next() {
			settledRows.Scan(&totalSettled)
		}
	}

	if group.CurrencyCode != "USD" {
		totalSettled, _ = currencyService.ConvertCurrency(totalSettled, "USD", group.CurrencyCode)
	}
	summary.TotalSettledCents = totalSettled

	// Get expense count
	var expenseCount int
	countRows, err := database.DB.Query(
		`SELECT COUNT(*) FROM expenses
		 WHERE group_id = ? AND paid_by_user_id = ?
		   AND deleted_at IS NULL
		   AND strftime('%Y', expense_date) = ?
		   AND strftime('%m', expense_date) = ?`,
		groupID, userID, formatYear(year), formatMonth(month),
	)
	if err == nil {
		defer countRows.Close()
		if countRows.Next() {
			countRows.Scan(&expenseCount)
		}
	}
	summary.ExpenseCount = expenseCount

	return summary, nil
}

// GetUserSummary returns global monthly summary across all groups for a user.
func (s *SummaryService) GetUserSummary(userID string, year, month int) ([]models.MonthlySummary, error) {
	groupService := NewGroupService()

	groups, err := groupService.GetUserGroups(userID)
	if err != nil {
		return nil, err
	}

	var summaries []models.MonthlySummary

	for _, group := range groups {
		summary, err := s.GetGroupSummary(group.ID, userID, year, month)
		if err != nil {
			continue
		}
		// Only include non-zero summaries
		if summary.TotalPaidCents > 0 || summary.TotalOwedCents > 0 || summary.ExpenseCount > 0 {
			summaries = append(summaries, *summary)
		}
	}

	return summaries, nil
}

// GetCategoryBreakdown returns spending by category for a user in a group for a specific month.
func (s *SummaryService) GetCategoryBreakdown(groupID, userID string, year, month int) (map[string]int64, error) {
	breakdown := make(map[string]int64)

	rows, err := database.DB.Query(
		`SELECT e.category, COALESCE(SUM(es.share_amount_cents), 0) as total
		 FROM expenses e
		 INNER JOIN expense_splits es ON e.id = es.expense_id
		 WHERE e.group_id = ? AND es.user_id = ?
		   AND e.deleted_at IS NULL
		   AND strftime('%Y', e.expense_date) = ?
		   AND strftime('%m', e.expense_date) = ?
		 GROUP BY e.category`,
		groupID, userID, formatYear(year), formatMonth(month),
	)
	if err != nil {
		return breakdown, err
	}
	defer rows.Close()

	for rows.Next() {
		var category string
		var total int64
		if err := rows.Scan(&category, &total); err != nil {
			continue
		}
		breakdown[category] = total
	}

	return breakdown, nil
}

func formatYear(year int) string {
	return fmt.Sprintf("%04d", year)
}

func formatMonth(month int) string {
	return fmt.Sprintf("%02d", month)
}
