package lenta

import (
	"encoding/json"
	"log"
	"ne-otchislyat/internal/sql"
	"net/http"

	"text/template"
)

func IndexPage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("web/templates/lenta.html")
	if err != nil {
		log.Println("Template error:", err)
		http.Error(w, "Ошибка шаблона", http.StatusInternalServerError)
		return
	}
	if err = tmpl.Execute(w, nil); err != nil {
		log.Fatal("Ошибка загрузки html lenta", err)
	}

}

func GiveLenta(w http.ResponseWriter, r *http.Request) {
	email, ok := r.Context().Value("email").(string)
	if !ok {
		email = ""
	}
	var Zapros struct {
		Page             int    `json:"page"`
		Tag              string `json:"tag"`
		PriceUpDownFalse string `json:"priceUpDownFalse"`
	}
	err := json.NewDecoder(r.Body).Decode(&Zapros)
	if err != nil || Zapros.Page < 1 {
		Zapros.Page = 1
	}

	cards, err := sql.GetVakans(email, Zapros.Page, Zapros.Tag, Zapros.PriceUpDownFalse)
	if err != nil {
		log.Printf("Ошибка загрузки ленты: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to load vacancies"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cards)
}
