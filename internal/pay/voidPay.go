package pay

import (
	"encoding/json"
	"fmt"
	"log"
	"ne-otchislyat/internal/sql"
	"net/http"
)

// VoidHandleDeposit - тестовое пополнение без реальной оплаты
func VoidHandleDeposit(w http.ResponseWriter, r *http.Request) {
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
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "Неверный формат запроса"})
		return
	}

	if req.Amount < 1 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "Минимальная сумма: 1 ₽"})
		return
	}

	// Сразу зачисляем баланс (без реальной оплаты)
	err := sql.VoidDeposit(email, req.Amount)
	if err != nil {
		log.Printf("Ошибка пополнения: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Ошибка пополнения"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Баланс пополнен на %.2f ₽ (тестовый режим)", req.Amount),
		"amount":  req.Amount,
	})
}

// VoidHandlePaymentNotification - заглушка для уведомлений (не используется)
func VoidHandlePaymentNotification(w http.ResponseWriter, r *http.Request) {
	log.Println("⚠️ Заглушка уведомления об оплате (тестовый режим)")
	w.Write([]byte("OK"))
}
