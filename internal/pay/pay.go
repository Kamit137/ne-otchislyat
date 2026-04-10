package pay

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"fmt"

	"log"
	"net/http"
	"net/url"

	"strings"
	"time"
)

// --- Конфигурация (замените на свои данные) ---
var (
	merchantLogin = "ne-otchislyat.ru"     // Ваш MerchantLogin
	password1     = "P7DZ3ID5v0W9znRuqKFI" // Пароль №1 для создания платежей
	password2     = "Xg5zLXc770dKr7VorLCS" // Пароль №2 для проверки уведомлений
	isTest        = true                   // true для тестов, false для боевого режима
)

// HashAlgorithm определяет алгоритм для подписи
type HashAlgorithm string

const (
	MD5    HashAlgorithm = "md5"
	SHA256 HashAlgorithm = "sha256"
)

// --- Генерация подписи ---
func calculateSignature(values url.Values, password string, algo HashAlgorithm) string {
	// Собираем строку для подписи согласно документации: https://docs.robokassa.ru/ru/quick-start#2368
	var signatureParts []string
	if algo == MD5 {
		signatureParts = append(signatureParts, values.Get("MerchantLogin"))
		signatureParts = append(signatureParts, values.Get("OutSum"))
		signatureParts = append(signatureParts, values.Get("InvId"))
		signatureParts = append(signatureParts, password)
	} else { // SHA256
		signatureParts = append(signatureParts, values.Get("MerchantLogin"))
		signatureParts = append(signatureParts, values.Get("OutSum"))
		signatureParts = append(signatureParts, values.Get("InvId"))
		signatureParts = append(signatureParts, password)
	}
	signatureString := strings.Join(signatureParts, ":")

	var hash []byte
	if algo == MD5 {
		hasher := md5.Sum([]byte(signatureString))
		hash = hasher[:]
	} else {
		hasher := sha256.Sum256([]byte(signatureString))
		hash = hasher[:]
	}
	return hex.EncodeToString(hash)
}

// --- 1. Создание платежа и получение URL для редиректа ---
// GeneratePaymentURL создает ссылку на оплату в Robokassa
func GeneratePaymentURL(amount float64, orderID string, email string, userParams map[string]string) (string, error) {
	// Формируем параметры запроса
	values := url.Values{}
	values.Set("MerchantLogin", merchantLogin)
	values.Set("OutSum", fmt.Sprintf("%.2f", amount))
	values.Set("InvId", orderID)
	values.Set("Description", "Пополнение баланса пользователя")
	values.Set("Email", email)
	values.Set("Culture", "ru") // Язык интерфейса

	// Добавляем пользовательские параметры (обязательно с префиксом shp_)
	for k, v := range userParams {
		values.Set("shp_"+k, v)
	}

	// Устанавливаем алгоритм (лучше SHA256)
	algo := SHA256
	values.Set("SignatureValue", calculateSignature(values, password1, algo))
	// В тестовом режиме нужно добавить параметр IsTest=1
	if isTest {
		values.Set("IsTest", "1")
	}

	// Формируем URL для редиректа
	paymentURL := "https://auth.robokassa.ru/Merchant/Index.aspx?" + values.Encode()
	log.Printf("✅ Сгенерирован URL для оплаты: %s", paymentURL)
	return paymentURL, nil
}

// --- 2. Обработчик для пополнения баланса (заменяет ваш HandleDeposit) ---
// HandleDepositRobokassa обрабатывает запрос на создание платежа
func HandleDepositRobokassa(w http.ResponseWriter, r *http.Request) {
	log.Println("💰 Обработка депозита через Robokassa")

	// Получаем email пользователя из контекста (как в вашем коде)
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

	if req.Amount <= 0 {
		http.Error(w, "Сумма должна быть больше 0", http.StatusBadRequest)
		return
	}
	log.Printf("📝 Запрос на депозит: email=%s, amount=%.2f", email, req.Amount)

	// Генерируем уникальный ID заказа (InvId)
	orderID := fmt.Sprintf("%d", time.Now().UnixNano())

	// Передаем email как пользовательский параметр, чтобы потом знать, кому начислить
	userParams := map[string]string{
		"email": email,
	}

	// Создаем ссылку на оплату
	paymentURL, err := GeneratePaymentURL(req.Amount, orderID, email, userParams)
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

// --- 3. Обработка уведомлений от Robokassa (Result URL) ---
// HandlePaymentNotification обрабатывает POST-запросы от Robokassa
func HandlePaymentNotification(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		log.Printf("❌ Ошибка парсинга формы: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Получаем параметры от Robokassa
	outSum := r.FormValue("OutSum")
	invId := r.FormValue("InvId")
	signature := r.FormValue("SignatureValue")

	// ВНИМАНИЕ: Для проверки подписи используем password2!
	// Собираем параметры для проверки
	values := url.Values{}
	values.Set("OutSum", outSum)
	values.Set("InvId", invId)
	values.Set("MerchantLogin", merchantLogin) // Нужен для формирования подписи

	// Проверяем подпись (алгоритм должен совпадать с тем, что использовали при создании)
	expectedSignature := calculateSignature(values, password2, SHA256)
	if signature != expectedSignature {
		log.Printf("❌ Ошибка проверки подписи для заказа %s", invId)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("FAIL"))
		return
	}

	// Извлекаем пользовательские параметры (shp_email)
	email := r.FormValue("shp_email")
	if email == "" {
		log.Printf("⚠️ Не передан email в shp_email для заказа %s", invId)
		// Но все равно можем попробовать найти пользователя по-другому
	}

	// --- ВАША ЛОГИКА ЗАЧИСЛЕНИЯ СРЕДСТВ ---
	// 1. Найти пользователя по email
	// 2. Начислить ему сумму outSum
	// 3. Записать операцию в БД
	log.Printf("✅ Платеж подтвержден! Заказ: %s, Сумма: %s, Email: %s", invId, outSum, email)

	// Обязательно вернуть "OK<InvId>", иначе Robokassa будет повторять запросы [citation:2][citation:8]
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("OK%s", invId)))
}
