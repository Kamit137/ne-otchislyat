package main

import (
	"ne-otchislyat/internal/hendlers/lenta"
	"ne-otchislyat/internal/hendlers/profile"
	"ne-otchislyat/internal/hendlers/reglog"
	"ne-otchislyat/internal/token"

	"net/http"
)

func main() {
	http.HandleFunc("/", reglog.IndexPage)

	http.HandleFunc("/reg", reglog.Reg)
	http.HandleFunc("/login", reglog.Login)

	http.HandleFunc("/lenta", token.AuthMiddleware(lenta.IndexPage))
	http.HandleFunc("/api/lenta", token.AuthMiddleware(lenta.GiveLenta))

	http.HandleFunc("/profile", token.AuthMiddleware(profile.IndexPage))
	http.HandleFunc("/api/profile", token.AuthMiddleware(profile.ProfilePrint))
	http.HandleFunc("/api/addCard", token.AuthMiddleware(profile.AddCard))

	http.HandleFunc("/logout", reglog.Logout)

	//http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	http.Handle("/templates/", http.StripPrefix("/templates/", http.FileServer(http.Dir("web/"))))
	http.ListenAndServe(":8080", nil)
}
