package api

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
			srv.SignHandler <- reqBody
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
			srv.VerifyHandler <- reqBody
			w.WriteHeader(http.StatusOK)
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message": "http method not implemented"}`))
		}
	}
}

type HTTPServer struct {
	SignHandler   chan []byte
	VerifyHandler chan []byte
}

func (srv *HTTPServer) Listen(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	s := &http.Server{Addr: ":8080"}
	http.HandleFunc("/sign", sign(srv))
	http.HandleFunc("/verify", verify(srv))

	go func() {
		<-ctx.Done()
		log.Println("shutting down http server")
		s.Shutdown(ctx)
		return
	}()

	err := s.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatalf("error starting http service: %v", err)
	}
}
