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
		page            int
		tagFilters      []string
		itemType        string
		priceUpDownFals string
	}

	err := json.NewDecoder(r.Body).Decode(&Zapros)
	if err != nil || Zapros.page < 1 {
		Zapros.page = 1
	}

	cards, err := sql.GetCards(Zapros.page, Zapros.tagFilters, Zapros.itemType, Zapros.priceUpDownFals)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cards)
}
