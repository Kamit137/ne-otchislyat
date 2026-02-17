package main

import (
	"ne-otchislyat/reglog"
	"net/http"
)

func main() {

	http.HandleFunc("/reg", reglog.Reg)
	http.HandleFunc("/login", reglog.Login)

	http.ListenAndServe(":8080", nil)
}
