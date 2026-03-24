package handlers

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/nupurbhaisare/splitcore-backend/internal/middleware"
	"github.com/nupurbhaisare/splitcore-backend/internal/services"
	"github.com/nupurbhaisare/splitcore-backend/pkg/utils"
)

type DebtHandler struct {
	debtService  *services.DebtService
	groupService *services.GroupService
}

func NewDebtHandler() *DebtHandler {
	return &DebtHandler{
		debtService:  services.NewDebtService(),
		groupService: services.NewGroupService(),
	}
}

// GetSimplifiedDebts returns the simplified debt graph for a group.
func (h *DebtHandler) GetSimplifiedDebts(w http.ResponseWriter, r *http.Request) {
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

	debts, err := h.debtService.GetSimplifiedDebts(groupID)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}

	utils.Success(w, http.StatusOK, debts)
}
