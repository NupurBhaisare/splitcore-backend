package services

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/nupurbhaisare/splitcore-backend/internal/database"
	"github.com/nupurbhaisare/splitcore-backend/internal/models"
)

type CommentService struct{}

func NewCommentService() *CommentService {
	return &CommentService{}
}

// CreateCommentInput is the input for creating a comment.
type CreateCommentInput struct {
	ExpenseID string
	UserID    string
	Body      string
}

// Create creates a new comment on an expense.
func (s *CommentService) Create(in CreateCommentInput) (*models.ExpenseComment, error) {
	id := uuid.New().String()
	now := time.Now()

	_, err := database.DB.Exec(
		`INSERT INTO expense_comments (id, expense_id, user_id, body, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		id, in.ExpenseID, in.UserID, in.Body, now, now,
	)
	if err != nil {
		return nil, err
	}

	return s.GetByID(id)
}

// GetByID retrieves a comment by ID.
func (s *CommentService) GetByID(id string) (*models.ExpenseComment, error) {
	row := database.DB.QueryRow(
		`SELECT ec.id, ec.expense_id, ec.user_id, ec.body, ec.created_at, ec.updated_at, ec.deleted_at,
		        u.id, u.email, u.display_name, u.avatar_url, u.created_at, u.updated_at
		 FROM expense_comments ec
		 LEFT JOIN users u ON ec.user_id = u.id
		 WHERE ec.id = ? AND ec.deleted_at IS NULL`,
		id,
	)

	var c models.ExpenseComment
	var deletedAt sql.NullTime
	var u models.User

	err := row.Scan(
		&c.ID, &c.ExpenseID, &c.UserID, &c.Body, &c.CreatedAt, &c.UpdatedAt, &deletedAt,
		&u.ID, &u.Email, &u.DisplayName, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("comment not found")
	}
	if err != nil {
		return nil, err
	}

	if deletedAt.Valid {
		c.DeletedAt = &deletedAt.Time
	}
	c.User = &u

	return &c, nil
}

// GetByExpense returns all comments for an expense.
func (s *CommentService) GetByExpense(expenseID string) ([]models.ExpenseComment, error) {
	rows, err := database.DB.Query(
		`SELECT ec.id, ec.expense_id, ec.user_id, ec.body, ec.created_at, ec.updated_at, ec.deleted_at,
		        u.id, u.email, u.display_name, u.avatar_url, u.created_at, u.updated_at
		 FROM expense_comments ec
		 LEFT JOIN users u ON ec.user_id = u.id
		 WHERE ec.expense_id = ? AND ec.deleted_at IS NULL
		 ORDER BY ec.created_at ASC`,
		expenseID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []models.ExpenseComment
	for rows.Next() {
		var c models.ExpenseComment
		var deletedAt sql.NullTime
		var u models.User

		if err := rows.Scan(
			&c.ID, &c.ExpenseID, &c.UserID, &c.Body, &c.CreatedAt, &c.UpdatedAt, &deletedAt,
			&u.ID, &u.Email, &u.DisplayName, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, err
		}

		if deletedAt.Valid {
			c.DeletedAt = &deletedAt.Time
		}
		c.User = &u
		comments = append(comments, c)
	}

	return comments, nil
}

// Delete deletes a comment (only the author can delete).
func (s *CommentService) Delete(commentID, userID string) error {
	result, err := database.DB.Exec(
		`UPDATE expense_comments SET deleted_at = ?, updated_at = ?
		 WHERE id = ? AND user_id = ? AND deleted_at IS NULL`,
		time.Now(), time.Now(), commentID, userID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("comment not found or you don't have permission to delete it")
	}

	return nil
}

// Update updates a comment body.
func (s *CommentService) Update(commentID, userID, body string) (*models.ExpenseComment, error) {
	result, err := database.DB.Exec(
		`UPDATE expense_comments SET body = ?, updated_at = ?
		 WHERE id = ? AND user_id = ? AND deleted_at IS NULL`,
		body, time.Now(), commentID, userID,
	)
	if err != nil {
		return nil, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rowsAffected == 0 {
		return nil, errors.New("comment not found or you don't have permission to edit it")
	}

	return s.GetByID(commentID)
}
