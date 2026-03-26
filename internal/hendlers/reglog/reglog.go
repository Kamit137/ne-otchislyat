package reglog

import (
	"encoding/json"

	"log"
	"ne-otchislyat/internal/sql"
	"ne-otchislyat/internal/token"
	"net/http"
	"text/template"
)

type registr struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

func IndexPage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("web/templates/register.html")
	if err != nil {
		http.Error(w, "Ошибка шаблона ", http.StatusInternalServerError)
		return
	}
	if err = tmpl.Execute(w, nil); err != nil {
		log.Fatal("Ошибка загрузки html lenta", err)
	}
}
func Reg(w http.ResponseWriter, r *http.Request) {
	var req registr
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "invalid_json",
			"message": "Invalid JSON registration",
		})
		return
	}
	defer r.Body.Close()

	if req.Email == "" || req.Password == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "missing_fields",
			"message": "Email and password are required",
		})
		return
	}

	err := sql.RegDb(req.Email, req.Password, req.Name)
	if err != nil {
		if err.Error() == "email exist" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{
				"error":   "email_exists",
				"message": "Email already exists",
			})
			return
		} else if err.Error() == "user not verified" {
			http.SetCookie(w, &http.Cookie{
				Name:     "verify_email",
				Value:    req.Email,
				Path:     "/",
				HttpOnly: false,
				MaxAge:   3600,
				SameSite: http.SameSiteLaxMode,
			})

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{
				"message":  "Registration updated",
				"redirect": "/verify",
				"email":    req.Email,
			})
			return
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error":   "registration_failed",
				"message": "Fatal Error registration.",
			})
			return
		}
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "verify_email",
		Value:    req.Email,
		Path:     "/",
		HttpOnly: false,
		MaxAge:   3600,
		SameSite: http.SameSiteLaxMode,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message":  "User registered successfully. Please verify your email.",
		"redirect": "/verify",
		"email":    req.Email,
		"status":   "verification_needed",
	})
}
func Login(w http.ResponseWriter, r *http.Request) {
	var req registr

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "invalid_json",
			"message": "Invalid JSON",
		})
		return
	}
	defer r.Body.Close()

	if req.Email == "" || req.Password == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "missing_fields",
			"message": "Email and password are required",
		})
		return
	}

	err := sql.LoginDb(req.Email, req.Password)
	if err != nil {
		switch err.Error() {
		case "user not found":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{
				"error":   "user_not_found",
				"message": "User not found",
			})
			return
		case "wrong password":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error":   "wrong_password",
				"message": "Wrong password",
			})
			return
		case "email not verified":
			http.SetCookie(w, &http.Cookie{
				Name:     "verify_email",
				Value:    req.Email,
				Path:     "/",
				HttpOnly: true,
				MaxAge:   3600,
			})

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{
				"error":    "email_not_verified",
				"message":  "Email not verified",
				"redirect": "/verify",
				"email":    req.Email,
			})
			return
		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error":   "login_failed",
				"message": "Login failed",
			})
			return
		}
	}

	token, err := token.GenerateToken(req.Email)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "token_failed",
			"message": "Failed to generate token",
		})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   864000,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"success":  "true",
		"message":  "Login successful",
		"redirect": "/",
		"email":    req.Email,
	})
}
func Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
