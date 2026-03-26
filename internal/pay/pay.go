package pay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"ne-otchislyat/internal/sql"
	"net/http"
	"strconv"

	"github.com/google/uuid"
)

const (
	shopID    = "514065"
	secretKey = "test_*g0KyEXeFC18jpmrJ5Gmy4UH2bpFgtMdmkW5JxvfCgCmo"
	apiURL    = "https://api.yookassa.ru/v3"
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

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func HandleDeposit(w http.ResponseWriter, r *http.Request) {
	email, ok := r.Context().Value("email").(string)
	if !ok {
		jsonError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Amount float64 `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Неверный формат запроса", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if req.Amount <= 0 {
		jsonError(w, "Сумма должна быть больше нуля", http.StatusBadRequest)
		return
	}

	payment := PaymentRequest{
		Amount: Amount{
			Value:    fmt.Sprintf("%.2f", req.Amount),
			Currency: "RUB",
		},
		Confirmation: Confirmation{
			Type:      "redirect",
			ReturnURL: "http://localhost:8080/",
		},
		Capture:     true,
		Description: "Пополнение баланса",
		Metadata: map[string]string{
			"email": email,
		},
	}

	jsonData, err := json.Marshal(payment)
	if err != nil {
		jsonError(w, "Ошибка формирования запроса", http.StatusInternalServerError)
		return
	}

	client := &http.Client{}
	request, err := http.NewRequest("POST", apiURL+"/payments", bytes.NewBuffer(jsonData))
	if err != nil {
		jsonError(w, "Ошибка создания запроса к ЮKassa", http.StatusInternalServerError)
		return
	}
	request.SetBasicAuth(shopID, secretKey)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotence-Key", uuid.New().String())

	resp, err := client.Do(request)
	if err != nil {
		jsonError(w, "Ошибка соединения с ЮKassa", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		jsonError(w, "Ошибка ответа от ЮKassa", http.StatusBadGateway)
		return
	}

	confirmation, ok := result["confirmation"].(map[string]interface{})
	if !ok {
		jsonError(w, "Неверный ответ от ЮKassa: нет confirmation", http.StatusBadGateway)
		return
	}
	confirmURL, ok := confirmation["confirmation_url"].(string)
	if !ok {
		jsonError(w, "Неверный ответ от ЮKassa: нет confirmation_url", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"url": confirmURL})
}

// ЮKassa присылает подтверждение оплаты
func HandleWebhook(w http.ResponseWriter, r *http.Request) {
	var notification struct {
		Event  string `json:"event"`
		Object struct {
			ID       string            `json:"id"`
			Status   string            `json:"status"`
			Metadata map[string]string `json:"metadata"`
			Amount   struct {
				Value string `json:"value"`
			} `json:"amount"`
		} `json:"object"`
	}

	if err := json.NewDecoder(r.Body).Decode(&notification); err != nil {
		jsonError(w, "Неверный формат уведомления", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if notification.Event == "payment.succeeded" {
		email := notification.Object.Metadata["email"]
		if email == "" {
			jsonError(w, "Email не найден в метаданных", http.StatusBadRequest)
			return
		}

		rubles, err := strconv.ParseFloat(notification.Object.Amount.Value, 64)
		if err != nil {
			jsonError(w, "Ошибка парсинга суммы: "+err.Error(), http.StatusBadRequest)
			return
		}
		kopecks := int64(rubles * 100)

		if err := sql.DepositSql(kopecks, email); err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

func GetBalance(w http.ResponseWriter, r *http.Request) {
	email, ok := r.Context().Value("email").(string)
	if !ok {
		jsonError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	balance, frozen, err := sql.GetUserBalance(email)
	if err != nil {
		jsonError(w, "Ошибка получения баланса", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"balance": float64(balance) / 100,
		"frozen":  float64(frozen) / 100,
	})
}

func CreateOrder(w http.ResponseWriter, r *http.Request) {
	email, ok := r.Context().Value("email").(string)
	if !ok {
		jsonError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		VakansID int `json:"vakans_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Неверный формат запроса", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	orderID, err := sql.CreateOrder(req.VakansID, email)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"order_id": orderID,
		"message":  "Заказ создан, деньги заморожены",
	})
}

func CompleteOrder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OrderID int64 `json:"order_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Неверный формат запроса", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if err := sql.CompleteOrder(req.OrderID); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Заказ выполнен, деньги переведены",
	})
}

func CancelOrder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OrderID int64 `json:"order_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Неверный формат запроса", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if err := sql.CancelOrder(req.OrderID); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Заказ отменен, деньги возвращены",
	})
}
