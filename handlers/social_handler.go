package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SocialHandler struct {
	DB *pgxpool.Pool
}

func NewSocialHandler(db *pgxpool.Pool) *SocialHandler {
	return &SocialHandler{
		DB: db,
	}
}

// =====================================
// CREATE GROUP
// =====================================

func (h *SocialHandler) CreateGroup(w http.ResponseWriter, r *http.Request) {

	type Request struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		CoverImage  string `json:"cover_image"`
		CreatedBy   int    `json:"created_by"`
	}

	var req Request

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(
			w,
			"Invalid request body",
			http.StatusBadRequest,
		)
		return
	}

	if req.Name == "" || req.CreatedBy <= 0 {
		http.Error(
			w,
			"Group name and creator are required",
			http.StatusBadRequest,
		)
		return
	}

	var groupID int

	err := h.DB.QueryRow(
		r.Context(),
		`INSERT INTO groups
			(name, description, cover_image, created_by)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id`,
		req.Name,
		req.Description,
		req.CoverImage,
		req.CreatedBy,
	).Scan(&groupID)

	if err != nil {
		http.Error(
			w,
			"Failed to create group: "+err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	response := map[string]interface{}{
		"id":          groupID,
		"name":        req.Name,
		"description": req.Description,
		"cover_image": req.CoverImage,
		"created_by":  req.CreatedBy,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(response)
}

// =====================================
// GET ALL GROUPS
// =====================================

func (h *SocialHandler) GetGroups(w http.ResponseWriter, r *http.Request) {

	rows, err := h.DB.Query(
		r.Context(),
		`SELECT
			id,
			name,
			description,
			cover_image,
			created_by,
			created_at
		 FROM groups
		 ORDER BY created_at DESC`,
	)

	if err != nil {
		http.Error(
			w,
			"Failed to query groups: "+err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	defer rows.Close()

	type Group struct {
		ID          int       `json:"id"`
		Name        string    `json:"name"`
		Description string    `json:"description"`
		CoverImage  string    `json:"cover_image"`
		CreatedBy   int       `json:"created_by"`
		CreatedAt   time.Time `json:"created_at"`
	}

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
			http.Error(
				w,
				"Failed to scan group: "+err.Error(),
				http.StatusInternalServerError,
			)
			return
		}

		groups = append(groups, group)
	}

	if err := rows.Err(); err != nil {
		http.Error(
			w,
			"Failed to read groups: "+err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(groups)
}

// =====================================
// JOIN GROUP
// =====================================

func (h *SocialHandler) JoinGroup(w http.ResponseWriter, r *http.Request) {

	groupID, err := strconv.Atoi(
		r.PathValue("groupId"),
	)

	if err != nil || groupID <= 0 {
		http.Error(
			w,
			"Invalid group ID",
			http.StatusBadRequest,
		)
		return
	}

	type Request struct {
		UserID int `json:"user_id"`
	}

	var req Request

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(
			w,
			"Invalid request body",
			http.StatusBadRequest,
		)
		return
	}

	if req.UserID <= 0 {
		http.Error(
			w,
			"Invalid user ID",
			http.StatusBadRequest,
		)
		return
	}

	_, err = h.DB.Exec(
		r.Context(),
		`INSERT INTO group_members
			(group_id, user_id)
		 VALUES ($1, $2)
		 ON CONFLICT (group_id, user_id)
		 DO NOTHING`,
		groupID,
		req.UserID,
	)

	if err != nil {
		http.Error(
			w,
			"Failed to join group: "+err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(
		map[string]interface{}{
			"message":  "Joined group successfully",
			"group_id": groupID,
			"user_id":  req.UserID,
		},
	)
}
// =====================================
// CHECK GROUP MEMBERSHIP
// =====================================

func (h *SocialHandler) CheckMembership(w http.ResponseWriter, r *http.Request) {

	groupID, err := strconv.Atoi(r.PathValue("groupId"))

	if err != nil || groupID <= 0 {
		http.Error(w, "Invalid group ID", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(r.URL.Query().Get("user_id"))

	if err != nil || userID <= 0 {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var isMember bool

	err = h.DB.QueryRow(
		r.Context(),
		`SELECT EXISTS (
			SELECT 1
			FROM group_members
			WHERE group_id = $1
			AND user_id = $2
		)`,
		groupID,
		userID,
	).Scan(&isMember)

	if err != nil {
		http.Error(
			w,
			"Failed to check membership: "+err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(
		map[string]interface{}{
			"group_id":  groupID,
			"user_id":   userID,
			"is_member": isMember,
		},
	)
}
// =====================================
// GET GROUP MEMBER COUNT
// =====================================

func (h *SocialHandler) GetMemberCount(w http.ResponseWriter, r *http.Request) {

	groupID, err := strconv.Atoi(r.PathValue("groupId"))

	if err != nil || groupID <= 0 {
		http.Error(w, "Invalid group ID", http.StatusBadRequest)
		return
	}

	var count int

	err = h.DB.QueryRow(
		r.Context(),
		`SELECT COUNT(*)
		 FROM group_members
		 WHERE group_id = $1`,
		groupID,
	).Scan(&count)

	if err != nil {
		http.Error(
			w,
			"Failed to get member count: "+err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(
		map[string]interface{}{
			"group_id":     groupID,
			"member_count": count,
		},
	)
}
// =====================================
// GET GROUP MEMBERS
// =====================================

func (h *SocialHandler) GetMembers(w http.ResponseWriter, r *http.Request) {

	groupID, err := strconv.Atoi(r.PathValue("groupId"))

	if err != nil || groupID <= 0 {
		http.Error(w, "Invalid group ID", http.StatusBadRequest)
		return
	}

	rows, err := h.DB.Query(
		r.Context(),
		`SELECT
			id,
			group_id,
			user_id,
			joined_at
		 FROM group_members
		 WHERE group_id = $1
		 ORDER BY joined_at ASC`,
		groupID,
	)

	if err != nil {
		http.Error(
			w,
			"Failed to get members: "+err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	defer rows.Close()

	type Member struct {
		ID       int       `json:"id"`
		GroupID  int       `json:"group_id"`
		UserID   int       `json:"user_id"`
		JoinedAt time.Time `json:"joined_at"`
	}

	members := []Member{}

	for rows.Next() {

		var member Member

		err := rows.Scan(
			&member.ID,
			&member.GroupID,
			&member.UserID,
			&member.JoinedAt,
		)

		if err != nil {
			http.Error(
				w,
				"Failed to read member: "+err.Error(),
				http.StatusInternalServerError,
			)
			return
		}

		members = append(members, member)
	}

	if err := rows.Err(); err != nil {
		http.Error(
			w,
			"Failed to read members: "+err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(members)
}
// =====================================
// GET USER GROUPS
// =====================================

func (h *SocialHandler) GetUserGroups(w http.ResponseWriter, r *http.Request) {

	userID, err := strconv.Atoi(r.PathValue("userId"))

	if err != nil || userID <= 0 {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	rows, err := h.DB.Query(
		r.Context(),
		`SELECT
			g.id,
			g.name,
			g.description,
			g.cover_image,
			g.created_by,
			g.created_at
		 FROM groups g
		 INNER JOIN group_members gm
			ON g.id = gm.group_id
		 WHERE gm.user_id = $1
		 ORDER BY gm.joined_at DESC`,
		userID,
	)

	if err != nil {
		http.Error(
			w,
			"Failed to get user groups: "+err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	defer rows.Close()

	type Group struct {
		ID          int       `json:"id"`
		Name        string    `json:"name"`
		Description string    `json:"description"`
		CoverImage  string    `json:"cover_image"`
		CreatedBy   int       `json:"created_by"`
		CreatedAt   time.Time `json:"created_at"`
	}

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
			http.Error(
				w,
				"Failed to read group: "+err.Error(),
				http.StatusInternalServerError,
			)
			return
		}

		groups = append(groups, group)
	}

	if err := rows.Err(); err != nil {
		http.Error(
			w,
			"Failed to read groups: "+err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(groups)
}