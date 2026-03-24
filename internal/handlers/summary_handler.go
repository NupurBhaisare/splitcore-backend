package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/nupurbhaisare/splitcore-backend/internal/middleware"
	"github.com/nupurbhaisare/splitcore-backend/internal/services"
	"github.com/nupurbhaisare/splitcore-backend/pkg/utils"
)

type SummaryHandler struct {
	summaryService *services.SummaryService
	groupService   *services.GroupService
}

func NewSummaryHandler() *SummaryHandler {
	return &SummaryHandler{
		summaryService: services.NewSummaryService(),
		groupService:   services.NewGroupService(),
	}
}

// GetGroupSummary returns monthly summary for a group.
func (h *SummaryHandler) GetGroupSummary(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	groupID := mux.Vars(r)["id"]

	if userID == "" {
		utils.Unauthorized(w, "User not authenticated")
		return
	}

	if !h.groupService.IsMember(groupID, userID) {
		utils.Forbidden(w, "You are not a member of this group")
		return
	}

	// Parse year and month from query params, default to current
	now := time.Now()
	year, _ := strconv.Atoi(r.URL.Query().Get("year"))
	if year == 0 {
		year = now.Year()
	}
	month, _ := strconv.Atoi(r.URL.Query().Get("month"))
	if month == 0 {
		month = int(now.Month())
	}

	summary, err := h.summaryService.GetGroupSummary(groupID, userID, year, month)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}

	// Also get category breakdown
	breakdown, _ := h.summaryService.GetCategoryBreakdown(groupID, userID, year, month)

	utils.Success(w, http.StatusOK, map[string]interface{}{
		"summary":   summary,
		"breakdown": breakdown,
	})
}

// GetUserSummary returns global monthly summary for the current user.
func (h *SummaryHandler) GetUserSummary(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	if userID == "" {
		utils.Unauthorized(w, "User not authenticated")
		return
	}

	now := time.Now()
	year, _ := strconv.Atoi(r.URL.Query().Get("year"))
	if year == 0 {
		year = now.Year()
	}
	month, _ := strconv.Atoi(r.URL.Query().Get("month"))
	if month == 0 {
		month = int(now.Month())
	}

	summaries, err := h.summaryService.GetUserSummary(userID, year, month)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}

	// Calculate totals across all groups
	var totalPaid, totalOwed, totalSettled int64
	var totalExpenses int
	for _, s := range summaries {
		totalPaid += s.TotalPaidCents
		totalOwed += s.TotalOwedCents
		totalSettled += s.TotalSettledCents
		totalExpenses += s.ExpenseCount
	}

	utils.Success(w, http.StatusOK, map[string]interface{}{
		"summaries":      summaries,
		"total_paid":     totalPaid,
		"total_owed":     totalOwed,
		"total_settled":  totalSettled,
		"total_expenses": totalExpenses,
		"year":           year,
		"month":          month,
	})
}
