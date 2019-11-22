package rest_api

import (
	"log"
	"net/http"
)

func sign(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == "POST" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "post called"}`))
	} else {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": "http method not implemented"}`))
	}
}

func verify(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == "POST" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "post called"}`))
	} else {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": "http method not implemented"}`))
	}
}

type HTTPServer struct {
	handler chan []byte
}

//noinspection GoUnhandledErrorResult
func (srv *HTTPServer) Listen() {
	http.HandleFunc("/sign", sign)
	http.HandleFunc("/verify", verify)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
