package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"betpay_mobile_api/middleware"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MeHandler struct {
	DB *pgxpool.Pool
}

func NewMeHandler(db *pgxpool.Pool) *MeHandler {
	return &MeHandler{
		DB: db,
	}
}

func (h *MeHandler) GetMe(
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	// Get user ID from JWT middleware
	userIDString := r.Context().Value(
		middleware.UserIDKey,
	)

	if userIDString == nil {
		http.Error(
			w,
			"User ID not found",
			http.StatusUnauthorized,
		)
		return
	}

	userID, ok := userIDString.(string)

	if !ok {
		http.Error(
			w,
			"Invalid user ID",
			http.StatusUnauthorized,
		)
		return
	}

	userIDInt, err := strconv.Atoi(userID)

	if err != nil {
		http.Error(
			w,
			"Invalid user ID",
			http.StatusUnauthorized,
		)
		return
	}

	var user UserResponse

	query := `
		SELECT
			id,
			full_name,
			email,
			phone,
			role
		FROM users
		WHERE id = $1
		LIMIT 1
	`

	err = h.DB.QueryRow(
		context.Background(),
		query,
		userIDInt,
	).Scan(
		&user.ID,
		&user.FullName,
		&user.Email,
		&user.Phone,
		&user.Role,
	)

	if err != nil {
		http.Error(
			w,
			"User not found",
			http.StatusNotFound,
		)
		return
	}

	json.NewEncoder(w).Encode(
		map[string]interface{}{
			"status": "success",
			"user":   user,
		},
	)
}