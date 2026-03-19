package profile

import (
	"encoding/json"
	"log"
	"ne-otchislyat/internal/sql"
	"net/http"
	"text/template"
)

func IndexPage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("web/templates/profile.html")
	if err != nil {
		log.Println("Template error:", err)
		http.Error(w, "Ошибка шаблона", http.StatusInternalServerError)
		return
	}
	if err = tmpl.Execute(w, nil); err != nil {
		log.Fatal("Ошибка загрузки html lenta", err)
	}
}

func ProfilePrint(w http.ResponseWriter, r *http.Request) {

	email, ok := r.Context().Value("email").(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	prof, err := sql.GetInfProfile(email)
	if err != nil {
		log.Println("GetInfProfile error:", err)
		http.Error(w, "Failed to get profile", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(prof)
	if err != nil {
		log.Fatal("ошибка отправки json в профильпринт")
	}

}

func WriteInProfile(w http.ResponseWriter, r *http.Request) {
	email, ok := r.Context().Value("email").(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	type UpdateData struct {
		Name      string `json:"name"`
		Password  string `json:"password"`
		TgUs      string `json:"tgUs"`
		Recvizits int    `json:"recvizits"`
	}
	var updateData UpdateData
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	err := sql.UpdateProf(updateData.Name, updateData.Password, updateData.TgUs, updateData.Recvizits, email)
	if err != nil {
		http.Error(w, "Invalid write infProf", http.StatusBadRequest)
	}
	err = json.NewEncoder(w).Encode(map[string]string{
		"message": "Profile updated successfully",
	})
	if err != nil {
		log.Fatal("Ошибка обновления инфы в профиль", err)
	}
}
func AddCard(w http.ResponseWriter, r *http.Request) {
	email, ok := r.Context().Value("email").(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var NewCard struct {
		Avtor       string `json:"name"`
		Title       string `json:"title"`
		Discription string `json:"discription"`
		Price       int    `json:"price"`
		Tag         string `json:"tags"`
	}

	if err := json.NewDecoder(r.Body).Decode(&NewCard); err != nil {
		log.Println("JSON decode error:", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	err := sql.AddVakans(email, NewCard.Avtor, NewCard.Title, NewCard.Discription, NewCard.Tag, NewCard.Price)
	if err != nil {
		log.Println("AddItem error:", err)
		http.Error(w, "Failed to add card: "+err.Error(), http.StatusBadRequest)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Card added successfully",
	})
}
