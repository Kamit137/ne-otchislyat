package lenta

import (
	"encoding/json"

	"log"
	"ne-otchislyat/internal/sql"
	"net/http"
	"os"
	"text/template"
)

func IndexPage(w http.ResponseWriter, r *http.Request) {
	//
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
	email, _ := r.Context().Value("email").(string)

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
	balance, frozenBalance, err := sql.GetUserBalance(email)
	if err != nil {
		balance, frozenBalance = 0, 0
	}
	var result struct {
		Balance  int64        `json:"balance"`
		FBalance int64        `json:"frozenBalance"`
		Cards    []sql.Vakans `json:"cards"`
	}
	result.Cards = cards
	result.Balance = balance
	result.FBalance = frozenBalance
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func DownloadOferta(w http.ResponseWriter, r *http.Request) {
	filename := "oferta.pdf"
	filePath := "oferta.pdf"

	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		http.Error(w, "Файл не найден", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(fileContent)
}
