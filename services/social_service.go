package services

import (
	"context"
	"errors"
	"strings"
	"time"
	
	"github.com/jackc/pgx/v5/pgxpool"
)

type SocialService struct {
	DB *pgxpool.Pool
}

func NewSocialService(db *pgxpool.Pool) *SocialService {
	return &SocialService{
		DB: db,
	}
}

type CreateGroupInput struct {
	Name        string
	Description string
	CoverImage  string
	CreatedBy   int
}

type Group struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CoverImage  string `json:"cover_image"`
	CreatedBy   int    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
}

// CreateGroup creates a new BetPay community group.
func (s *SocialService) CreateGroup(
	ctx context.Context,
	input CreateGroupInput,
) (*Group, error) {

	input.Name = strings.TrimSpace(input.Name)

	if input.Name == "" {
		return nil, errors.New("group name is required")
	}

	if input.CreatedBy <= 0 {
		return nil, errors.New("invalid creator")
	}

	var group Group

	err := s.DB.QueryRow(
		ctx,
		`INSERT INTO groups
			(name, description, cover_image, created_by)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, name, description, cover_image, created_by, created_at`,
		input.Name,
		input.Description,
		input.CoverImage,
		input.CreatedBy,
	).Scan(
		&group.ID,
		&group.Name,
		&group.Description,
		&group.CoverImage,
		&group.CreatedBy,
		&group.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &group, nil
}

// GetGroups returns all community groups.
func (s *SocialService) GetGroups(
	ctx context.Context,
) ([]Group, error) {

	rows, err := s.DB.Query(
		ctx,
		`SELECT id, name, description, cover_image, created_by, created_at
		 FROM groups
		 ORDER BY created_at DESC`,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	groups := []Group{}

	for rows.Next() {
		var group Group

		err := rows.Scan(
			&group.ID,
			&group.Name,
			&group.Description,
			&group.CoverImage,
			&group.CreatedBy,
			&group.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		groups = append(groups, group)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return groups, nil
}

// JoinGroup adds a user to a group.
func (s *SocialService) JoinGroup(
	ctx context.Context,
	groupID int,
	userID int,
) error {

	if groupID <= 0 {
		return errors.New("invalid group ID")
	}

	if userID <= 0 {
		return errors.New("invalid user ID")
	}

	_, err := s.DB.Exec(
		ctx,
		`INSERT INTO group_members (group_id, user_id)
		 VALUES ($1, $2)
		 ON CONFLICT (group_id, user_id) DO NOTHING`,
		groupID,
		userID,
	)

	return err
}

// LeaveGroup removes a user from a group.
func (s *SocialService) LeaveGroup(
	ctx context.Context,
	groupID int,
	userID int,
) error {

	if groupID <= 0 {
		return errors.New("invalid group ID")
	}

	if userID <= 0 {
		return errors.New("invalid user ID")
	}

	_, err := s.DB.Exec(
		ctx,
		`DELETE FROM group_members
		 WHERE group_id = $1 AND user_id = $2`,
		groupID,
		userID,
	)

	return err
}

// IsMember checks whether a user belongs to a group.
func (s *SocialService) IsMember(
	ctx context.Context,
	groupID int,
	userID int,
) (bool, error) {

	var exists bool

	err := s.DB.QueryRow(
		ctx,
		`SELECT EXISTS (
			SELECT 1
			FROM group_members
			WHERE group_id = $1 AND user_id = $2
		)`,
		groupID,
		userID,
	).Scan(&exists)

	return exists, err
}