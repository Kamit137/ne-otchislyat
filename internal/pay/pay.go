package pay

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"ne-otchislyat/internal/sql"
	"net/http"
	"strconv"
	"time"
)

const baseURL = "https://api.intellectmoney.ru/merchant"

type client struct {
	eshopID    string
	secretKey  string
	httpClient *http.Client
}

var c *client

func InitPay(eshopID, secretKey string) {
	c = &client{
		eshopID:    eshopID,
		secretKey:  secretKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func md5hash(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

// ─── Создание инвойса ─────────────────────────────────────────────────────────

type invoiceResponse struct {
	OperationState struct {
		Code int    `json:"Code"`
		Desc string `json:"Desc"`
	} `json:"OperationState"`
	EshopID int `json:"EshopId"`
	Result  struct {
		State struct {
			Code             int    `json:"Code"`
			Desc             string `json:"Desc"`
			ErrorSourceParam string `json:"ErrorSourceParam,omitempty"`
		} `json:"State"`
		InvoiceID   interface{} `json:"InvoiceId"`
		PaymentWays []struct {
			ID         int    `json:"Id"`
			Preference string `json:"Preference"`
		} `json:"PaymentWays"`
	} `json:"Result"`
}

func (r *invoiceResponse) getInvoiceID() string {
	if r.Result.InvoiceID == nil {
		return ""
	}
	switch v := r.Result.InvoiceID.(type) {
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', 0, 64)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func createInvoice(orderID string, amount float64, email, successURL, failURL, resultURL string) (*invoiceResponse, error) {
	amountStr := fmt.Sprintf("%.2f", amount)
	serviceName := "Пополнение баланса"
	currency := "TST" // Тестовая валюта

	// Формируем строку для хеша (15 элементов, разделенных ::)
	hashStr := fmt.Sprintf("%s::%s::%s::%s::%s::%s::%s::%s::%s::%s::%s::%s::%s::%s::%s",
		c.eshopID,   // eshopId
		orderID,     // orderId
		serviceName, // serviceName
		amountStr,   // recipientAmount
		currency,    // recipientCurrency
		"",          // userName
		email,       // email
		successURL,  // successUrl
		failURL,     // failUrl
		"",          // backUrl
		resultURL,   // resultUrl
		"",          // expireDate
		"",          // holdMode
		"",          // preference
		c.secretKey, // secretKey
	)

	paramHash := md5hash(hashStr)

	log.Printf("🔐 Хеш-строка: %s", hashStr)
	log.Printf("🔐 MD5 хеш: %s", paramHash)

	body := map[string]interface{}{
		"eshopId":           c.eshopID,
		"orderId":           orderID,
		"recipientAmount":   amountStr,
		"recipientCurrency": currency,
		"email":             email,
		"serviceName":       serviceName,
		"successUrl":        successURL,
		"failUrl":           failURL,
		"resultUrl":         resultURL,
		"hash":              paramHash,
	}

	jsonBody, _ := json.Marshal(body)
	log.Printf("📤 Запрос к IntellectMoney: %s", string(jsonBody))

	req, err := http.NewRequest("POST", baseURL+"/createInvoice", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("запрос к IntellectMoney: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("📥 IntellectMoney createInvoice (статус %d): %s", resp.StatusCode, string(respBody))

	var result invoiceResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("парсинг ответа: %w, тело: %s", err, string(respBody))
	}

	if result.OperationState.Code != 0 {
		return nil, fmt.Errorf("ошибка API OperationState: %s", result.OperationState.Desc)
	}

	if result.Result.State.Code != 0 {
		return nil, fmt.Errorf("ошибка API Result.State: %s (параметр: %s)",
			result.Result.State.Desc, result.Result.State.ErrorSourceParam)
	}

	return &result, nil
}

func paymentLink(invoiceID string) string {
	return fmt.Sprintf("https://merchant.intellectmoney.ru/pay/%s", invoiceID)
}

// ─── HTTP-обработчики ─────────────────────────────────────────────────────────

// HandleDeposit — пополнение баланса (тестовый режим)
func HandleDeposit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	email, ok := r.Context().Value("email").(string)
	if !ok || email == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Amount float64 `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Неверный формат запроса", http.StatusBadRequest)
		return
	}
	if req.Amount < 1 {
		http.Error(w, "Минимальная сумма: 1 TST", http.StatusBadRequest)
		return
	}

	orderID := fmt.Sprintf("dep_%d", time.Now().UnixNano())

	// Сохраняем pending-транзакцию
	tx, err := sql.DB.Begin()
	if err != nil {
		log.Printf("❌ Ошибка начала транзакции: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var userID int
	if err := tx.QueryRow("SELECT id FROM users WHERE email = $1", email).Scan(&userID); err != nil {
		log.Printf("❌ Пользователь не найден: %v", err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	_, err = tx.Exec(`
		INSERT INTO transactions (user_id, type, amount, status, payment_id)
		VALUES ($1, 'deposit', $2, 'pending', $3)`,
		userID, int64(req.Amount*100), orderID)
	if err != nil {
		log.Printf("❌ Ошибка создания транзакции: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("❌ Ошибка коммита: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Создаём инвойс
	inv, err := createInvoice(
		orderID, req.Amount, email,
		"http://localhost:8080/profile", // successURL
		"http://localhost:8080/profile", // failURL
		"https://ne-otchislyat.ru/",     // resultURL (должен быть публичным)
	)
	if err != nil {
		log.Printf("❌ createInvoice: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	invoiceID := inv.getInvoiceID()
	log.Printf("✅ Тестовый инвойс создан: %s", invoiceID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"payment_url": paymentLink(invoiceID),
		"invoice_id":  invoiceID,
		"test_mode":   true,
	})
}

// HandlePaymentNotification — callback от IntellectMoney
func HandlePaymentNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("❌ Ошибка чтения тела: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	log.Printf("📥 Payment notification: %s", string(body))

	// Проверяем подпись
	receivedHash := r.Header.Get("Hash")
	expected := md5hash(string(body) + c.secretKey)
	if expected != receivedHash {
		log.Printf("❌ Неверная подпись callback. Ожидалась: %s, получена: %s", expected, receivedHash)
		http.Error(w, "Invalid signature", http.StatusForbidden)
		return
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		log.Printf("❌ Ошибка парсинга JSON: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	orderID, _ := data["orderId"].(string)
	state, _ := data["state"].(string)

	log.Printf("📝 Callback: orderId=%s, state=%s", orderID, state)

	if orderID == "" || state != "Paid" {
		log.Printf("⚠️ Пропускаем: state=%s", state)
		w.Write([]byte("OK"))
		return
	}

	tx, err := sql.DB.Begin()
	if err != nil {
		log.Printf("❌ Ошибка начала транзакции: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var userID int
	var amount int64
	var status string
	err = tx.QueryRow(`
		SELECT user_id, amount, status FROM transactions
		WHERE payment_id = $1 AND type = 'deposit'`, orderID).Scan(&userID, &amount, &status)
	if err != nil {
		log.Printf("❌ Транзакция не найдена: %v", err)
		http.Error(w, "Transaction not found", http.StatusNotFound)
		return
	}

	if status == "success" {
		log.Printf("⚠️ Транзакция уже обработана: %s", orderID)
		tx.Commit()
		w.Write([]byte("OK"))
		return
	}

	_, err = tx.Exec(`UPDATE transactions SET status = 'success' WHERE payment_id = $1`, orderID)
	if err != nil {
		log.Printf("❌ Ошибка обновления статуса: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec(`UPDATE users SET balance = balance + $1 WHERE id = $2`, amount, userID)
	if err != nil {
		log.Printf("❌ Ошибка обновления баланса: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("❌ Ошибка коммита: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Тестовый депозит зачислен: user_id=%d amount=%d копеек", userID, amount)
	w.Write([]byte("OK"))
}

// GetBalance — баланс пользователя
func GetBalance(w http.ResponseWriter, r *http.Request) {
	email, ok := r.Context().Value("email").(string)
	if !ok || email == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	balance, frozen, err := sql.GetUserBalance(email)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"balance": float64(balance) / 100,
		"frozen":  float64(frozen) / 100,
	})
}

// CreateOrder — создание заказа
func CreateOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	email, ok := r.Context().Value("email").(string)
	if !ok || email == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		VakansID int `json:"vakans_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	orderID, err := sql.CreateOrder(req.VakansID, email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "order_id": orderID})
}

// CompleteOrder — завершение заказа
func CompleteOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		OrderID int64 `json:"order_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if err := sql.CompleteOrder(req.OrderID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

// CancelOrder — отмена заказа
func CancelOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		OrderID int64 `json:"order_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if err := sql.CancelOrder(req.OrderID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}
