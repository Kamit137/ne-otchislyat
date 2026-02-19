package profile

import (
	"encoding/json"
	"log"
	"ne-otchislyat/sql"
	"net/http"
	"text/template"
)

func IndexPage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("Project-3/src/profile.html")
	if err != nil {
		log.Println("Template error:", err)
		http.Error(w, "Ошибка шаблона", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

func ProfilePrint(w http.ResponseWriter, r *http.Request) {
	// Получаем email из контекста (установлен middleware)
	email, ok := r.Context().Value("email").(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Получаем данные профиля
	prof, err := sql.GetInfProfile(email)
	if err != nil {
		log.Println("GetInfProfile error:", err)
		http.Error(w, "Failed to get profile", http.StatusInternalServerError)
		return
	}

	// Отправляем JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(prof)
}
