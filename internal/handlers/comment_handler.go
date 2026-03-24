package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/splitcore/backend/internal/middleware"
	"github.com/splitcore/backend/internal/models"
	"github.com/splitcore/backend/internal/services"
	"github.com/splitcore/backend/pkg/utils"
)

type CommentHandler struct {
	commentService  *services.CommentService
	expenseService *services.ExpenseService
	groupService   *services.GroupService
	activityService *services.ActivityService
}

func NewCommentHandler() *CommentHandler {
	return &CommentHandler{
		commentService:  services.NewCommentService(),
		expenseService:  services.NewExpenseService(),
		groupService:    services.NewGroupService(),
		activityService: services.NewActivityService(),
	}
}

type CreateCommentRequest struct {
	Body string `json:"body"`
}

type UpdateCommentRequest struct {
	Body string `json:"body"`
}

// GetComments returns all comments for an expense.
func (h *CommentHandler) GetComments(w http.ResponseWriter, r *http.Request) {
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

	comments, err := h.commentService.GetByExpense(expenseID)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}

	if comments == nil {
		comments = []models.ExpenseComment{}
	}

	utils.Success(w, http.StatusOK, comments)
}

// CreateComment adds a new comment to an expense.
func (h *CommentHandler) CreateComment(w http.ResponseWriter, r *http.Request) {
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

	var req CreateCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body")
		return
	}

	if strings.TrimSpace(req.Body) == "" {
		utils.BadRequest(w, "Comment body cannot be empty")
		return
	}

	comment, err := h.commentService.Create(services.CreateCommentInput{
		ExpenseID: expenseID,
		UserID:    userID,
		Body:      strings.TrimSpace(req.Body),
	})
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}

	// Record activity
	expense, _ := h.expenseService.GetByID(expenseID, groupID)
	expenseTitle := ""
	if expense != nil {
		expenseTitle = expense.Title
	}
	h.activityService.Create(services.CreateActivityInput{
		GroupID:      groupID,
		UserID:       userID,
		ActivityType: services.ActivityCommentAdded,
		Metadata: map[string]interface{}{
			"expense_id":    expenseID,
			"comment_id":    comment.ID,
			"action":        "comment_added",
			"expense_title": expenseTitle,
		},
	})

	utils.Success(w, http.StatusCreated, comment)
}

// DeleteComment deletes a comment (only author can delete).
func (h *CommentHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	commentID := mux.Vars(r)["commentId"]

	if userID == "" {
		utils.Unauthorized(w, "User not authenticated")
		return
	}

	if err := h.commentService.Delete(commentID, userID); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	utils.Success(w, http.StatusOK, map[string]string{"message": "Comment deleted"})
}
