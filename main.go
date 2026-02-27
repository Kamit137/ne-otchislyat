package main

import (
	"fmt"

	"ne-otchislyat/lenta"
	"ne-otchislyat/profile"
	"ne-otchislyat/reglog"
	"ne-otchislyat/token"

	"net/http"
)

func A(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "ab")
}
func main() {
	http.HandleFunc("/", reglog.IndexPage)

	http.HandleFunc("/reg", reglog.Reg)
	http.HandleFunc("/login", reglog.Login)
	http.HandleFunc("/a", A)

	http.HandleFunc("/lenta", token.AuthMiddleware(lenta.IndexPage))
	http.HandleFunc("/api/lenta", token.AuthMiddleware(lenta.GiveLenta))

	http.HandleFunc("/profile", token.AuthMiddleware(profile.IndexPage))
	http.HandleFunc("/api/profile", token.AuthMiddleware(profile.ProfilePrint))
	// http.HandleFunc("/api/addCard", token.AuthMiddleware(profile.AddCard))

	http.HandleFunc("/logout", reglog.Logout)

	http.Handle("/src/", http.StripPrefix("/src/", http.FileServer(http.Dir("Project-3/src"))))
	http.ListenAndServe(":8080", nil)
}
