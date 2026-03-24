package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/nupurbhaisare/splitcore-backend/internal/middleware"
	"github.com/nupurbhaisare/splitcore-backend/internal/services"
	"github.com/nupurbhaisare/splitcore-backend/pkg/utils"
)

type DeviceHandler struct {
	pushTokenService *services.PushTokenService
}

func NewDeviceHandler() *DeviceHandler {
	return &DeviceHandler{
		pushTokenService: services.NewPushTokenService(),
	}
}

type RegisterDeviceRequest struct {
	DeviceToken string `json:"device_token"`
	Platform    string `json:"platform"`
}

// RegisterDevice registers a push token for the authenticated user.
func (h *DeviceHandler) RegisterDevice(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	if userID == "" {
		utils.Unauthorized(w, "User not authenticated")
		return
	}

	var req RegisterDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body")
		return
	}

	if req.DeviceToken == "" {
		utils.BadRequest(w, "device_token is required")
		return
	}

	platform := req.Platform
	if platform == "" {
		platform = "ios"
	}

	token, err := h.pushTokenService.Register(userID, req.DeviceToken, platform)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}

	utils.Success(w, http.StatusCreated, token)
}

// UnregisterDevice removes a push token.
func (h *DeviceHandler) UnregisterDevice(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	if userID == "" {
		utils.Unauthorized(w, "User not authenticated")
		return
	}

	deviceToken := mux.Vars(r)["deviceToken"]
	if deviceToken == "" {
		utils.BadRequest(w, "device token is required")
		return
	}

	if err := h.pushTokenService.Unregister(deviceToken, userID); err != nil {
		utils.InternalError(w, err.Error())
		return
	}

	utils.Success(w, http.StatusOK, map[string]string{"message": "Device unregistered"})
}
