package favorite

import (
	"encoding/json"
	"log"
	"ne-otchislyat/internal/sql"
	"net/http"
	"text/template"
)

func IndexPage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("web/templates/favorites.html")
	if err != nil {
		log.Println("Template error:", err)
		http.Error(w, "Ошибка шаблона favorites", http.StatusInternalServerError)
		return
	}
	if err = tmpl.Execute(w, nil); err != nil {
		log.Fatal("Ошибка загрузки html lenta", err)
	}
}

func GetCards(w http.ResponseWriter, r *http.Request) {
	email, ok := r.Context().Value("email").(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	cards, err := sql.GetFavorite(email)
	if err != nil {
		log.Println("Ошибка favorites ", err)
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(cards)
	if err != nil {
		log.Fatal("Ошибка отправки json в getcards favorite", err)
	}
}

func AddCard(w http.ResponseWriter, r *http.Request) {
	email, ok := r.Context().Value("email").(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

}
