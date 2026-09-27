package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"betpay_mobile_api/services"
)

// ------------------------------------
// StarPay Handler
// ------------------------------------

type StarPayHandler struct {
	StarPay *services.StarPayService
}

// ------------------------------------
// Request From Flutter
// ------------------------------------

type StarPayPaymentRequest struct {
	Amount              float64 `json:"amount"`
	Description         string  `json:"description"`
	Currency            string  `json:"currency"`
	CustomerName        string  `json:"customerName"`
	CustomerPhoneNumber string  `json:"customerPhoneNumber"`

	Items []StarPayItemRequest `json:"items"`

	CallbackURL   string                 `json:"callbackURL,omitempty"`
	RedirectURL   string                 `json:"redirectUrl,omitempty"`
	CustomerEmail string                 `json:"customerEmail,omitempty"`
	ExpiredAt     string                 `json:"expiredAt,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// ------------------------------------
// StarPay Item
// ------------------------------------

type StarPayItemRequest struct {
	ProductID string  `json:"productId"`
	Quantity  int     `json:"quantity"`
	ItemName  string  `json:"item_name"`
	UnitPrice float64 `json:"unit_price"`
}

// ------------------------------------
// Constructor
// ------------------------------------

func NewStarPayHandler(
	starPay *services.StarPayService,
) *StarPayHandler {

	return &StarPayHandler{
		StarPay: starPay,
	}
}

// ------------------------------------
// Create Payment
// ------------------------------------

func (h *StarPayHandler) CreatePayment(
	w http.ResponseWriter,
	r *http.Request,
) {

	// --------------------------------
	// Only POST is allowed
	// --------------------------------

	if r.Method != http.MethodPost {

		http.Error(
			w,
			"Method not allowed",
			http.StatusMethodNotAllowed,
		)

		return
	}

	// --------------------------------
	// Decode request
	// --------------------------------

	var request StarPayPaymentRequest

	err := json.NewDecoder(
		r.Body,
	).Decode(&request)

	if err != nil {

		http.Error(
			w,
			"Invalid request body",
			http.StatusBadRequest,
		)

		return
	}

	// --------------------------------
	// Validate amount
	// --------------------------------

	if request.Amount <= 0 {

		http.Error(
			w,
			"Amount must be greater than zero",
			http.StatusBadRequest,
		)

		return
	}

	// --------------------------------
	// Validate description
	// --------------------------------

	if strings.TrimSpace(
		request.Description,
	) == "" {

		http.Error(
			w,
			"Description is required",
			http.StatusBadRequest,
		)

		return
	}

	// --------------------------------
	// Validate currency
	// --------------------------------

	if request.Currency == "" {
		request.Currency = "ETB"
	}

	if strings.ToUpper(
		request.Currency,
	) != "ETB" {

		http.Error(
			w,
			"Only ETB currency is supported",
			http.StatusBadRequest,
		)

		return
	}

	request.Currency = "ETB"

	// --------------------------------
	// Validate customer name
	// --------------------------------

	if strings.TrimSpace(
		request.CustomerName,
	) == "" {

		http.Error(
			w,
			"Customer name is required",
			http.StatusBadRequest,
		)

		return
	}

	// --------------------------------
	// Validate customer phone
	// --------------------------------

	if strings.TrimSpace(
		request.CustomerPhoneNumber,
	) == "" {

		http.Error(
			w,
			"Customer phone number is required",
			http.StatusBadRequest,
		)

		return
	}

	// StarPay expects international format
	// such as +251987654567.

	if !strings.HasPrefix(
		request.CustomerPhoneNumber,
		"+251",
	) {

		http.Error(
			w,
			"Customer phone number must use international format, for example +251987654567",
			http.StatusBadRequest,
		)

		return
	}

	// --------------------------------
	// Validate items
	// --------------------------------

	if len(request.Items) == 0 {

		http.Error(
			w,
			"At least one item is required",
			http.StatusBadRequest,
		)

		return
	}

	// --------------------------------
	// Validate each item
	// --------------------------------

	for i, item := range request.Items {

		if strings.TrimSpace(
			item.ProductID,
		) == "" {

			http.Error(
				w,
				fmt.Sprintf(
					"items[%d].productId is required",
					i,
				),
				http.StatusBadRequest,
			)

			return
		}

		if item.Quantity <= 0 {

			http.Error(
				w,
				fmt.Sprintf(
					"items[%d].quantity must be greater than zero",
					i,
				),
				http.StatusBadRequest,
			)

			return
		}

		if strings.TrimSpace(
			item.ItemName,
		) == "" {

			http.Error(
				w,
				fmt.Sprintf(
					"items[%d].item_name is required",
					i,
				),
				http.StatusBadRequest,
			)

			return
		}

		if item.UnitPrice <= 0 {

			http.Error(
				w,
				fmt.Sprintf(
					"items[%d].unit_price must be greater than zero",
					i,
				),
				http.StatusBadRequest,
			)

			return
		}
	}

	// --------------------------------
	// Validate expiredAt
	// --------------------------------

	if request.ExpiredAt != "" {

		_, err := time.Parse(
			time.RFC3339,
			request.ExpiredAt,
		)

		if err != nil {

			http.Error(
				w,
				"expiredAt must use ISO 8601 format",
				http.StatusBadRequest,
			)

			return
		}
	}

	// --------------------------------
	// Convert items
	// --------------------------------

	items := make(
		[]services.StarPayItem,
		0,
		len(request.Items),
	)

	for _, item := range request.Items {

		items = append(
			items,
			services.StarPayItem{
				ProductID: item.ProductID,
				Quantity:  item.Quantity,
				ItemName:  item.ItemName,
				UnitPrice: item.UnitPrice,
			},
		)
	}

	// --------------------------------
	// Create StarPay transaction
	// --------------------------------

	transaction :=
		services.CreateTransactionRequest{

			Amount:
				request.Amount,

			Description:
				request.Description,

			Currency:
				request.Currency,

			CustomerName:
				request.CustomerName,

			CustomerPhoneNumber:
				request.CustomerPhoneNumber,

			Items:
				items,

			CallbackURL:
				request.CallbackURL,

			RedirectURL:
				request.RedirectURL,

			CustomerEmail:
				request.CustomerEmail,

			ExpiredAt:
				request.ExpiredAt,

			Metadata:
				request.Metadata,
		}

	// --------------------------------
	// Send transaction to StarPay
	// --------------------------------

	result, err :=
		h.StarPay.CreateTransaction(
			transaction,
		)

	if err != nil {

		log.Println(
			"StarPay payment error:",
			err,
		)

		http.Error(
			w,
			err.Error(),
			http.StatusBadGateway,
		)

		return
	}

	// --------------------------------
	// Return StarPay response to Flutter
	// --------------------------------

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	response := map[string]interface{}{

		"status":
			result.Status,

		"message":
			result.Message,

		"order_id":
			result.Data.OrderID,

		"payment_status":
			result.Data.Status,

		"amount":
			result.Data.Amount,

		"currency":
			result.Data.Currency,

		"payment_url":
			result.Data.PaymentURL,

		"redirectUrl":
			result.Data.RedirectURL,

		"expires_at":
			result.Data.ExpiresAt,
	}

	err = json.NewEncoder(
		w,
	).Encode(response)

	if err != nil {

		log.Println(
			"Failed to send StarPay response:",
			err,
		)
	}
}