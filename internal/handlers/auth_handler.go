package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/nupurbhaisare/splitcore-backend/internal/services"
	"github.com/nupurbhaisare/splitcore-backend/pkg/utils"
)

type AuthHandler struct {
	userService *services.UserService
}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{
		userService: services.NewUserService(),
	}
}

type RegisterRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		utils.BadRequest(w, "Email and password are required")
		return
	}

	if len(req.Password) < 6 {
		utils.BadRequest(w, "Password must be at least 6 characters")
		return
	}

	user, err := h.userService.Create(req.Email, req.Password, req.DisplayName)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	accessToken, err := utils.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		utils.InternalError(w, "Failed to generate access token")
		return
	}

	refreshToken, err := utils.GenerateRefreshToken(user.ID, user.Email)
	if err != nil {
		utils.InternalError(w, "Failed to generate refresh token")
		return
	}

	utils.Success(w, http.StatusCreated, map[string]interface{}{
		"user":           user,
		"access_token":   accessToken,
		"refresh_token":  refreshToken,
		"expires_in":     utils.GetAccessTokenExpirySeconds(),
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		utils.BadRequest(w, "Email and password are required")
		return
	}

	user, err := h.userService.Authenticate(req.Email, req.Password)
	if err != nil {
		utils.Unauthorized(w, err.Error())
		return
	}

	accessToken, err := utils.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		utils.InternalError(w, "Failed to generate access token")
		return
	}

	refreshToken, err := utils.GenerateRefreshToken(user.ID, user.Email)
	if err != nil {
		utils.InternalError(w, "Failed to generate refresh token")
		return
	}

	utils.Success(w, http.StatusOK, map[string]interface{}{
		"user":           user,
		"access_token":   accessToken,
		"refresh_token":  refreshToken,
		"expires_in":     utils.GetAccessTokenExpirySeconds(),
	})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body")
		return
	}

	if req.RefreshToken == "" {
		utils.BadRequest(w, "Refresh token is required")
		return
	}

	claims, err := utils.ValidateToken(req.RefreshToken)
	if err != nil {
		utils.Unauthorized(w, "Invalid or expired refresh token")
		return
	}

	// Verify user still exists
	user, err := h.userService.GetByID(claims.UserID)
	if err != nil {
		utils.Unauthorized(w, "User not found")
		return
	}

	accessToken, err := utils.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		utils.InternalError(w, "Failed to generate access token")
		return
	}

	refreshToken, err := utils.GenerateRefreshToken(user.ID, user.Email)
	if err != nil {
		utils.InternalError(w, "Failed to generate refresh token")
		return
	}

	utils.Success(w, http.StatusOK, map[string]interface{}{
		"user":           user,
		"access_token":   accessToken,
		"refresh_token":  refreshToken,
		"expires_in":     utils.GetAccessTokenExpirySeconds(),
	})
}
