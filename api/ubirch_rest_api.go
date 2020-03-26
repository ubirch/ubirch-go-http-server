package api

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
)

func returnErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	log.Println(message)
	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(message))
}

func handleRequest(srv *HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// only accept POST requests
		if r.Method != "POST" {
			returnErrorResponse(w, http.StatusNotFound, fmt.Sprintf("%s not implemented", r.Method))
			return
		}

		// read request body
		reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			returnErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("error reading request body: %v", err))
			return
		}

		srv.ReceiveHandler <- reqBody

		// wait for response from ubirch backend to be forwarded
		select {
		case resp := <-srv.ResponseHandler:
			w.WriteHeader(resp.Code)
			for k, v := range resp.Header {
				w.Header().Set(k, v[0])
			}
			w.Write(resp.Content)
		}
	}
}

type HTTPServer struct {
	ReceiveHandler  chan []byte
	ResponseHandler chan Response
	Endpoint        string
}

type Response struct {
	Code    int
	Header  map[string][]string
	Content []byte
}

func (srv *HTTPServer) Listen(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	s := &http.Server{Addr: ":8080"}
	http.HandleFunc(srv.Endpoint, handleRequest(srv))

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
