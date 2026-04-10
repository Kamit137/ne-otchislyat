package pay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type Amount struct {
	Value    string `json:"value"`
	Currency string `json:"currency"`
}

type Confirmation struct {
	Type      string `json:"type"`
	ReturnURL string `json:"return_url"`
}

type PaymentRequest struct {
	Amount       Amount            `json:"amount"`
	Confirmation Confirmation      `json:"confirmation"`
	Capture      bool              `json:"capture"`
	Description  string            `json:"description"`
	Metadata     map[string]string `json:"metadata"`
}

// CreatePayment создает платеж в ЮKassa и возвращает URL для оплаты
func CreatePayment(amount float64, returnURL string, metadata map[string]string) (string, error) {
	shopID := "515309"
	secretKey := "test_*g1lrspzB6cRGyDpQvvFBe5p2K5ZwPY-jrW9ZMO1ub3Xw"
	apiURL := "https://api.yookassa.ru/v3"

	payment := PaymentRequest{
		Amount: Amount{
			Value:    fmt.Sprintf("%.2f", amount),
			Currency: "RUB",
		},
		Confirmation: Confirmation{
			Type:      "redirect",
			ReturnURL: returnURL,
		},
		Capture:     true,
		Description: "Пополнение баланса",
		Metadata:    metadata,
	}

	jsonData, err := json.Marshal(payment)
	if err != nil {
		return "", fmt.Errorf("ошибка сериализации: %v", err)
	}

	req, err := http.NewRequest("POST", apiURL+"/payments", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("ошибка создания запроса: %v", err)
	}

	req.SetBasicAuth(shopID, secretKey) // ← это всё, что нужно
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotence-Key", uuid.New().String())

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	// ... остальной код без изменений

	if err != nil {
		return "", fmt.Errorf("ошибка запроса к ЮKassa: %v", err)
	}
	defer resp.Body.Close()

	// Читаем тело ответа
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ошибка чтения ответа: %v", err)
	}

	log.Printf("📥 Ответ ЮKassa (статус %d): %s", resp.StatusCode, string(bodyBytes))

	// Парсим JSON
	var result map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return "", fmt.Errorf("ошибка парсинга ответа: %v", err)
	}

	// Проверяем статус ответа
	if resp.StatusCode != http.StatusOK {
		errorMsg := "неизвестная ошибка"
		if description, ok := result["description"].(string); ok {
			errorMsg = description
		}
		if errMsg, ok := result["error"].(string); ok {
			errorMsg = errMsg
		}
		return "", fmt.Errorf("❌ ЮKassa ошибка %d: %s", resp.StatusCode, errorMsg)
	}

	// Проверяем наличие поля confirmation
	confirmation, ok := result["confirmation"].(map[string]interface{})
	if !ok {
		status, _ := result["status"].(string)
		id, _ := result["id"].(string)
		return "", fmt.Errorf("❌ неверный ответ: нет confirmation. Статус: %s, ID: %s", status, id)
	}

	confirmURL, ok := confirmation["confirmation_url"].(string)
	if !ok {
		return "", fmt.Errorf("❌ неверный ответ: нет confirmation_url")
	}

	log.Printf("✅ Платеж создан, URL: %s", confirmURL)
	return confirmURL, nil
}

// HandleDeposit - обработчик для пополнения баланса
func HandleDeposit(w http.ResponseWriter, r *http.Request) {
	fmt.Println("💰 Обработка депозита")

	// Получаем email пользователя из контекста
	email, ok := r.Context().Value("email").(string)
	if !ok || email == "" {
		log.Printf("❌ Ошибка: email не найден в контексте")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Парсим запрос
	var req struct {
		Amount float64 `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ Ошибка парсинга JSON: %v", err)
		http.Error(w, "Неверный формат запроса", http.StatusBadRequest)
		return
	}

	log.Printf("📝 Запрос на депозит: email=%s, amount=%.2f", email, req.Amount)

	if req.Amount <= 0 {
		http.Error(w, "Сумма должна быть больше 0", http.StatusBadRequest)
		return
	}

	// Создаем платеж в ЮKassa
	metadata := map[string]string{
		"email":  email,
		"amount": fmt.Sprintf("%.2f", req.Amount),
	}

	paymentURL, err := CreatePayment(
		req.Amount,
		"http://ne-otchislyat.ru/profile",
		metadata,
	)

	if err != nil {
		log.Printf("❌ Ошибка создания платежа: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "payment_creation_failed",
			"message": err.Error(),
		})
		return
	}

	// Возвращаем URL для редиректа
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"payment_url": paymentURL,
	})
}
