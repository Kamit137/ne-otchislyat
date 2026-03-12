package pay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"ne-otchislyat/internal/sql"
	"net/http"

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

func HandleDeposit(w http.ResponseWriter, r *http.Request) {
	email := r.Context().Value("email").(string)
	var req struct {
		Amount float64 `json:"amount"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	payment := PaymentRequest{
		Amount: Amount{
			Value:    fmt.Sprintf("%.2f", req.Amount),
			Currency: "RUB",
		},
		Confirmation: Confirmation{
			Type:      "redirect",
			ReturnURL: "http://localhost:8080/lenta",
		},
		Capture:     true,
		Description: "Пополнение баланса",
		Metadata: map[string]string{
			"email": email,
		},
	}

	jsonData, _ := json.Marshal(payment)

	// Отправляем в ЮKassa
	client := &http.Client{}
	request, _ := http.NewRequest("POST", apiURL+"/payments", bytes.NewBuffer(jsonData))
	request.SetBasicAuth(shopID, secretKey)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotence-Key", uuid.New().String())

	resp, _ := client.Do(request)
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	// Отдаем ссылку на оплату
	json.NewEncoder(w).Encode(map[string]string{
		"url": result["confirmation"].(map[string]interface{})["confirmation_url"].(string),
	})
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

	json.NewDecoder(r.Body).Decode(&notification)

	// платеж прошел успешно
	if notification.Event == "payment.succeeded" {
		email := notification.Object.Metadata["email"]

		var rubles int64
		fmt.Sscanf(notification.Object.Amount.Value, "%f", &rubles)
		rubles *= 100

		err := sql.DepositSql(rubles, email)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

func GetBalance(w http.ResponseWriter, r *http.Request) {
	email := r.Context().Value("email").(string)

	balance, frozen, err := sql.GetUserBalance(email)
	if err != nil {
		http.Error(w, "Ошибка получения баланса", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"balance": float64(balance) / 100,
		"frozen":  float64(frozen) / 100,
	})
}

func CreateOrder(w http.ResponseWriter, r *http.Request) {
	email := r.Context().Value("email").(string)

	var req struct {
		VakansID int `json:"vakans_id"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	orderID, err := sql.CreateOrder(req.VakansID, email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"order_id": orderID,
		"message":  "Заказ создан, деньги заморожены",
	})
}

func CompleteOrder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OrderID int64 `json:"order_id"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	err := sql.CompleteOrder(req.OrderID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"message": "Заказ выполнен, деньги переведены",
	})
}

// sdf
func CancelOrder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OrderID int64 `json:"order_id"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	err := sql.CancelOrder(req.OrderID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"message": "Заказ отменен, деньги возвращены",
	})
}
