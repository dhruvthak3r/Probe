package main

import (
	"net/http"
)

func main() {
	http.HandleFunc("/", home)
	http.ListenAndServe(":8080", nil)
}

func home(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello, World!"))
}
