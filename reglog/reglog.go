package reglog

import (
	"encoding/json"
	"ne-otchislyat/sql"
	"net/http"
	"text/template"
)

type registr struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func IndexPage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("Project-3/src/index.html")
	if err != nil {
		http.Error(w, "Ошибка шаблона", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}
func Reg(w http.ResponseWriter, r *http.Request) {

	var req registr

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if req.Email == "" || req.Password == "" {
		http.Error(w, "Email and password are required", http.StatusBadRequest)
		return
	}

	err := sql.RegDb(req.Email, req.Password)
	if err != nil {
		if err.Error() == "email exist" {
			http.Error(w, "Email already exists", http.StatusConflict)
		} else {
			http.Error(w, "Registration failed", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "User registered successfully",
		"email":   req.Email,
	})
}

func Login(w http.ResponseWriter, r *http.Request) {
	var req registr

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if req.Email == "" || req.Password == "" {
		http.Error(w, "Email and password are required", http.StatusBadRequest)
		return
	}

	err := sql.LoginDb(req.Email, req.Password)
	if err != nil {
		switch err.Error() {
		case "user not found":
			http.Error(w, "User not found", http.StatusNotFound)
		case "wrong password":
			http.Error(w, "Wrong password", http.StatusUnauthorized)
		default:
			http.Error(w, "Login failed", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Login successful",
		"email":   req.Email,
	})
}
