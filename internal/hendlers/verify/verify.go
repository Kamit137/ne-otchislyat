package verify

import (
	"encoding/json"
	"log"
	"ne-otchislyat/internal/sql"
	"ne-otchislyat/internal/token"
	"net/http"
	"text/template"
	"time"
)

func IndexPage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("web/templates/vrfk.html")
	if err != nil {
		http.Error(w, "Ошибка шаблона", http.StatusInternalServerError)
		return
	}
	if err = tmpl.Execute(w, nil); err != nil {
		log.Fatal("Ошибка загрузки html lenta", err)
	}
}

func ValidateCod(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("verify_email")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"status":   "no_cookie",
			"message":  "Email cookie not found",
			"redirect": "/",
		})
		return
	}

	email := cookie.Value

	var req struct {
		Code string `json:"code"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON validate cod", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if email == "" || req.Code == "" {
		http.Error(w, "Email and code are required", http.StatusBadRequest)
		return
	}

	storedCode, timeLive, verify, err := sql.VerifyCodeInSql(email)
	if err != nil {
		if err.Error() == "user not found" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{
				"status":   "user_not_found",
				"message":  "Email not found. Please register again.",
				"redirect": "/",
			})
			return
		}
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if verify {
		token, err := token.GenerateToken(email)
		if err != nil {
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			MaxAge:   864000,
		})

		http.SetCookie(w, &http.Cookie{
			Name:     "verify_email",
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			MaxAge:   -1,
		})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status":   "already_verified",
			"message":  "Email already verified",
			"redirect": "/lenta",
		})
		return
	}

	if time.Now().After(timeLive) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusGone)
		json.NewEncoder(w).Encode(map[string]string{
			"status":   "code_expired",
			"redirect": "/reg",
			"message":  "Verification code expired",
		})
		return
	}

	if storedCode != req.Code {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "invalid_code",
			"message": "Invalid verification code",
		})
		return
	}

	err = sql.UpdateUserVerified(email)
	if err != nil {
		http.Error(w, "Ошибка активации: "+err.Error(), http.StatusInternalServerError)
		return
	}

	token, err := token.GenerateToken(email)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   864000,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "verify_email",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "success",
		"message":  "Email verified successfully",
		"redirect": "/lenta",
	})
}
