package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/splitcore/backend/internal/middleware"
	"github.com/splitcore/backend/internal/services"
	"github.com/splitcore/backend/pkg/utils"
)

type SettlementHandler struct {
	settlementService *services.SettlementService
	groupService      *services.GroupService
	activityService   *services.ActivityService
}

func NewSettlementHandler() *SettlementHandler {
	return &SettlementHandler{
		settlementService: services.NewSettlementService(),
		groupService:       services.NewGroupService(),
		activityService:    services.NewActivityService(),
	}
}

type CreateSettlementRequest struct {
	FromUserID    string `json:"from_user_id"`
	ToUserID      string `json:"to_user_id"`
	AmountCents   int64  `json:"amount_cents"`
	Note          string `json:"note"`
	PaymentMethod string `json:"payment_method"`
}

// CreateSettlement records a new settlement.
func (h *SettlementHandler) CreateSettlement(w http.ResponseWriter, r *http.Request) {
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

	var req CreateSettlementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body")
		return
	}

	if req.FromUserID == "" || req.ToUserID == "" {
		utils.BadRequest(w, "from_user_id and to_user_id are required")
		return
	}
	if req.AmountCents <= 0 {
		utils.BadRequest(w, "Amount must be greater than 0")
		return
	}

	// Validate both users are members of the group
	if !h.groupService.IsMember(groupID, req.FromUserID) {
		utils.BadRequest(w, "from_user_id is not a member of this group")
		return
	}
	if !h.groupService.IsMember(groupID, req.ToUserID) {
		utils.BadRequest(w, "to_user_id is not a member of this group")
		return
	}

	settlement, err := h.settlementService.Create(services.CreateSettlementInput{
		GroupID:         groupID,
		FromUserID:      req.FromUserID,
		ToUserID:        req.ToUserID,
		AmountCents:     req.AmountCents,
		Note:            req.Note,
		PaymentMethod:   req.PaymentMethod,
		CreatedByUserID: userID,
	})
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}

	// Record activity
	h.activityService.Create(services.CreateActivityInput{
		GroupID:       groupID,
		UserID:        userID,
		TargetUserIDs: []string{req.FromUserID, req.ToUserID},
		ActivityType:  services.ActivitySettlementMade,
		Metadata: map[string]interface{}{
			"settlement_id": settlement.ID,
			"amount_cents":  req.AmountCents,
		},
	})

	utils.Success(w, http.StatusCreated, settlement)
}

// GetSettlements returns all settlements for a group.
func (h *SettlementHandler) GetSettlements(w http.ResponseWriter, r *http.Request) {
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

	settlements, err := h.settlementService.GetByGroup(groupID)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}

	utils.Success(w, http.StatusOK, settlements)
}

// DeleteSettlement removes a settlement (only the creator can delete it).
func (h *SettlementHandler) DeleteSettlement(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	groupID := mux.Vars(r)["id"]
	settlementID := mux.Vars(r)["settlementId"]

	if userID == "" {
		utils.Unauthorized(w, "User not authenticated")
		return
	}

	if !h.groupService.IsMember(groupID, userID) {
		utils.Forbidden(w, "You are not a member of this group")
		return
	}

	if err := h.settlementService.Delete(settlementID, groupID, userID); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	utils.Success(w, http.StatusOK, map[string]string{"message": "Settlement deleted"})
}
