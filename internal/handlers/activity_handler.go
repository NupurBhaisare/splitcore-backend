package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/nupurbhaisare/splitcore-backend/internal/middleware"
	"github.com/nupurbhaisare/splitcore-backend/internal/services"
	"github.com/nupurbhaisare/splitcore-backend/pkg/utils"
)

type ActivityHandler struct {
	activityService *services.ActivityService
	groupService    *services.GroupService
}

func NewActivityHandler() *ActivityHandler {
	return &ActivityHandler{
		activityService: services.NewActivityService(),
		groupService:    services.NewGroupService(),
	}
}

// GetActivities returns paginated activity feed for a group.
func (h *ActivityHandler) GetActivities(w http.ResponseWriter, r *http.Request) {
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

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	activities, total, err := h.activityService.GetByGroup(groupID, page, perPage)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}

	utils.Success(w, http.StatusOK, map[string]interface{}{
		"activities": activities,
		"total":      total,
		"page":       page,
		"per_page":   perPage,
	})
}

// GetUnreadCount returns the count of unread activities.
func (h *ActivityHandler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	if userID == "" {
		utils.Unauthorized(w, "User not authenticated")
		return
	}

	count, err := h.activityService.GetUnreadCount(userID)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}

	utils.Success(w, http.StatusOK, map[string]int{"unread_count": count})
}

// MarkActivitiesRead marks all activities in a group as read.
func (h *ActivityHandler) MarkActivitiesRead(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	groupID := mux.Vars(r)["id"]

	if userID == "" {
		utils.Unauthorized(w, "User not authenticated")
		return
	}

	var req struct {
		GroupID string `json:"group_id"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if req.GroupID == "" {
		req.GroupID = groupID
	}

	if !h.groupService.IsMember(req.GroupID, userID) {
		utils.Forbidden(w, "You are not a member of this group")
		return
	}

	if err := h.activityService.MarkRead(req.GroupID, userID); err != nil {
		utils.InternalError(w, err.Error())
		return
	}

	utils.Success(w, http.StatusOK, map[string]string{"message": "Activities marked as read"})
}
