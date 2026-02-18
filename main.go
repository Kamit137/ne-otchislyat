package main

import (
	"fmt"
	"ne-otchislyat/reglog"
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

	// Статические файлы (CSS, JS)
	http.Handle("/src/", http.StripPrefix("/src/", http.FileServer(http.Dir("Project-3/src/css"))))

	http.ListenAndServe(":8080", nil)
}
