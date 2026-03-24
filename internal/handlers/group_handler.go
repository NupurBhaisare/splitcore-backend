package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/nupurbhaisare/splitcore-backend/internal/middleware"
	"github.com/nupurbhaisare/splitcore-backend/internal/services"
	"github.com/nupurbhaisare/splitcore-backend/pkg/utils"
)

type GroupHandler struct {
	groupService    *services.GroupService
	expenseService  *services.ExpenseService
	balanceService  *services.BalanceService
	userService     *services.UserService
	activityService *services.ActivityService
}

func NewGroupHandler() *GroupHandler {
	return &GroupHandler{
		groupService:    services.NewGroupService(),
		expenseService:  services.NewExpenseService(),
		balanceService: services.NewBalanceService(),
		userService:     services.NewUserService(),
		activityService: services.NewActivityService(),
	}
}

type CreateGroupRequest struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	IconEmoji    string `json:"icon_emoji"`
	CurrencyCode string `json:"currency_code"`
}

type UpdateGroupRequest struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	IconEmoji    string `json:"icon_emoji"`
	CurrencyCode string `json:"currency_code"`
}

type AddMemberRequest struct {
	Email    string `json:"email"`
	Nickname string `json:"nickname"`
	UserID   string `json:"user_id"`
}

type CreateExpenseRequest struct {
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	AmountCents  int64    `json:"amount_cents"`
	CurrencyCode string   `json:"currency_code"`
	Category     string   `json:"category"`
	ExpenseDate  string   `json:"expense_date"`
	SplitUserIDs []string `json:"split_user_ids"`
	SplitType   string             `json:"split_type"` // equal, percentage, exact, shares
	Splits       []ExpenseSplitInput `json:"splits"`    // individual split data for percentage/exact/shares
}

type ExpenseSplitInput struct {
	UserID      string  `json:"user_id"`
	ShareCents  int64   `json:"share_cents"`
	Percentage  float64 `json:"percentage"`
	ShareCount  int     `json:"share_count"`
}

type UpdateExpenseRequest struct {
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	AmountCents  int64    `json:"amount_cents"`
	Category     string   `json:"category"`
	SplitUserIDs []string `json:"split_user_ids"`
}

// --- Groups ---

func (h *GroupHandler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	if userID == "" {
		utils.Unauthorized(w, "User not authenticated")
		return
	}

	var req CreateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body")
		return
	}

	if req.Name == "" {
		utils.BadRequest(w, "Group name is required")
		return
	}

	group, err := h.groupService.Create(userID, req.Name, req.Description, req.IconEmoji, req.CurrencyCode)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}

	members, _ := h.groupService.GetMembers(group.ID)

	// Record group_created activity
	h.activityService.Create(services.CreateActivityInput{
		GroupID:      group.ID,
		UserID:       userID,
		ActivityType: services.ActivityGroupCreated,
		Metadata: map[string]interface{}{
			"group_name": req.Name,
		},
	})

	utils.Success(w, http.StatusCreated, map[string]interface{}{
		"group":   group,
		"members": members,
	})
}

func (h *GroupHandler) GetGroups(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	if userID == "" {
		utils.Unauthorized(w, "User not authenticated")
		return
	}

	groups, err := h.groupService.GetUserGroups(userID)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}

	type GroupWithMeta struct {
		Group        interface{} `json:"group"`
		MemberCount  int         `json:"member_count"`
		YourBalance  int64       `json:"your_balance_cents"`
		YourStatus   string      `json:"your_status"`
	}

	var enriched []GroupWithMeta
	for _, g := range groups {
		members, _ := h.groupService.GetMembers(g.ID)
		balance, _ := h.balanceService.GetUserGroupBalance(g.ID, userID)

		var balanceCents int64
		var status string
		if balance != nil {
			balanceCents = balance.NetCents
			status = balance.Status
		}

		enriched = append(enriched, GroupWithMeta{
			Group:        g,
			MemberCount:  len(members),
			YourBalance:  balanceCents,
			YourStatus:   status,
		})
	}

	utils.Success(w, http.StatusOK, enriched)
}

