package profile

import (
	"encoding/json"
	"log"
	"ne-otchislyat/internal/sql"
	"net/http"
	"strconv"
	"text/template"
	"time"
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

	err, avtor, userID := sql.AddVakans(email, NewCard.Title, NewCard.Discription, NewCard.Tag, NewCard.Price)
	if err != nil || avtor == "" || userID == 0 {
		log.Println("AddItem error:", err)
		http.Error(w, "Failed to add card: "+err.Error(), http.StatusBadRequest)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{
		"id":          strconv.Itoa(userID),
		"avtor":       avtor,
		"title":       NewCard.Title,
		"discription": NewCard.Discription,
		"price":       strconv.Itoa(NewCard.Price),
	})
}

func Exit(w http.ResponseWriter, r *http.Request) {
	// Удаляем куку с путем /
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
	})

	// Удаляем куку с путем /registration (если есть)

	http.Redirect(w, r, "/registration", http.StatusFound)
}

func RemoveCard(w http.ResponseWriter, r *http.Request) {
	email, ok := r.Context().Value("email").(string)
	if !ok {
		http.Redirect(w, r, "/registration", http.StatusFound)
		return
	}
	var CardId struct {
		Id int `json:"id"`
	}
	err := json.NewDecoder(r.Body).Decode(&CardId)
	if err != nil {
		log.Fatal("нет id карточки")
		http.Error(w, "Не получилось взять email", http.StatusFound)
		return
	}
	err = sql.RemoveVakans(email, CardId.Id)
	if err != nil {
		http.Error(w, "Не удалось удалить карточку", http.StatusBadRequest)

	}
	w.WriteHeader(http.StatusOK)
}
