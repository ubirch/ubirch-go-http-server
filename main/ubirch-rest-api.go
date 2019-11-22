package main

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
)

func sign(srv *HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("error reading http request body: %v", err)
			return
		}

		if r.Method == "POST" {
			log.Println(reqBody)
			srv.signHandler <- reqBody
			w.WriteHeader(http.StatusOK)
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message": "http method not implemented"}`))
		}
	}
}

func verify(srv *HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("error reading http request body: %v", err)
			return
		}

		if r.Method == "POST" {
			log.Println(reqBody)
			srv.verifyHandler <- reqBody
			w.WriteHeader(http.StatusOK)
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message": "http method not implemented"}`))
		}
	}
}

type HTTPServer struct {
	signHandler   chan []byte
	verifyHandler chan []byte
}

func (srv *HTTPServer) Listen(ctx context.Context, wg *sync.WaitGroup) error {

	http.HandleFunc("/sign", sign(srv))
	http.HandleFunc("/verify", verify(srv))

	return http.ListenAndServe(":8080", nil)
}
