package routes

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/nupurbhaisare/splitcore-backend/internal/handlers"
	"github.com/nupurbhaisare/splitcore-backend/internal/middleware"
)

func NewRouter() *mux.Router {
	router := mux.NewRouter()
	router.Use(middleware.CORS)
	router.Use(middleware.Logger)

	authHandler := handlers.NewAuthHandler()
	userHandler := handlers.NewUserHandler()
	groupHandler := handlers.NewGroupHandler()
	debtHandler := handlers.NewDebtHandler()
	settlementHandler := handlers.NewSettlementHandler()
	activityHandler := handlers.NewActivityHandler()
	deviceHandler := handlers.NewDeviceHandler()
	currencyHandler := handlers.NewCurrencyHandler()
	commentHandler := handlers.NewCommentHandler()
	summaryHandler := handlers.NewSummaryHandler()
	searchHandler := handlers.NewSearchHandler()

	// Health check
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}).Methods(http.MethodGet, http.MethodOptions)

	// Auth routes (public)
	router.HandleFunc("/auth/register", authHandler.Register).Methods(http.MethodPost, http.MethodOptions)
	router.HandleFunc("/auth/login", authHandler.Login).Methods(http.MethodPost, http.MethodOptions)
	router.HandleFunc("/auth/refresh", authHandler.Refresh).Methods(http.MethodPost, http.MethodOptions)

	// Public user lookup
	router.HandleFunc("/users/lookup", userHandler.GetUserByEmail).Methods(http.MethodGet)

	// Public currency routes
	router.HandleFunc("/currencies", currencyHandler.GetCurrencies).Methods(http.MethodGet, http.MethodOptions)
	router.HandleFunc("/exchange-rates", currencyHandler.GetExchangeRates).Methods(http.MethodGet, http.MethodOptions)

	// Protected routes
	api := router.PathPrefix("/api/v1").Subrouter()
	api.Use(middleware.AuthMiddleware)

	// User routes
	api.HandleFunc("/users/me", userHandler.GetMe).Methods(http.MethodGet)
	api.HandleFunc("/users/me", userHandler.UpdateMe).Methods(http.MethodPatch)

	// Push token routes
	api.HandleFunc("/users/devices", deviceHandler.RegisterDevice).Methods(http.MethodPost, http.MethodOptions)
	api.HandleFunc("/users/devices/{deviceToken}", deviceHandler.UnregisterDevice).Methods(http.MethodDelete)

	// Group routes
	api.HandleFunc("/groups", groupHandler.GetGroups).Methods(http.MethodGet)
	api.HandleFunc("/groups", groupHandler.CreateGroup).Methods(http.MethodPost)
	api.HandleFunc("/groups/join", groupHandler.JoinByCode).Methods(http.MethodPost)
	api.HandleFunc("/groups/{id}", groupHandler.GetGroup).Methods(http.MethodGet)
	api.HandleFunc("/groups/{id}", groupHandler.UpdateGroup).Methods(http.MethodPatch)
	api.HandleFunc("/groups/{id}", groupHandler.DeleteGroup).Methods(http.MethodDelete)

	// Group members
	api.HandleFunc("/groups/{id}/members", groupHandler.GetMembers).Methods(http.MethodGet)
	api.HandleFunc("/groups/{id}/members", groupHandler.AddMember).Methods(http.MethodPost)
	api.HandleFunc("/groups/{id}/members/{memberUserId}", groupHandler.RemoveMember).Methods(http.MethodDelete)

	// Group expenses
	api.HandleFunc("/groups/{id}/expenses", groupHandler.GetExpenses).Methods(http.MethodGet)
	api.HandleFunc("/groups/{id}/expenses", groupHandler.CreateExpense).Methods(http.MethodPost)
	api.HandleFunc("/groups/{id}/expenses/{expenseId}", groupHandler.GetExpense).Methods(http.MethodGet)
	api.HandleFunc("/groups/{id}/expenses/{expenseId}", groupHandler.UpdateExpense).Methods(http.MethodPatch)
	api.HandleFunc("/groups/{id}/expenses/{expenseId}", groupHandler.DeleteExpense).Methods(http.MethodDelete)

	// Expense comments
	api.HandleFunc("/groups/{id}/expenses/{expenseId}/comments", commentHandler.GetComments).Methods(http.MethodGet)
	api.HandleFunc("/groups/{id}/expenses/{expenseId}/comments", commentHandler.CreateComment).Methods(http.MethodPost, http.MethodOptions)

	// Group balances
	api.HandleFunc("/groups/{id}/balances", groupHandler.GetBalances).Methods(http.MethodGet)

	// Group debts (simplified)
	api.HandleFunc("/groups/{id}/debts", debtHandler.GetSimplifiedDebts).Methods(http.MethodGet)

	// Group settlements
	api.HandleFunc("/groups/{id}/settlements", settlementHandler.GetSettlements).Methods(http.MethodGet)
	api.HandleFunc("/groups/{id}/settlements", settlementHandler.CreateSettlement).Methods(http.MethodPost, http.MethodOptions)
	api.HandleFunc("/groups/{id}/settlements/{settlementId}", settlementHandler.DeleteSettlement).Methods(http.MethodDelete)

	// Group activities
	api.HandleFunc("/groups/{id}/activities", activityHandler.GetActivities).Methods(http.MethodGet)

	// Global activities
	api.HandleFunc("/activities/unread-count", activityHandler.GetUnreadCount).Methods(http.MethodGet)
	api.HandleFunc("/activities/read", activityHandler.MarkActivitiesRead).Methods(http.MethodPatch, http.MethodOptions)

	// Comments delete
	api.HandleFunc("/comments/{commentId}", commentHandler.DeleteComment).Methods(http.MethodDelete)

	// Group summary
	api.HandleFunc("/groups/{id}/summary", summaryHandler.GetGroupSummary).Methods(http.MethodGet)

	// User summary
	api.HandleFunc("/users/me/summary", summaryHandler.GetUserSummary).Methods(http.MethodGet)

	// Search expenses within a group
	api.HandleFunc("/groups/{id}/expenses/search", searchHandler.SearchExpenses).Methods(http.MethodGet)

	// Export CSV
	api.HandleFunc("/groups/{id}/expenses/export", searchHandler.ExportCSV).Methods(http.MethodPost)

	return router
}
