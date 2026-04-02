package main

import (
	"ne-otchislyat/internal/hendlers/favorite"
	"ne-otchislyat/internal/hendlers/lenta"
	"ne-otchislyat/internal/hendlers/profile"
	"ne-otchislyat/internal/hendlers/reglog"
	"ne-otchislyat/internal/hendlers/verify"

	"ne-otchislyat/internal/pay"
	"ne-otchislyat/internal/sql"
	"ne-otchislyat/internal/token"

	"log"
	"net/http"
)

func main() {
	if err := sql.InitDB(); err != nil {
		log.Fatal("Ошибка инициализации БД:", err)
	}

	http.HandleFunc("/registration", reglog.IndexPage)

	http.HandleFunc("/reg", reglog.Reg)
	http.HandleFunc("/login", reglog.Login)
	http.HandleFunc("/verify", verify.IndexPage)
	http.HandleFunc("/api/verify", verify.ValidateCod)
	http.HandleFunc("/", lenta.IndexPage)
	http.HandleFunc("/api/lenta", lenta.GiveLenta)

	http.HandleFunc("/profile", token.AuthMiddleware(profile.IndexPage))
	http.HandleFunc("/api/profile", token.AuthMiddleware(profile.ProfilePrint))
	http.HandleFunc("/api/addCard", token.AuthMiddleware(profile.AddCard))

	http.HandleFunc("/api/deposit", token.AuthMiddleware(pay.HandleDeposit))
	http.HandleFunc("/api/yookassa/webhook", pay.HandleWebhook)
	http.HandleFunc("/api/balance", token.AuthMiddleware(pay.GetBalance))
	http.HandleFunc("/api/order/create", token.AuthMiddleware(pay.CreateOrder))
	http.HandleFunc("/api/order/complete", token.AuthMiddleware(pay.CompleteOrder))
	http.HandleFunc("/api/order/cancel", token.AuthMiddleware(pay.CancelOrder))

	http.HandleFunc("/favorite", token.AuthMiddleware(favorite.IndexPage))
	http.HandleFunc("/api/printfavorite", token.AuthMiddleware(favorite.GetCards))
	http.HandleFunc("/api/favorites", token.AuthMiddleware(favorite.AddCard))

	http.HandleFunc("/logout", reglog.Logout)

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	http.Handle("/templates/", http.StripPrefix("/templates/", http.FileServer(http.Dir("web/templates"))))
	http.ListenAndServe(":8080", nil)
}
