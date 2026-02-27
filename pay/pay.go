package pay

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
)

const (
	shopID    = "514065"
	secretKey = "test_*g0KyEXeFC18jpmrJ5Gmy4UH2bpFgtMdmkW5JxvfCgCmo"
	returnURL = "http://localhost:8080/lenta"
)

type PaymentRequest struct {
	Amount struct {
		Value    string `json:"value"`
		Currency string `json:"currency"`
	} `json:"amount"`
	Confirmation struct {
		Type      string `json:"type"`
		ReturnURL string `json:"return_url"`
	} `json:"confirmation"`
	Capture     bool   `json:"capture"`
	Description string `json:"description"`
}

func createPayment(w http.ResponseWriter, r *http.Request) {
	// данные платежа
	reqBody := PaymentRequest{}
	reqBody.Amount.Value = "100.00" // сумма
	reqBody.Amount.Currency = "RUB"
	reqBody.Confirmation.Type = "redirect"
	reqBody.Confirmation.ReturnURL = returnURL
	reqBody.Capture = true
	reqBody.Description = "Заказ №1"

	jsonData, _ := json.Marshal(reqBody)

	// HTTP-запрос к ЮKassa
	client := &http.Client{}
	req, _ := http.NewRequest("POST", "https://api.yookassa.ru/v3/payments", bytes.NewBuffer(jsonData))
	req.SetBasicAuth(shopID, secretKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotence-Key", "уникальный_ключ_идентификации") // например, UUID

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Ошибка создания платежа", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	// парсим ответ, чтобы получить ссылку для оплаты
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	// отправляем пользователю ссылку на оплату
	confirmationURL := result["confirmation"].(map[string]interface{})["confirmation_url"].(string)
	http.Redirect(w, r, confirmationURL, http.StatusSeeOther)
}

func main() {
	http.HandleFunc("/pay", createPayment)
	http.ListenAndServe(":8080", nil)

}
