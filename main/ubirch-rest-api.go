package main

import (
	"io/ioutil"
	"log"
	"net/http"
)

func sign(srv *HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Fatal(err)
		}

		if r.Method == "POST" {
			srv.handler<-reqBody
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message": "http method not implemented"}`))
		}
	}
}

func verify(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == "POST" {
		reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("%s", reqBody)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "got it!"}`))
	} else {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": "http method not implemented"}`))
	}
}

type HTTPServer struct {
	handler chan []byte
}


func (srv *HTTPServer) Listen() error {

    http.HandleFunc("/sign", sign(srv))
	http.HandleFunc("/verify", verify(srv))

	return http.ListenAndServe(":8080", nil)
}
