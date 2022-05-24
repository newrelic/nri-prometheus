package main

import (
	"net/http"
)

func headers(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func main() {
	http.HandleFunc("/remote-write-mock", headers)

	http.ListenAndServe(":8000", nil)
}
