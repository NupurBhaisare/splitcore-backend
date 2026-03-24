package services

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/nupurbhaisare/splitcore-backend/internal/database"
	"github.com/nupurbhaisare/splitcore-backend/internal/models"
)

// ActivityService handles activity feed operations.
type ActivityService struct{}

func NewActivityService() *ActivityService {
	return &ActivityService{}
}

// ActivityType constants.
const (
	ActivityExpenseAdded   = "expense_added"
	ActivityExpenseEdited  = "expense_edited"
	ActivityExpenseDeleted = "expense_deleted"
	ActivityMemberJoined   = "member_joined"
	ActivityMemberLeft     = "member_left"
	ActivitySettlementMade = "settlement_made"
	ActivityGroupCreated   = "group_created"
	ActivityCommentAdded   = "comment_added"
)

// CreateActivityInput is the input for creating an activity.
type CreateActivityInput struct {
	GroupID       string
	UserID        string // actor
	TargetUserIDs []string
	ActivityType  string
	Metadata      map[string]interface{}
}

// Create records a new activity feed entry.
func (s *ActivityService) Create(in CreateActivityInput) (*models.Activity, error) {
	id := uuid.New().String()
	createdAt := time.Now()

	targetJSON, _ := json.Marshal(in.TargetUserIDs)
	metadataJSON, _ := json.Marshal(in.Metadata)
	if metadataJSON == nil {
		metadataJSON = []byte("{}")
	}

	_, err := database.DB.Exec(
		`INSERT INTO activities
		(id, group_id, user_id, target_user_ids, activity_type, metadata, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, in.GroupID, in.UserID, string(targetJSON), in.ActivityType, string(metadataJSON), createdAt,
	)
	if err != nil {
		return nil, err
	}

	return s.GetByID(id)
}

// GetByID retrieves an activity by ID.
func (s *ActivityService) GetByID(id string) (*models.Activity, error) {
	row := database.DB.QueryRow(
		`SELECT a.id, a.group_id, a.user_id, a.target_user_ids, a.activity_type,
		        a.metadata, a.created_at, a.read_at,
		        u.display_name, u.email, u.avatar_url
		 FROM activities a
		 LEFT JOIN users u ON a.user_id = u.id
		 WHERE a.id = ?`,
		id,
	)

	var activity models.Activity
	var targetUserIDsJSON, metadataJSON string
	var userName, userEmail, userAvatar string

	err := row.Scan(
		&activity.ID, &activity.GroupID, &activity.UserID,
		&targetUserIDsJSON, &activity.ActivityType,
		&metadataJSON, &activity.CreatedAt, &activity.ReadAt,
		&userName, &userEmail, &userAvatar,
	)
	if err != nil {
		return nil, err
	}

	json.Unmarshal([]byte(targetUserIDsJSON), &activity.TargetUserIDs)
	json.Unmarshal([]byte(metadataJSON), &activity.Metadata)

	if userName != "" {
		activity.User = &models.User{
			ID:          activity.UserID,
			DisplayName: userName,
			Email:       userEmail,
			AvatarURL:   userAvatar,
		}
	}

	return &activity, nil
}

// GetByGroup returns paginated activities for a group.
func (s *ActivityService) GetByGroup(groupID string, page, perPage int) ([]models.Activity, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	// Count total
	var total int
	countRow := database.DB.QueryRow(
		`SELECT COUNT(*) FROM activities WHERE group_id = ?`,
		groupID,
	)
	countRow.Scan(&total)

	rows, err := database.DB.Query(
		`SELECT a.id, a.group_id, a.user_id, a.target_user_ids, a.activity_type,
		        a.metadata, a.created_at, a.read_at,
		        u.display_name, u.email, u.avatar_url
		 FROM activities a
		 LEFT JOIN users u ON a.user_id = u.id
		 WHERE a.group_id = ?
		 ORDER BY a.created_at DESC
		 LIMIT ? OFFSET ?`,
		groupID, perPage, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var activities []models.Activity
	for rows.Next() {
		var activity models.Activity
		var targetUserIDsJSON, metadataJSON string
		var userName, userEmail, userAvatar string

		err := rows.Scan(
			&activity.ID, &activity.GroupID, &activity.UserID,
			&targetUserIDsJSON, &activity.ActivityType,
			&metadataJSON, &activity.CreatedAt, &activity.ReadAt,
			&userName, &userEmail, &userAvatar,
		)
		if err != nil {
			return nil, 0, err
		}

		json.Unmarshal([]byte(targetUserIDsJSON), &activity.TargetUserIDs)
		json.Unmarshal([]byte(metadataJSON), &activity.Metadata)

		if userName != "" {
			activity.User = &models.User{
				ID:          activity.UserID,
				DisplayName: userName,
				Email:       userEmail,
				AvatarURL:   userAvatar,
			}
		}

		activities = append(activities, activity)
	}

	return activities, total, nil
}

// GetUnreadCount returns the count of unread activities for a user across all their groups.
func (s *ActivityService) GetUnreadCount(userID string) (int, error) {
	rows, err := database.DB.Query(
		`SELECT COUNT(*) FROM activities a
		 INNER JOIN group_members gm ON a.group_id = gm.group_id
		 WHERE gm.user_id = ? AND gm.removed_at IS NULL
		   AND (a.read_at IS NULL OR a.read_at > a.created_at)`,
		userID,
	)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var count int
	if rows.Next() {
		rows.Scan(&count)
	}
	return count, nil
}

// MarkRead marks all activities in a group as read for a user.
func (s *ActivityService) MarkRead(groupID, userID string) error {
	_, err := database.DB.Exec(
		`UPDATE activities SET read_at = ? WHERE group_id = ?`,
		time.Now(), groupID,
	)
	return err
}
