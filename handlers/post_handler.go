package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostHandler struct {
	DB *pgxpool.Pool
}

func NewPostHandler(db *pgxpool.Pool) *PostHandler {
	return &PostHandler{
		DB: db,
	}
}

// =====================================
// CREATE POST
// =====================================

func (h *PostHandler) CreatePost(w http.ResponseWriter, r *http.Request) {

	groupID, err := strconv.Atoi(r.PathValue("groupId"))

	if err != nil || groupID <= 0 {
		http.Error(w, "Invalid group ID", http.StatusBadRequest)
		return
	}

	type Request struct {
		UserID   int    `json:"user_id"`
		Content  string `json:"content"`
		ImageURL string `json:"image_url"`
	}

	var req Request

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req.Content = strings.TrimSpace(req.Content)

	if req.UserID <= 0 {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		http.Error(w, "Post content is required", http.StatusBadRequest)
		return
	}

	var postID int

	err = h.DB.QueryRow(
		r.Context(),
		`INSERT INTO posts
			(group_id, user_id, content, image_url)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id`,
		groupID,
		req.UserID,
		req.Content,
		req.ImageURL,
	).Scan(&postID)

	if err != nil {
		http.Error(
			w,
			"Failed to create post: "+err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	response := map[string]interface{}{
		"id":         postID,
		"group_id":   groupID,
		"user_id":    req.UserID,
		"content":    req.Content,
		"image_url":  req.ImageURL,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(response)
}
// =====================================
// GET POSTS FOR A GROUP
// =====================================

func (h *PostHandler) GetGroupPosts(w http.ResponseWriter, r *http.Request) {

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
			content,
			image_url,
			created_at
		 FROM posts
		 WHERE group_id = $1
		 ORDER BY created_at DESC`,
		groupID,
	)

	if err != nil {
		http.Error(
			w,
			"Failed to get posts: "+err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	defer rows.Close()

	type Post struct {
		ID        int       `json:"id"`
		GroupID   int       `json:"group_id"`
		UserID    int       `json:"user_id"`
		Content   string    `json:"content"`
		ImageURL  string    `json:"image_url"`
		CreatedAt time.Time `json:"created_at"`
	}

	posts := []Post{}

	for rows.Next() {

		var post Post

		err := rows.Scan(
			&post.ID,
			&post.GroupID,
			&post.UserID,
			&post.Content,
			&post.ImageURL,
			&post.CreatedAt,
		)

		if err != nil {
			http.Error(
				w,
				"Failed to read post: "+err.Error(),
				http.StatusInternalServerError,
			)
			return
		}

		posts = append(posts, post)
	}

	if err := rows.Err(); err != nil {
		http.Error(
			w,
			"Failed to read posts: "+err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(posts)
}
// =====================================
// LIKE POST
// =====================================

func (h *PostHandler) LikePost(w http.ResponseWriter, r *http.Request) {

	postID, err := strconv.Atoi(r.PathValue("postId"))

	if err != nil || postID <= 0 {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	type Request struct {
		UserID int `json:"user_id"`
	}

	var req Request

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserID <= 0 {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	_, err = h.DB.Exec(
		r.Context(),
		`INSERT INTO likes (post_id, user_id)
		 VALUES ($1, $2)
		 ON CONFLICT (post_id, user_id)
		 DO NOTHING`,
		postID,
		req.UserID,
	)

	if err != nil {
		http.Error(
			w,
			"Failed to like post: "+err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(
		map[string]interface{}{
			"message": "Post liked successfully",
			"post_id": postID,
			"user_id": req.UserID,
		},
	)
}
// =====================================
// CREATE COMMENT
// =====================================

func (h *PostHandler) CreateComment(w http.ResponseWriter, r *http.Request) {

	postID, err := strconv.Atoi(r.PathValue("postId"))

	if err != nil || postID <= 0 {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	type Request struct {
		UserID  int    `json:"user_id"`
		Content string `json:"content"`
	}

	var req Request

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req.Content = strings.TrimSpace(req.Content)

	if req.UserID <= 0 {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		http.Error(w, "Comment content is required", http.StatusBadRequest)
		return
	}

	var commentID int

	err = h.DB.QueryRow(
		r.Context(),
		`INSERT INTO comments
			(post_id, user_id, content)
		 VALUES ($1, $2, $3)
		 RETURNING id`,
		postID,
		req.UserID,
		req.Content,
	).Scan(&commentID)

	if err != nil {
		http.Error(
			w,
			"Failed to create comment: "+err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(
		map[string]interface{}{
			"id":       commentID,
			"post_id":  postID,
			"user_id":  req.UserID,
			"content":  req.Content,
		},
	)
}
// =====================================
// GET COMMENTS FOR A POST
// =====================================

func (h *PostHandler) GetPostComments(w http.ResponseWriter, r *http.Request) {

	postID, err := strconv.Atoi(r.PathValue("postId"))

	if err != nil || postID <= 0 {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	rows, err := h.DB.Query(
		r.Context(),
		`SELECT
			id,
			post_id,
			user_id,
			content,
			created_at
		 FROM comments
		 WHERE post_id = $1
		 ORDER BY created_at ASC`,
		postID,
	)

	if err != nil {
		http.Error(
			w,
			"Failed to get comments: "+err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	defer rows.Close()

	type Comment struct {
		ID        int       `json:"id"`
		PostID    int       `json:"post_id"`
		UserID    int       `json:"user_id"`
		Content   string    `json:"content"`
		CreatedAt time.Time `json:"created_at"`
	}

	comments := []Comment{}

	for rows.Next() {

		var comment Comment

		err := rows.Scan(
			&comment.ID,
			&comment.PostID,
			&comment.UserID,
			&comment.Content,
			&comment.CreatedAt,
		)

		if err != nil {
			http.Error(
				w,
				"Failed to read comment: "+err.Error(),
				http.StatusInternalServerError,
			)
			return
		}

		comments = append(comments, comment)
	}

	if err := rows.Err(); err != nil {
		http.Error(
			w,
			"Failed to read comments: "+err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(comments)
}
// =====================================
// UNLIKE POST
// =====================================

func (h *PostHandler) UnlikePost(w http.ResponseWriter, r *http.Request) {

	postID, err := strconv.Atoi(r.PathValue("postId"))

	if err != nil || postID <= 0 {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	type Request struct {
		UserID int `json:"user_id"`
	}

	var req Request

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserID <= 0 {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	_, err = h.DB.Exec(
		r.Context(),
		`DELETE FROM likes
		 WHERE post_id = $1 AND user_id = $2`,
		postID,
		req.UserID,
	)

	if err != nil {
		http.Error(
			w,
			"Failed to unlike post: "+err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(
		map[string]interface{}{
			"message": "Post unliked successfully",
			"post_id": postID,
			"user_id": req.UserID,
		},
	)
}
// =====================================
// GET LIKE COUNT
// =====================================

func (h *PostHandler) GetLikeCount(w http.ResponseWriter, r *http.Request) {

	postID, err := strconv.Atoi(r.PathValue("postId"))

	if err != nil || postID <= 0 {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	var count int

	err = h.DB.QueryRow(
		r.Context(),
		`SELECT COUNT(*)
		 FROM likes
		 WHERE post_id = $1`,
		postID,
	).Scan(&count)

	if err != nil {
		http.Error(
			w,
			"Failed to get like count: "+err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(
		map[string]interface{}{
			"post_id":    postID,
			"like_count": count,
		},
	)
}
// =====================================
// CHECK IF USER LIKED POST
// =====================================

func (h *PostHandler) CheckLike(w http.ResponseWriter, r *http.Request) {

	postID, err := strconv.Atoi(r.PathValue("postId"))

	if err != nil || postID <= 0 {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(r.URL.Query().Get("user_id"))

	if err != nil || userID <= 0 {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var liked bool

	err = h.DB.QueryRow(
		r.Context(),
		`SELECT EXISTS (
			SELECT 1
			FROM likes
			WHERE post_id = $1
			AND user_id = $2
		)`,
		postID,
		userID,
	).Scan(&liked)

	if err != nil {
		http.Error(
			w,
			"Failed to check like: "+err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(
		map[string]interface{}{
			"post_id": postID,
			"user_id": userID,
			"liked":   liked,
		},
	)
}
// =====================================
// LEAVE GROUP
// =====================================

func (h *SocialHandler) LeaveGroup(w http.ResponseWriter, r *http.Request) {

	groupID, err := strconv.Atoi(r.PathValue("groupId"))

	if err != nil || groupID <= 0 {
		http.Error(w, "Invalid group ID", http.StatusBadRequest)
		return
	}

	type Request struct {
		UserID int `json:"user_id"`
	}

	var req Request

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserID <= 0 {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	_, err = h.DB.Exec(
		r.Context(),
		`DELETE FROM group_members
		 WHERE group_id = $1
		 AND user_id = $2`,
		groupID,
		req.UserID,
	)

	if err != nil {
		http.Error(
			w,
			"Failed to leave group: "+err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(
		map[string]interface{}{
			"message":  "Left group successfully",
			"group_id": groupID,
			"user_id":  req.UserID,
		},
	)
}
// =====================================
// DELETE COMMENT
// =====================================

func (h *PostHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {

	commentID, err := strconv.Atoi(r.PathValue("commentId"))

	if err != nil || commentID <= 0 {
		http.Error(w, "Invalid comment ID", http.StatusBadRequest)
		return
	}

	_, err = h.DB.Exec(
		r.Context(),
		`DELETE FROM comments
		 WHERE id = $1`,
		commentID,
	)

	if err != nil {
		http.Error(
			w,
			"Failed to delete comment: "+err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(
		map[string]interface{}{
			"message":    "Comment deleted successfully",
			"comment_id": commentID,
		},
	)
}
// =====================================
// DELETE POST
// =====================================

func (h *PostHandler) DeletePost(w http.ResponseWriter, r *http.Request) {

	postID, err := strconv.Atoi(r.PathValue("postId"))

	if err != nil || postID <= 0 {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	_, err = h.DB.Exec(
		r.Context(),
		`DELETE FROM posts
		 WHERE id = $1`,
		postID,
	)

	if err != nil {
		http.Error(
			w,
			"Failed to delete post: "+err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(
		map[string]interface{}{
			"message": "Post deleted successfully",
			"post_id": postID,
		},
	)
}
// =====================================
// UPDATE POST
// =====================================

func (h *PostHandler) UpdatePost(w http.ResponseWriter, r *http.Request) {

	postID, err := strconv.Atoi(r.PathValue("postId"))

	if err != nil || postID <= 0 {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	type Request struct {
		Content  string `json:"content"`
		ImageURL string `json:"image_url"`
	}

	var req Request

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req.Content = strings.TrimSpace(req.Content)

	if req.Content == "" {
		http.Error(w, "Post content is required", http.StatusBadRequest)
		return
	}

	var updatedPost struct {
		ID        int       `json:"id"`
		GroupID   int       `json:"group_id"`
		UserID    int       `json:"user_id"`
		Content   string    `json:"content"`
		ImageURL  string    `json:"image_url"`
		CreatedAt time.Time `json:"created_at"`
	}

	err = h.DB.QueryRow(
		r.Context(),
		`UPDATE posts
		 SET content = $1,
		     image_url = $2
		 WHERE id = $3
		 RETURNING id, group_id, user_id, content, image_url, created_at`,
		req.Content,
		req.ImageURL,
		postID,
	).Scan(
		&updatedPost.ID,
		&updatedPost.GroupID,
		&updatedPost.UserID,
		&updatedPost.Content,
		&updatedPost.ImageURL,
		&updatedPost.CreatedAt,
	)

	if err != nil {
		http.Error(
			w,
			"Failed to update post: "+err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(updatedPost)
}
// =====================================
// UPDATE COMMENT
// =====================================

func (h *PostHandler) UpdateComment(w http.ResponseWriter, r *http.Request) {

	commentID, err := strconv.Atoi(r.PathValue("commentId"))

	if err != nil || commentID <= 0 {
		http.Error(w, "Invalid comment ID", http.StatusBadRequest)
		return
	}

	type Request struct {
		Content string `json:"content"`
	}

	var req Request

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req.Content = strings.TrimSpace(req.Content)

	if req.Content == "" {
		http.Error(w, "Comment content is required", http.StatusBadRequest)
		return
	}

	var updatedComment struct {
		ID        int       `json:"id"`
		PostID    int       `json:"post_id"`
		UserID    int       `json:"user_id"`
		Content   string    `json:"content"`
		CreatedAt time.Time `json:"created_at"`
	}

	err = h.DB.QueryRow(
		r.Context(),
		`UPDATE comments
		 SET content = $1
		 WHERE id = $2
		 RETURNING id, post_id, user_id, content, created_at`,
		req.Content,
		commentID,
	).Scan(
		&updatedComment.ID,
		&updatedComment.PostID,
		&updatedComment.UserID,
		&updatedComment.Content,
		&updatedComment.CreatedAt,
	)

	if err != nil {
		http.Error(
			w,
			"Failed to update comment: "+err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(updatedComment)
}
// =====================================
// GET USER POSTS
// =====================================

func (h *PostHandler) GetUserPosts(w http.ResponseWriter, r *http.Request) {

	userID, err := strconv.Atoi(r.PathValue("userId"))

	if err != nil || userID <= 0 {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	rows, err := h.DB.Query(
		r.Context(),
		`SELECT
			id,
			group_id,
			user_id,
			content,
			image_url,
			created_at
		 FROM posts
		 WHERE user_id = $1
		 ORDER BY created_at DESC`,
		userID,
	)

	if err != nil {
		http.Error(
			w,
			"Failed to get user posts: "+err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	defer rows.Close()

	type Post struct {
		ID        int       `json:"id"`
		GroupID   int       `json:"group_id"`
		UserID    int       `json:"user_id"`
		Content   string    `json:"content"`
		ImageURL  string    `json:"image_url"`
		CreatedAt time.Time `json:"created_at"`
	}

	posts := []Post{}

	for rows.Next() {

		var post Post

		err := rows.Scan(
			&post.ID,
			&post.GroupID,
			&post.UserID,
			&post.Content,
			&post.ImageURL,
			&post.CreatedAt,
		)

		if err != nil {
			http.Error(
				w,
				"Failed to read post: "+err.Error(),
				http.StatusInternalServerError,
			)
			return
		}

		posts = append(posts, post)
	}

	if err := rows.Err(); err != nil {
		http.Error(
			w,
			"Failed to read posts: "+err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(posts)
}
// =====================================
// GET USER COMMENTS
// =====================================

func (h *PostHandler) GetUserComments(w http.ResponseWriter, r *http.Request) {

	userID, err := strconv.Atoi(r.PathValue("userId"))

	if err != nil || userID <= 0 {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	rows, err := h.DB.Query(
		r.Context(),
		`SELECT
			id,
			post_id,
			user_id,
			content,
			created_at
		 FROM comments
		 WHERE user_id = $1
		 ORDER BY created_at DESC`,
		userID,
	)

	if err != nil {
		http.Error(
			w,
			"Failed to get user comments: "+err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	defer rows.Close()

	type Comment struct {
		ID        int       `json:"id"`
		PostID    int       `json:"post_id"`
		UserID    int       `json:"user_id"`
		Content   string    `json:"content"`
		CreatedAt time.Time `json:"created_at"`
	}

	comments := []Comment{}

	for rows.Next() {

		var comment Comment

		err := rows.Scan(
			&comment.ID,
			&comment.PostID,
			&comment.UserID,
			&comment.Content,
			&comment.CreatedAt,
		)

		if err != nil {
			http.Error(
				w,
				"Failed to read comment: "+err.Error(),
				http.StatusInternalServerError,
			)
			return
		}

		comments = append(comments, comment)
	}

	if err := rows.Err(); err != nil {
		http.Error(
			w,
			"Failed to read comments: "+err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(comments)
}