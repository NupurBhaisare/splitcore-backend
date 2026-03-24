package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/nupurbhaisare/splitcore-backend/internal/middleware"
	"github.com/nupurbhaisare/splitcore-backend/internal/services"
	"github.com/nupurbhaisare/splitcore-backend/pkg/utils"
)

type UserHandler struct {
	userService *services.UserService
}

func NewUserHandler() *UserHandler {
	return &UserHandler{
		userService: services.NewUserService(),
	}
}

func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	if userID == "" {
		utils.Unauthorized(w, "User not authenticated")
		return
	}

	user, err := h.userService.GetByID(userID)
	if err != nil {
		utils.NotFound(w, err.Error())
		return
	}

	utils.Success(w, http.StatusOK, user)
}

type UpdateUserRequest struct {
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
}

func (h *UserHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	if userID == "" {
		utils.Unauthorized(w, "User not authenticated")
		return
	}

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body")
		return
	}

	user, err := h.userService.Update(userID, req.DisplayName, req.AvatarURL)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}

	utils.Success(w, http.StatusOK, user)
}

func (h *UserHandler) GetUserByEmail(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	if email == "" {
		utils.BadRequest(w, "Email query parameter is required")
		return
	}

	user, err := h.userService.GetByEmail(email)
	if err != nil {
		utils.NotFound(w, "User not found")
		return
	}

	// Only return public info
	utils.Success(w, http.StatusOK, map[string]string{
		"id":           user.ID,
		"email":        user.Email,
		"display_name": user.DisplayName,
		"avatar_url":   user.AvatarURL,
	})
}
