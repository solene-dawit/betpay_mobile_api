package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
	
)

// ==========================================
// StarPay Service
// ==========================================

type StarPayService struct {
	BaseURL     string
	APISecret   string
	MerchantID  string
	CallbackKey string
}

// ==========================================
// StarPay Item
// ==========================================

type StarPayItem struct {
	ProductID string  `json:"productId"`
	Quantity  int     `json:"quantity"`
	ItemName  string  `json:"item_name"`
	UnitPrice float64 `json:"unit_price"`
}

// ==========================================
// Create Transaction Request
// ==========================================

type CreateTransactionRequest struct {
	Amount              float64       `json:"amount"`
	Description         string        `json:"description"`
	Currency            string        `json:"currency"`
	CustomerName        string        `json:"customerName"`
	CustomerPhoneNumber string        `json:"customerPhoneNumber"`
	Items               []StarPayItem `json:"items"`

	CallbackURL   string                 `json:"callbackURL,omitempty"`
	RedirectURL   string                 `json:"redirectUrl,omitempty"`
	CustomerEmail string                 `json:"customerEmail,omitempty"`
	ExpiredAt     string                 `json:"expiredAt,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// ==========================================
// StarPay Transaction Response
// ==========================================

type CreateTransactionResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`

	Data struct {
		OrderID     string `json:"order_id"`
		Status      string `json:"status"`
		Amount      string `json:"amount"`
		Currency    string `json:"currency"`
		PaymentURL  string `json:"payment_url"`
		RedirectURL string `json:"redirectUrl"`
		ExpiresAt   string `json:"expires_at"`
	} `json:"data"`
}

// ==========================================
// Constructor
// ==========================================

func NewStarPayService() *StarPayService {
	return &StarPayService{
		BaseURL: strings.TrimRight(
			os.Getenv("STARPAY_API_URL"),
			"/",
		),

		APISecret: os.Getenv(
			"STARPAY_API_SECRET",
		),

		MerchantID: os.Getenv(
			"STARPAY_MERCHANT_ID",
		),

		CallbackKey: os.Getenv(
			"STARPAY_CALLBACK_KEY",
		),
	}
}

// ==========================================
// NORMALIZE INTERNATIONAL PHONE
// ==========================================
//
// Database examples:
//
// +251904380908
// +12025550123
// +447911123456
// +254712345678
//
// We DO NOT force +251.
// ==========================================

func normalizeStarPayPhone(phone string) string {

	phone = strings.TrimSpace(phone)

	// Remove spaces
	phone = strings.ReplaceAll(phone, " ", "")

	// Remove hyphens
	phone = strings.ReplaceAll(phone, "-", "")

	// Remove parentheses
	phone = strings.ReplaceAll(phone, "(", "")
	phone = strings.ReplaceAll(phone, ")", "")

	// Already international
	if strings.HasPrefix(phone, "+") {
		return phone
	}

	// International without +
	if len(phone) > 0 {
		return "+" + phone
	}

	return ""
}

// ==========================================
// CREATE TRANSACTION
// ==========================================

