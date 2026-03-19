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
	var Zapros struct {
		Page            int    `json:"page"`
		Tag             string `json:"tagFilter"`
		ItemType        string `json:"itemType"`
		PriceUpDownFals string `json:"priceUpDownFalse"`
	}
	err := json.NewDecoder(r.Body).Decode(&Zapros)
	if err != nil || Zapros.Page < 1 {
		Zapros.Page = 1
	}

	cards, err := sql.GetVakans(Zapros.Page, Zapros.Tag, Zapros.PriceUpDownFals)
	if err != nil {
		log.Fatal("Ошибка загрузки ленты", err)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cards)
}