func (h *GroupHandler) GetGroup(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	groupID := mux.Vars(r)["id"]

	if userID == "" {
		utils.Unauthorized(w, "User not authenticated")
		return
	}

	group, err := h.groupService.GetByID(groupID)
	if err != nil {
		utils.NotFound(w, err.Error())
		return
	}

	if !h.groupService.IsMember(groupID, userID) {
		utils.Forbidden(w, "You are not a member of this group")
		return
	}

	members, _ := h.groupService.GetMembers(groupID)
	expenses, _ := h.expenseService.GetByGroup(groupID)
	balances, _ := h.balanceService.GetGroupBalances(groupID)

	utils.Success(w, http.StatusOK, map[string]interface{}{
		"group":    group,
		"members":  members,
		"expenses": expenses,
		"balances": balances,
	})
}

func (h *GroupHandler) UpdateGroup(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	groupID := mux.Vars(r)["id"]

	if userID == "" {
		utils.Unauthorized(w, "User not authenticated")
		return
	}

	var req UpdateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body")
		return
	}

	group, err := h.groupService.Update(groupID, userID, req.Name, req.Description, req.IconEmoji, req.CurrencyCode)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	utils.Success(w, http.StatusOK, group)
}

func (h *GroupHandler) DeleteGroup(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	groupID := mux.Vars(r)["id"]

	if userID == "" {
		utils.Unauthorized(w, "User not authenticated")
		return
	}

	if err := h.groupService.Delete(groupID, userID); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	utils.Success(w, http.StatusOK, map[string]string{"message": "Group deleted"})
}

// --- Members ---

func (h *GroupHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	groupID := mux.Vars(r)["id"]

	if userID == "" {
		utils.Unauthorized(w, "User not authenticated")
		return
	}

	var req AddMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body")
		return
	}

	targetUserID := req.UserID
	if targetUserID == "" && req.Email != "" {
		user, err := h.userService.GetByEmail(req.Email)
		if err != nil {
			utils.NotFound(w, "User not found with that email")
			return
		}
		targetUserID = user.ID
	}

	if targetUserID == "" {
		utils.BadRequest(w, "Either email or user_id is required")
		return
	}

	member, err := h.groupService.AddMember(groupID, targetUserID, req.Nickname, userID)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	// Record member_joined activity
	h.activityService.Create(services.CreateActivityInput{
		GroupID:       groupID,
		UserID:        userID,
		TargetUserIDs: []string{targetUserID},
		ActivityType: services.ActivityMemberJoined,
		Metadata: map[string]interface{}{
			"member_user_id": targetUserID,
		},
	})

	utils.Success(w, http.StatusCreated, member)
}

func (h *GroupHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	groupID := mux.Vars(r)["id"]
	memberUserID := mux.Vars(r)["memberUserId"]

	if userID == "" {
		utils.Unauthorized(w, "User not authenticated")
		return
	}

	if err := h.groupService.RemoveMember(groupID, memberUserID, userID); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	// Record member_left activity
	h.activityService.Create(services.CreateActivityInput{
		GroupID:       groupID,
		UserID:        userID,
		TargetUserIDs: []string{memberUserID},
		ActivityType: services.ActivityMemberLeft,
		Metadata: map[string]interface{}{
			"member_user_id": memberUserID,
		},
	})

	utils.Success(w, http.StatusOK, map[string]string{"message": "Member removed"})
}

func (h *GroupHandler) GetMembers(w http.ResponseWriter, r *http.Request) {
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

	members, err := h.groupService.GetMembers(groupID)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}

	utils.Success(w, http.StatusOK, members)
}

// --- Expenses ---

func (h *GroupHandler) GetExpenses(w http.ResponseWriter, r *http.Request) {
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

	expenses, err := h.expenseService.GetByGroup(groupID)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}

	utils.Success(w, http.StatusOK, expenses)
}

