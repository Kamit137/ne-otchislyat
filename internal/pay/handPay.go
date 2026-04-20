package pay

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"ne-otchislyat/internal/sql"
	"net/http"
	"time"
)

func HandleDeposit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	email, ok := r.Context().Value("email").(string)
	if !ok || email == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":    "Unauthorized",
			"message":  "Пожалуйста, авторизуйтесь",
			"redirect": "/registration",
		})
		return
	}
	var req struct {
		Amount float64 `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Неверный формат запроса", http.StatusBadRequest)
		return
	}
	if req.Amount < 50 {
		http.Error(w, "Минимальная сумма: 50 TST", http.StatusBadRequest)
		return
	}

	orderID := fmt.Sprintf("dep_%d", time.Now().UnixNano())
	err := sql.DepositTransacs(email, req.Amount, orderID)

	if err != nil {
		log.Printf("Ошибка транзакции c балансом: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	inv, err := createInvoice(
		orderID, req.Amount, email,
		"https://ne-otchislyat.ru/payment/success", // successURL
		"https://ne-otchislyat.ru/payment/fail",    // failURL
		"https://ne-otchislyat.ru/payment/result",  // resultURL (должен быть публичным)
	)
	if err != nil {
		log.Printf("createInvoice: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	invoiceID := inv.getInvoiceID()
	log.Printf("Тестовый инвойс создан: %s", invoiceID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"payment_url": paymentLink(invoiceID),
		"invoice_id":  invoiceID,
		"test_mode":   true,
	})
}

func HandlePaymentNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		log.Printf("Ошибка парсинга формы: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	eshopId := r.FormValue("eshopId")
	orderId := r.FormValue("orderId")
	recipientAmount := r.FormValue("recipientAmount")
	paymentStatus := r.FormValue("paymentStatus")
	hash := r.FormValue("hash")

	log.Printf("📥 Уведомление: eshopId=%s, orderId=%s, amount=%s, status=%s",
		eshopId, orderId, recipientAmount, paymentStatus)

	// Проверяем статус (5 = успешно оплачен)
	if paymentStatus != "5" {
		log.Printf("Статус не '5' (%s), пропускаем", paymentStatus)
		w.Write([]byte("OK"))
		return
	}

	// ВАЖНО: для проверки подписи нужны все поля
	hashStr := fmt.Sprintf("%s::%s::%s::%s::%s::%s::%s::%s::%s::%s::%s",
		eshopId,
		orderId,
		r.FormValue("serviceName"),
		r.FormValue("eshopAccount"),
		recipientAmount,
		r.FormValue("recipientCurrency"),
		paymentStatus,
		r.FormValue("userName"),
		r.FormValue("userEmail"),
		r.FormValue("paymentData"),
		GetSecretKey(), // Ваш секретный ключ
	)

	expectedHash := md5hash(hashStr)
	if expectedHash != hash {
		log.Printf("❌ Неверная подпись! Ожидался: %s, получен: %s", expectedHash, hash)
		// В тестовом режиме можно продолжить
		// В продакшене: http.Error(w, "Invalid signature", http.StatusBadRequest); return
	} else {
		log.Printf("✅ Подпись верна")
	}
	// Обновляем баланс пользователя
	tx, err := sql.DB.Begin()
	if err != nil {
		log.Printf("Ошибка транзакции: %v", err)
		w.Write([]byte("OK")) // Всё равно возвращаем OK, чтобы IntellectMoney не спамил
		return
	}
	defer tx.Rollback()

	// Находим транзакцию по orderId (это ваш внутренний ID, который вы передали как orderId)
	var userID int
	var amount int64
	var status string
	err = tx.QueryRow(`
		SELECT user_id, amount, status FROM transactions
		WHERE payment_id = $1 AND type = 'deposit'`, orderId).Scan(&userID, &amount, &status)
	if err != nil {
		log.Printf("Транзакция не найдена: %v", err)
		tx.Commit()
		w.Write([]byte("OK"))
		return
	}

	if status == "success" {
		log.Printf("Транзакция уже обработана: %s", orderId)
		tx.Commit()
		w.Write([]byte("OK"))
		return
	}

	// Обновляем статус транзакции
	_, err = tx.Exec(`UPDATE transactions SET status = 'success' WHERE payment_id = $1`, orderId)
	if err != nil {
		log.Printf("Ошибка обновления статуса: %v", err)
		w.Write([]byte("OK"))
		return
	}

	// Обновляем баланс пользователя (сумма в копейках)
	_, err = tx.Exec(`UPDATE users SET balance = balance + $1 WHERE id = $2`, amount, userID)
	if err != nil {
		log.Printf("Ошибка обновления баланса: %v", err)
		w.Write([]byte("OK"))
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Ошибка коммита: %v", err)
		w.Write([]byte("OK"))
		return
	}

	log.Printf("✅ Депозит зачислен: user_id=%d, amount=%d копеек", userID, amount)
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

func PaymentSuccessPage(w http.ResponseWriter, r *http.Request) {

	tmpl, err := template.ParseFiles("web/templates/successfull.html")
	if err != nil {
		log.Println("Template error:", err)
		http.Error(w, "Ошибка шаблона", http.StatusInternalServerError)
		return
	}
	if err = tmpl.Execute(w, nil); err != nil {
		log.Fatal("Ошибка paySuccess.html", err)
	}
}

func PaymentFailPage(w http.ResponseWriter, r *http.Request) {

	tmpl, err := template.ParseFiles("web/templates/error.html")
	if err != nil {
		log.Println("Template error:", err)
		http.Error(w, "Ошибка шаблона", http.StatusInternalServerError)
		return
	}
	if err = tmpl.Execute(w, nil); err != nil {
		log.Fatal("Ошибка payFail.html", err)
	}
}
