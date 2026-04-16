package pay

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
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

// ─── Создание инвойса

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
		c.eshopID,   // 1. eshopId
		orderID,     // 2. orderId
		serviceName, // 3. serviceName
		amountStr,   // 4. recipientAmount
		currency,    // 5. recipientCurrency
		"",          // 6. userName (пустая строка, если не передаем)
		email,       // 7. email
		successURL,  // 8. successUrl
		failURL,     // 9. failUrl
		"",          // 10. backUrl (пустая строка)
		resultURL,   // 11. resultUrl
		"",          // 12. expireDate (пустая строка)
		"",          // 13. holdMode (пустая строка)
		"",          // 14. preference (пустая строка)
		c.secretKey, // 15. secretKey
	)

	paramHash := md5hash(hashStr)

	log.Printf("Хеш-строка: %s", hashStr)
	log.Printf("MD5 хеш: %s", paramHash)

	body := map[string]interface{}{
		"eshopId":           c.eshopID,
		"orderId":           orderID,
		"serviceName":       serviceName,
		"recipientAmount":   amountStr,
		"recipientCurrency": currency,
		"userName":          "", // Обязательно добавить!
		"email":             email,
		"successUrl":        successURL,
		"failUrl":           failURL,
		"backUrl":           "", // Обязательно добавить!
		"resultUrl":         resultURL,
		"expireDate":        "", // Обязательно добавить!
		"holdMode":          "", // Обязательно добавить!
		"preference":        "", // Обязательно добавить!
		"hash":              paramHash,
	}

	jsonBody, _ := json.Marshal(body)
	log.Printf("Запрос к IntellectMoney: %s", string(jsonBody))

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
	log.Printf("IntellectMoney createInvoice (статус %d): %s", resp.StatusCode, string(respBody))

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

func GetSecretKey() string {
	return c.secretKey
}
