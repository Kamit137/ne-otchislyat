package dopfunc

import (
	"encoding/json"
	"ne-otchislyat/internal/sql"
	"net/http"
	"time"
)

type VerifyRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

func VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req VerifyRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	storedCode, timeLive, verified, err := sql.VerifyCodeInSql(req.Email)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if verified {
		http.Error(w, "Email already verified", http.StatusBadRequest)
		return
	}

	if storedCode != req.Code {
		http.Error(w, "Invalid verification code", http.StatusBadRequest)
		return
	}

	if time.Now().After(timeLive) {
		http.Error(w, "Verification code expired", http.StatusBadRequest)
		return
	}

	err = sql.UpdateUserVerified(req.Email)
	if err != nil {
		http.Error(w, "Failed to update verification status", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Email successfully verified",
	})
}
