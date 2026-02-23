package profile

import (
	"encoding/json"
	"log"
	"ne-otchislyat/sql"
	"net/http"
	"text/template"
)

func IndexPage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("Project-3/src/profile.html")
	if err != nil {
		log.Println("Template error:", err)
		http.Error(w, "Ошибка шаблона", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
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
	json.NewEncoder(w).Encode(prof)
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
		IsCompany bool   `json:"isCompany"`
		Rating    int    `json:"rating"`
		TgUs      string `json:"tgUs"`
		Recvizits int64  `json:"recvizits"`
	}
	var updateData UpdateData
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	err := sql.UpdateProf(updateData.Name, updateData.Password, updateData.IsCompany, updateData.Rating, updateData.TgUs, updateData.Recvizits, email)
	if err != nil {
		http.Error(w, "Invalid write infProf", http.StatusBadRequest)
	}
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Profile updated successfully",
	})
}
