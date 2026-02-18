package main

import (
	"ne-otchislyat/reglog"
	"net/http"
)

func main() {
	http.HandleFunc("/", reglog.IndexPage)

	// API эндпоинты
	http.HandleFunc("/reg", reglog.Reg)
	http.HandleFunc("/login", reglog.Login)

	// Статические файлы (CSS, JS)
	http.Handle("/src/", http.StripPrefix("/src/", http.FileServer(http.Dir("Project-3/src/css"))))

	http.ListenAndServe(":8080", nil)
}
