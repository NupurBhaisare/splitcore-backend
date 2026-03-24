package services

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/splitcore/backend/internal/database"
	"github.com/splitcore/backend/internal/models"
)

type GroupService struct{}

func NewGroupService() *GroupService {
	return &GroupService{}
}

func (s *GroupService) Create(userID, name, description, iconEmoji, currencyCode string) (*models.Group, error) {
	group := &models.Group{
		ID:            uuid.New().String(),
		Name:          name,
		Description:   description,
		IconEmoji:     iconEmoji,
		CurrencyCode:  currencyCode,
		CreatedByUser: userID,
		InviteCode:    generateInviteCode(),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if group.CurrencyCode == "" {
		group.CurrencyCode = "USD"
	}
	if group.IconEmoji == "" {
		group.IconEmoji = "💰"
	}

	tx, err := database.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`INSERT INTO groups (id, name, description, icon_emoji, currency_code, created_by_user_id, invite_code, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		group.ID, group.Name, group.Description, group.IconEmoji, group.CurrencyCode,
		group.CreatedByUser, group.InviteCode, group.CreatedAt, group.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Add creator as owner member
	memberID := uuid.New().String()
	_, err = tx.Exec(
		`INSERT INTO group_members (id, group_id, user_id, nickname_in_group, role, joined_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		memberID, group.ID, userID, "", "owner", time.Now(),
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return group, nil
}

func (s *GroupService) GetByID(id string) (*models.Group, error) {
	group := &models.Group{}
	var deletedAt sql.NullTime
	err := database.DB.QueryRow(
		`SELECT id, name, description, icon_emoji, currency_code, created_by_user_id, invite_code, created_at, updated_at, deleted_at
		 FROM groups WHERE id = ? AND deleted_at IS NULL`,
		id,
	).Scan(&group.ID, &group.Name, &group.Description, &group.IconEmoji, &group.CurrencyCode,
		&group.CreatedByUser, &group.InviteCode, &group.CreatedAt, &group.UpdatedAt, &deletedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("group not found")
	}
	if err != nil {
		return nil, err
	}

	if deletedAt.Valid {
		group.DeletedAt = &deletedAt.Time
	}

	return group, nil
}

func (s *GroupService) GetUserGroups(userID string) ([]models.Group, error) {
	rows, err := database.DB.Query(
		`SELECT g.id, g.name, g.description, g.icon_emoji, g.currency_code, g.created_by_user_id,
		        g.invite_code, g.created_at, g.updated_at
		 FROM groups g
		 INNER JOIN group_members gm ON g.id = gm.group_id
		 WHERE gm.user_id = ? AND gm.removed_at IS NULL AND g.deleted_at IS NULL
		 ORDER BY g.created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []models.Group
	for rows.Next() {
		var g models.Group
		if err := rows.Scan(&g.ID, &g.Name, &g.Description, &g.IconEmoji, &g.CurrencyCode,
			&g.CreatedByUser, &g.InviteCode, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}

	return groups, nil
}

func (s *GroupService) Update(id, userID, name, description, iconEmoji, currencyCode string) (*models.Group, error) {
	// Verify user is owner or member
	if !s.IsMember(id, userID) {
		return nil, errors.New("not a member of this group")
	}

	_, err := database.DB.Exec(
		`UPDATE groups SET name = COALESCE(NULLIF(?, ''), name),
		 description = COALESCE(NULLIF(?, ''), description),
		 icon_emoji = COALESCE(NULLIF(?, ''), icon_emoji),
		 currency_code = COALESCE(NULLIF(?, ''), currency_code),
		 updated_at = ?
		 WHERE id = ? AND deleted_at IS NULL`,
		name, description, iconEmoji, currencyCode, time.Now(), id,
	)
	if err != nil {
		return nil, err
	}

	return s.GetByID(id)
}

func (s *GroupService) Delete(id, userID string) error {
	group, err := s.GetByID(id)
	if err != nil {
		return err
	}

	if group.CreatedByUser != userID {
		return errors.New("only the group creator can delete the group")
	}

	_, err = database.DB.Exec(
		`UPDATE groups SET deleted_at = ?, updated_at = ? WHERE id = ?`,
		time.Now(), time.Now(), id,
	)
	return err
}

func (s *GroupService) AddMember(groupID, userID, nickname, inviterUserID string) (*models.GroupMember, error) {
	// Check if group exists
	if _, err := s.GetByID(groupID); err != nil {
		return nil, err
	}

	// Check if already a member
	var existingID string
	err := database.DB.QueryRow(
		`SELECT id FROM group_members WHERE group_id = ? AND user_id = ? AND removed_at IS NULL`,
		groupID, userID,
	).Scan(&existingID)
	if err == nil {
		return nil, errors.New("user is already a member of this group")
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	// Check if user exists and get their display name
	userService := NewUserService()
	user, err := userService.GetByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	member := &models.GroupMember{
		ID:       uuid.New().String(),
		GroupID:  groupID,
		UserID:   userID,
		Nickname: nickname,
		Role:     "member",
		JoinedAt: time.Now(),
	}

	if member.Nickname == "" {
		member.Nickname = user.DisplayName
	}

	_, err = database.DB.Exec(
		`INSERT INTO group_members (id, group_id, user_id, nickname_in_group, role, joined_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		member.ID, member.GroupID, member.UserID, member.Nickname, member.Role, member.JoinedAt,
	)
	if err != nil {
		return nil, err
	}

	return member, nil
}

func (s *GroupService) RemoveMember(groupID, userID, requesterUserID string) error {
	// Check if requester is owner or removing themselves
	group, err := s.GetByID(groupID)
	if err != nil {
		return err
	}

	if requesterUserID != userID && group.CreatedByUser != requesterUserID {
		return errors.New("only the owner or the member themselves can remove a member")
	}

	if group.CreatedByUser == userID {
		return errors.New("cannot remove the group creator")
	}

	_, err = database.DB.Exec(
		`UPDATE group_members SET removed_at = ? WHERE group_id = ? AND user_id = ? AND removed_at IS NULL`,
		time.Now(), groupID, userID,
	)
	return err
}

func (s *GroupService) GetMembers(groupID string) ([]models.GroupMember, error) {
	rows, err := database.DB.Query(
		`SELECT gm.id, gm.group_id, gm.user_id, gm.nickname_in_group, gm.role, gm.joined_at,
		        u.id, u.email, u.display_name, u.avatar_url, u.created_at, u.updated_at
		 FROM group_members gm
		 INNER JOIN users u ON gm.user_id = u.id
		 WHERE gm.group_id = ? AND gm.removed_at IS NULL AND u.deleted_at IS NULL
		 ORDER BY gm.joined_at ASC`,
		groupID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []models.GroupMember
	for rows.Next() {
		var m models.GroupMember
		var u models.User
		if err := rows.Scan(&m.ID, &m.GroupID, &m.UserID, &m.Nickname, &m.Role, &m.JoinedAt,
			&u.ID, &u.Email, &u.DisplayName, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		m.User = &u
		members = append(members, m)
	}

	return members, nil
}

func (s *GroupService) IsMember(groupID, userID string) bool {
	var id string
	err := database.DB.QueryRow(
		`SELECT id FROM group_members WHERE group_id = ? AND user_id = ? AND removed_at IS NULL`,
		groupID, userID,
	).Scan(&id)
	return err == nil
}

func (s *GroupService) IsOwner(groupID, userID string) bool {
	var id string
	err := database.DB.QueryRow(
		`SELECT id FROM group_members WHERE group_id = ? AND user_id = ? AND role = 'owner' AND removed_at IS NULL`,
		groupID, userID,
	).Scan(&id)
	return err == nil
}

func (s *GroupService) JoinByInviteCode(inviteCode, userID string) (*models.Group, error) {
	group := &models.Group{}
	err := database.DB.QueryRow(
		`SELECT id, name, description, icon_emoji, currency_code, created_by_user_id, invite_code, created_at, updated_at
		 FROM groups WHERE invite_code = ? AND deleted_at IS NULL`,
		inviteCode,
	).Scan(&group.ID, &group.Name, &group.Description, &group.IconEmoji, &group.CurrencyCode,
		&group.CreatedByUser, &group.InviteCode, &group.CreatedAt, &group.UpdatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("invalid invite code")
	}
	if err != nil {
		return nil, err
	}

	_, err = s.AddMember(group.ID, userID, "", userID)
	if err != nil {
		return nil, err
	}

	return group, nil
}

func generateInviteCode() string {
	// Generate a short, readable invite code
	id := uuid.New().String()
	// Take first 8 chars and uppercase
	code := id[:8]
	return code
}
