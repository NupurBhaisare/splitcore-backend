package handlers

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/splitcore/backend/internal/middleware"
	"github.com/splitcore/backend/internal/services"
	"github.com/splitcore/backend/pkg/utils"
)

type SearchHandler struct {
	searchService *services.SearchService
	groupService  *services.GroupService
}

func NewSearchHandler() *SearchHandler {
	return &SearchHandler{
		searchService: services.NewSearchService(),
		groupService:  services.NewGroupService(),
	}
}

// SearchExpenses searches expenses within a group.
func (h *SearchHandler) SearchExpenses(w http.ResponseWriter, r *http.Request) {
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

	query := r.URL.Query()
	page, _ := strconv.Atoi(query.Get("page"))
	perPage, _ := strconv.Atoi(query.Get("per_page"))

	minAmount, _ := strconv.ParseInt(query.Get("min_amount"), 10, 64)
	maxAmount, _ := strconv.ParseInt(query.Get("max_amount"), 10, 64)

	results, err := h.searchService.SearchExpenses(services.SearchExpensesInput{
		GroupID:   groupID,
		Query:     query.Get("q"),
		Category:  query.Get("category"),
		MinAmount: minAmount,
		MaxAmount: maxAmount,
		StartDate: query.Get("start_date"),
		EndDate:   query.Get("end_date"),
		PayerID:   query.Get("payer_id"),
		Page:      page,
		PerPage:   perPage,
	})
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}

	utils.Success(w, http.StatusOK, results)
}

// GlobalSearch searches across all user's groups.
func (h *SearchHandler) GlobalSearch(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	if userID == "" {
		utils.Unauthorized(w, "User not authenticated")
		return
	}

	query := r.URL.Query()
	searchQuery := query.Get("q")
	page, _ := strconv.Atoi(query.Get("page"))
	perPage, _ := strconv.Atoi(query.Get("per_page"))

	if searchQuery == "" {
		utils.BadRequest(w, "Search query (q) is required")
		return
	}

	results, err := h.searchService.GlobalSearch(userID, searchQuery, page, perPage)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}

	utils.Success(w, http.StatusOK, results)
}

// ExportCSV exports group expenses as CSV.
func (h *SearchHandler) ExportCSV(w http.ResponseWriter, r *http.Request) {
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

	csv, err := h.searchService.ExportToCSV(groupID)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=expenses.csv")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(csv))
}
