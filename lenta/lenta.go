package lenta

import (
	"encoding/json"
	"log"
	"ne-otchislyat/sql"
	"net/http"
	"strconv"
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

func GiveLentaZakaz(w http.ResponseWriter, r *http.Request) {
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	cards, err := sql.GetCards(page, "zakaz")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cards)
}

func GiveLentaVakans(w http.ResponseWriter, r *http.Request) {
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	cards, err := sql.GetCards(page, "vakans")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cards)
}
