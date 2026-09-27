package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PaymentHandler struct {
	DB *pgxpool.Pool
}

type SavePaymentRequest struct {
	UserID              int     `json:"user_id"`
	PaymentType         string  `json:"payment_type"`
	Amount              float64 `json:"amount"`
	Currency            string  `json:"currency"`
	Status              string  `json:"status"`
	StarPayOrderID      string  `json:"starpay_order_id"`
	TransactionReference string `json:"transaction_reference"`
	ReceiptNumber       string  `json:"receipt_number"`
}

func NewPaymentHandler(db *pgxpool.Pool) *PaymentHandler {
	return &PaymentHandler{
		DB: db,
	}
}

func (h *PaymentHandler) SavePayment(
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

	var request SavePaymentRequest

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(
			w,
			"Invalid request body",
			http.StatusBadRequest,
		)
		return
	}

	query := `
		INSERT INTO payments (
			user_id,
			payment_type,
			amount,
			currency,
			status,
			starpay_order_id,
			transaction_reference,
			receipt_number,
			paid_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			CASE
				WHEN $5 = 'PAID'
				THEN CURRENT_TIMESTAMP
				ELSE NULL
			END
		)
		RETURNING id, created_at
	`

	var paymentID int
	var createdAt interface{}

	err = h.DB.QueryRow(
		context.Background(),
		query,
		request.UserID,
		request.PaymentType,
		request.Amount,
		request.Currency,
		request.Status,
		request.StarPayOrderID,
		request.TransactionReference,
		request.ReceiptNumber,
	).Scan(
		&paymentID,
		&createdAt,
	)

	if err != nil {
		http.Error(
			w,
			"Failed to save payment",
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Payment saved successfully",
		"payment": map[string]interface{}{
			"id":         paymentID,
			"created_at": createdAt,
		},
	})
}