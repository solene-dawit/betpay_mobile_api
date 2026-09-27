package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// ==============================
// REGISTER REQUEST
// ==============================

type RegisterRequest struct {
	FullName        string `json:"full_name"`
	Email           string `json:"email"`
	Phone           string `json:"phone"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
	Role            string `json:"role"`
}

// ==============================
// REGISTER RESPONSE
// ==============================

type RegisterResponse struct {
	Message string `json:"message"`
	UserID  int    `json:"user_id"`
	Role    string `json:"role"`
}

// ==============================
// REGISTER HANDLER
// ==============================

func RegisterHandler(db *pgxpool.Pool) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		// --------------------------------
		// Only POST is allowed
		// --------------------------------

		if r.Method != http.MethodPost {
			sendError(
				w,
				http.StatusMethodNotAllowed,
				"Only POST requests are allowed",
			)
			return
		}

		// --------------------------------
		// Read JSON
		// --------------------------------

		var request RegisterRequest

		err := json.NewDecoder(r.Body).Decode(&request)

		if err != nil {
			sendError(
				w,
				http.StatusBadRequest,
				"Invalid JSON",
			)
			return
		}

		// --------------------------------
		// Clean input
		// --------------------------------

		request.FullName = strings.TrimSpace(request.FullName)

		request.Email = strings.ToLower(
			strings.TrimSpace(request.Email),
		)

		request.Phone = strings.TrimSpace(
			request.Phone,
		)

		request.Role = strings.ToLower(
			strings.TrimSpace(request.Role),
		)

		// --------------------------------
		// Check required fields
		// --------------------------------

		if request.FullName == "" ||
			request.Email == "" ||
			request.Phone == "" ||
			request.Password == "" ||
			request.ConfirmPassword == "" ||
			request.Role == "" {

			sendError(
				w,
				http.StatusBadRequest,
				"All fields are required",
			)
			return
		}

		// --------------------------------
		// Normalize international phone
		// --------------------------------
		//
		// The Flutter application already
		// converts the selected country and
		// local number into international format.
		//
		// Examples:
		//
		// +251912345678
		// +12025550123
		// +447911123456
		//
		// The backend keeps the international
		// format and removes accidental spaces
		// and hyphens.
		// --------------------------------

		normalizedPhone, err := normalizeInternationalPhone(
			request.Phone,
		)

		if err != nil {
			sendError(
				w,
				http.StatusBadRequest,
				err.Error(),
			)
			return
		}

		// From this point onward, always use
		// the normalized international number.

		request.Phone = normalizedPhone

		// --------------------------------
		// Check password confirmation
		// --------------------------------

		if request.Password != request.ConfirmPassword {

			sendError(
				w,
				http.StatusBadRequest,
				"Passwords do not match",
			)
			return
		}

		// --------------------------------
		// Check password length
		// --------------------------------

		if len(request.Password) < 8 {

			sendError(
				w,
				http.StatusBadRequest,
				"Password must be at least 8 characters",
			)
			return
		}

		// --------------------------------
		// Check allowed public roles
		// --------------------------------

		if request.Role != "tenant" &&
			request.Role != "maintenance_worker" {

			sendError(
				w,
				http.StatusBadRequest,
				"Invalid registration role. Only tenant and maintenance_worker can register.",
			)
			return
		}

		// --------------------------------
		// Check whether email already exists
		// --------------------------------

		var existingEmail string

		err = db.QueryRow(
			context.Background(),
			"SELECT email FROM users WHERE email = $1",
			request.Email,
		).Scan(&existingEmail)

		if err == nil {

			sendError(
				w,
				http.StatusConflict,
				"Email is already registered",
			)
			return
		}

		if err != pgx.ErrNoRows {

			sendError(
				w,
				http.StatusInternalServerError,
				"Failed to check email",
			)
			return
		}

		// --------------------------------
		// Check whether phone already exists
		// --------------------------------

		var existingPhone string

		err = db.QueryRow(
			context.Background(),
			"SELECT phone FROM users WHERE phone = $1",
			request.Phone,
		).Scan(&existingPhone)

		if err == nil {

			sendError(
				w,
				http.StatusConflict,
				"Phone number is already registered",
			)
			return
		}

		if err != pgx.ErrNoRows {

			sendError(
				w,
				http.StatusInternalServerError,
				"Failed to check phone number",
			)
			return
		}

		// --------------------------------
		// Hash password
		// --------------------------------

		hashedPassword, err := bcrypt.GenerateFromPassword(
			[]byte(request.Password),
			bcrypt.DefaultCost,
		)

		if err != nil {

			sendError(
				w,
				http.StatusInternalServerError,
				"Failed to secure password",
			)
			return
		}

		// --------------------------------
		// Insert user into PostgreSQL
		// --------------------------------

		var userID int

		err = db.QueryRow(
			context.Background(),
			`
			INSERT INTO users
			(
				full_name,
				email,
				phone,
				password_hash,
				role
			)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id
			`,
			request.FullName,
			request.Email,
			request.Phone,
			string(hashedPassword),
			request.Role,
		).Scan(&userID)

		if err != nil {

			sendError(
				w,
				http.StatusInternalServerError,
				"Could not create user",
			)
			return
		}

		// --------------------------------
		// Successful response
		// --------------------------------

		w.Header().Set(
			"Content-Type",
			"application/json",
		)

		w.WriteHeader(http.StatusCreated)

		response := RegisterResponse{
			Message: "User registered successfully",
			UserID:  userID,
			Role:    request.Role,
		}

		json.NewEncoder(w).Encode(response)
	}
}

// ==============================
// INTERNATIONAL PHONE NORMALIZATION
// ==============================
//
// Expected format:
//
// +251912345678
// +12025550123
// +447911123456
//
// The phone number must:
// - start with +
// - contain only digits after +
// - contain between 7 and 15 digits
//
// This follows the general international
// phone-number length convention used by
// E.164-style numbers.
//
// ==============================

func normalizeInternationalPhone(phone string) (string, error) {

	phone = strings.TrimSpace(phone)

	// Remove spaces and hyphens.
	//
	// Example:
	// +251 912 345 678
	//
	// becomes:
	// +251912345678

	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")

	// International numbers must begin
	// with a plus sign.

	if !strings.HasPrefix(phone, "+") {

		return "", errorMessage(
			"Invalid phone number. Use international format such as +251912345678",
		)
	}

	// Remove the + temporarily so we
	// can validate the digits.

	digits := phone[1:]

	// International phone number must
	// contain between 7 and 15 digits.

	if len(digits) < 7 || len(digits) > 15 {

		return "", errorMessage(
			"Invalid international phone number",
		)
	}

	// First digit after + cannot be zero.

	if digits[0] == '0' {

		return "", errorMessage(
			"Invalid international phone number",
		)
	}

	// Make sure every remaining character
	// is a digit.

	for _, character := range digits {

		if character < '0' || character > '9' {

			return "", errorMessage(
				"Invalid international phone number",
			)
		}
	}

	// Return the normalized international
	// phone number.

	return "+" + digits, nil
}

// ==============================
// SIMPLE ERROR HELPER
// ==============================

type errorMessage string

func (e errorMessage) Error() string {
	return string(e)
}

// ==============================
// ERROR RESPONSE
// ==============================

func sendError(
	w http.ResponseWriter,
	statusCode int,
	message string,
) {

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(statusCode)

	response := map[string]string{
		"message": message,
	}

	json.NewEncoder(w).Encode(response)
}