package lenta

import (
	"encoding/json"
	"log"
	"ne-otchislyat/sql"
	"net/http"

	"text/template"
)

func IndexPage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("Project-3/src/lenta.html")
	if err != nil {
		log.Println("Template error:", err)
		http.Error(w, "Ошибка шаблона", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

func GiveLenta(w http.ResponseWriter, r *http.Request) {
	var Zapros struct {
		Page            int      `json:"page"`
		TagFilters      []string `json:"tagFilters"`
		ItemType        string   `json:"itemType"`
		PriceUpDownFals string   `json:"priceUpDownFalse"`
	}
	err := json.NewDecoder(r.Body).Decode(&Zapros)
	if err != nil || Zapros.Page < 1 {
		Zapros.Page = 1
	}

	cards, err := sql.GetCards(Zapros.Page, Zapros.TagFilters, Zapros.ItemType, Zapros.PriceUpDownFals)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cards)
}
