package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type LoginHandler struct {
	DB *pgxpool.Pool
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Status  string       `json:"status"`
	Message string       `json:"message"`
	Token   string       `json:"token"`
	User    UserResponse `json:"user"`
}

type UserResponse struct {
	ID       int    `json:"id"`
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Role     string `json:"role"`
}

func NewLoginHandler(db *pgxpool.Pool) *LoginHandler {
	return &LoginHandler{
		DB: db,
	}
}

func (h *LoginHandler) Login(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		http.Error(
			w,
			"Method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	var request LoginRequest

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(
			w,
			"Invalid request body",
			http.StatusBadRequest,
		)
		return
	}

	email := strings.TrimSpace(
		strings.ToLower(request.Email),
	)

	password := request.Password

	if email == "" || password == "" {
		http.Error(
			w,
			"Email and password are required",
			http.StatusBadRequest,
		)
		return
	}

	var user UserResponse
	var passwordHash string

	query := `
		SELECT
			id,
			full_name,
			email,
			phone,
			password_hash,
			role
		FROM users
		WHERE LOWER(email) = $1
		LIMIT 1
	`

	err = h.DB.QueryRow(
		context.Background(),
		query,
		email,
	).Scan(
		&user.ID,
		&user.FullName,
		&user.Email,
		&user.Phone,
		&passwordHash,
		&user.Role,
	)

	if err != nil {
		http.Error(
			w,
			"Invalid email or password",
			http.StatusUnauthorized,
		)
		return
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(passwordHash),
		[]byte(password),
	)

	if err != nil {
		http.Error(
			w,
			"Invalid email or password",
			http.StatusUnauthorized,
		)
		return
	}

	jwtSecret := os.Getenv("JWT_SECRET")

	if jwtSecret == "" {
		http.Error(
			w,
			"JWT_SECRET is not configured",
			http.StatusInternalServerError,
		)
		return
	}

	claims := jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"role":    user.Role,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)

	signedToken, err := token.SignedString(
		[]byte(jwtSecret),
	)

	if err != nil {
		http.Error(
			w,
			"Failed to create authentication token",
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	response := LoginResponse{
		Status:  "success",
		Message: "Login successful",
		Token:   signedToken,
		User:    user,
	}

	json.NewEncoder(w).Encode(response)
}