func (h *GroupHandler) CreateExpense(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	groupID := mux.Vars(r)["id"]

	if userID == "" {
		utils.Unauthorized(w, "User not authenticated")
		return
	}

	var req CreateExpenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body")
		return
	}

	if req.Title == "" {
		utils.BadRequest(w, "Title is required")
		return
	}
	if req.AmountCents <= 0 {
		utils.BadRequest(w, "Amount must be greater than 0")
		return
	}

	expenseDate := time.Now()
	if req.ExpenseDate != "" {
		parsed, err := time.Parse(time.RFC3339, req.ExpenseDate)
		if err == nil {
			expenseDate = parsed
		}
	}

	expense, err := h.expenseService.Create(services.CreateExpenseInput{
		GroupID:      groupID,
		PaidByUserID: userID,
		Title:        req.Title,
		Description:  req.Description,
		AmountCents:  req.AmountCents,
		CurrencyCode: req.CurrencyCode,
		Category:     req.Category,
		ExpenseDate:  expenseDate,
		SplitUserIDs: req.SplitUserIDs,
		SplitType:   req.SplitType,
		Splits: func() []services.SplitInput {
			var out []services.SplitInput
			for _, sp := range req.Splits {
				out = append(out, services.SplitInput{
					UserID:     sp.UserID,
					ShareCents: sp.ShareCents,
					Percentage: sp.Percentage,
					ShareCount: sp.ShareCount,
				})
			}
			return out
		}(),
	})
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	// Record expense_added activity
	h.activityService.Create(services.CreateActivityInput{
		GroupID:       groupID,
		UserID:        userID,
		TargetUserIDs: req.SplitUserIDs,
		ActivityType:  services.ActivityExpenseAdded,
		Metadata: map[string]interface{}{
			"expense_id":    expense.ID,
			"title":         req.Title,
			"amount_cents":  req.AmountCents,
			"category":     req.Category,
		},
	})

	utils.Success(w, http.StatusCreated, expense)
}

func (h *GroupHandler) GetExpense(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	groupID := mux.Vars(r)["id"]
	expenseID := mux.Vars(r)["expenseId"]

	if userID == "" {
		utils.Unauthorized(w, "User not authenticated")
		return
	}

	if !h.groupService.IsMember(groupID, userID) {
		utils.Forbidden(w, "You are not a member of this group")
		return
	}

	expense, err := h.expenseService.GetByID(expenseID, groupID)
	if err != nil {
		utils.NotFound(w, err.Error())
		return
	}

	utils.Success(w, http.StatusOK, expense)
}

func (h *GroupHandler) UpdateExpense(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	groupID := mux.Vars(r)["id"]
	expenseID := mux.Vars(r)["expenseId"]

	if userID == "" {
		utils.Unauthorized(w, "User not authenticated")
		return
	}

	var req UpdateExpenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body")
		return
	}

	expense, err := h.expenseService.Update(expenseID, groupID, userID, req.Title, req.Description, req.Category, req.AmountCents, req.SplitUserIDs)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	// Record expense_edited activity
	h.activityService.Create(services.CreateActivityInput{
		GroupID:       groupID,
		UserID:        userID,
		TargetUserIDs: req.SplitUserIDs,
		ActivityType:  services.ActivityExpenseEdited,
		Metadata: map[string]interface{}{
			"expense_id":   expenseID,
			"title":        req.Title,
			"amount_cents": req.AmountCents,
		},
	})

	utils.Success(w, http.StatusOK, expense)
}

func (h *GroupHandler) DeleteExpense(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	groupID := mux.Vars(r)["id"]
	expenseID := mux.Vars(r)["expenseId"]

	if userID == "" {
		utils.Unauthorized(w, "User not authenticated")
		return
	}

	if err := h.expenseService.Delete(expenseID, groupID, userID); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	// Record expense_deleted activity
	h.activityService.Create(services.CreateActivityInput{
		GroupID:      groupID,
		UserID:       userID,
		ActivityType: services.ActivityExpenseDeleted,
		Metadata: map[string]interface{}{
			"expense_id": expenseID,
		},
	})

	utils.Success(w, http.StatusOK, map[string]string{"message": "Expense deleted"})
}

// --- Balances ---

func (h *GroupHandler) GetBalances(w http.ResponseWriter, r *http.Request) {
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

	balances, err := h.balanceService.GetGroupBalances(groupID)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}

	utils.Success(w, http.StatusOK, balances)
}

// --- Join by invite ---

func (h *GroupHandler) JoinByCode(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	if userID == "" {
		utils.Unauthorized(w, "User not authenticated")
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		utils.BadRequest(w, "Invite code is required")
		return
	}

	group, err := h.groupService.JoinByInviteCode(code, userID)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	members, _ := h.groupService.GetMembers(group.ID)

	utils.Success(w, http.StatusOK, map[string]interface{}{
		"group":   group,
		"members": members,
	})
}

func getPagination(r *http.Request) (int, int) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	return page, perPage
}