func (s *StarPayService) CreateTransaction(
	requestData CreateTransactionRequest,
) (*CreateTransactionResponse, error) {

	// ==========================================
	// Validate configuration
	// ==========================================

	if s.BaseURL == "" {
		return nil, fmt.Errorf(
			"STARPAY_API_URL is not configured",
		)
	}

	if s.APISecret == "" {
		return nil, fmt.Errorf(
			"STARPAY_API_SECRET is not configured",
		)
	}

	// ==========================================
	// Normalize phone
	// ==========================================

	requestData.CustomerPhoneNumber =
		normalizeStarPayPhone(
			requestData.CustomerPhoneNumber,
		)

	if requestData.CustomerPhoneNumber == "" {
		return nil, fmt.Errorf(
			"customer phone number is required",
		)
	}

	// ==========================================
	// Convert request to JSON
	// ==========================================

	jsonData, err := json.Marshal(requestData)

	if err != nil {
		return nil, fmt.Errorf(
			"failed to encode StarPay request: %w",
			err,
		)
	}

	// ==========================================
	// DEBUG REQUEST
	// ==========================================

	fmt.Println()
	fmt.Println("========== STARPAY JSON ==========")
	fmt.Println(string(jsonData))
	fmt.Println("==================================")

	// ==========================================
	// STARPay endpoint
	// ==========================================

	url := s.BaseURL + "/trdp/order"

	fmt.Println()
	fmt.Println("========== STARPAY REQUEST ==========")
	fmt.Println("URL:", url)
	fmt.Println("Amount:", requestData.Amount)
	fmt.Println("Currency:", requestData.Currency)
	fmt.Println("Customer:", requestData.CustomerName)
	fmt.Println("Phone:", requestData.CustomerPhoneNumber)
	fmt.Println("Description:", requestData.Description)
	fmt.Println("=====================================")

	// ==========================================
	// HTTP REQUEST
	// ==========================================

	req, err := http.NewRequest(
		http.MethodPost,
		url,
		bytes.NewBuffer(jsonData),
	)

	if err != nil {
		return nil, fmt.Errorf(
			"failed to create StarPay request: %w",
			err,
		)
	}

	// ==========================================
	// HEADERS
	// ==========================================

	req.Header.Set(
		"Content-Type",
		"application/json",
	)

	req.Header.Set(
		"x-api-secret",
		s.APISecret,
	)

	// ==========================================
	// HTTP CLIENT
	// ==========================================

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// ==========================================
	// SEND REQUEST
	// ==========================================

	response, err := client.Do(req)

	if err != nil {
		return nil, fmt.Errorf(
			"failed to connect to StarPay: %w",
			err,
		)
	}

	defer response.Body.Close()

	// ==========================================
	// READ RESPONSE
	// ==========================================

	responseBody, err := io.ReadAll(
		response.Body,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"failed to read StarPay response: %w",
			err,
		)
	}

	// ==========================================
	// PRINT RAW RESPONSE
	// ==========================================

	fmt.Println()
	fmt.Println("========== STARPAY RESPONSE ==========")
	fmt.Println("HTTP Status:", response.StatusCode)
	fmt.Println("Response:", string(responseBody))
	fmt.Println("======================================")
	fmt.Println()

	// ==========================================
	// CHECK HTTP STATUS
	// ==========================================

	if response.StatusCode < 200 ||
		response.StatusCode >= 300 {

		return nil, fmt.Errorf(
			"StarPay returned status %d: %s",
			response.StatusCode,
			string(responseBody),
		)
	}

	// ==========================================
	// DECODE RESPONSE
	// ==========================================

	var result CreateTransactionResponse

	err = json.Unmarshal(
		responseBody,
		&result,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"failed to decode StarPay response: %w",
			err,
		)
	}

	// ==========================================
	// DEBUG DECODED DATA
	// ==========================================

	fmt.Println("========== DECODED STARPAY DATA ==========")
	fmt.Println("Status:", result.Status)
	fmt.Println("Message:", result.Message)
	fmt.Println("Order ID:", result.Data.OrderID)
	fmt.Println("Payment Status:", result.Data.Status)
	fmt.Println("Amount:", result.Data.Amount)
	fmt.Println("Currency:", result.Data.Currency)
	fmt.Println("Payment URL:", result.Data.PaymentURL)
	fmt.Println("Redirect URL:", result.Data.RedirectURL)
	fmt.Println("Expires At:", result.Data.ExpiresAt)
	fmt.Println("==========================================")

	// ==========================================
	// CHECK STARPAY STATUS
	// ==========================================

	if result.Status != "success" {
		return nil, fmt.Errorf(
			"StarPay transaction failed: %s",
			result.Message,
		)
	}

	// ==========================================
	// CHECK ORDER ID
	// ==========================================

	if result.Data.OrderID == "" {
		return nil, fmt.Errorf(
			"StarPay response did not contain order_id",
		)
	}

	// ==========================================
	// CHECK PAYMENT URL
	// ==========================================

	if result.Data.PaymentURL == "" {
		return nil, fmt.Errorf(
			"StarPay response did not contain payment_url. Raw response: %s",
			string(responseBody),
		)
	}

	// ==========================================
	// SUCCESS
	// ==========================================

	fmt.Println()
	fmt.Println("========== STARPAY SUCCESS ==========")
	fmt.Println("Order ID:", result.Data.OrderID)
	fmt.Println("Payment URL:", result.Data.PaymentURL)
	fmt.Println("=====================================")
	fmt.Println()

	return &result, nil
}