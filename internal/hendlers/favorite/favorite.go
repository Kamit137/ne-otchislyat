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
		http.Error(w, "Failed GetCards", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(cards)
	if err != nil {
		log.Fatal("Ошибка отправки json в getcards favorite", err)
		http.Error(w, "Failed GetCards", http.StatusInternalServerError)
		return
	}
}

func AddCard(w http.ResponseWriter, r *http.Request) {
	log.Println("✅ Вызван AddCard")
	email, ok := r.Context().Value("email").(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	var req struct {
		ID int `json:"id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.ID == 0 {
		http.Error(w, "ID is required", http.StatusBadRequest)
		return
	}
	id_card := req.ID

	err := sql.Like(email, id_card)
	if err != nil {
		log.Println("ошибка добавления в избранное", err)
		http.Error(w, "Failed to add favorite", http.StatusInternalServerError)

		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
