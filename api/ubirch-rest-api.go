package api

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
)

func handleRequest(requestChan chan []byte, responseChan chan Response) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("error reading http request body: %v", err)
			return
		}

		if r.Method == "POST" {
			// get the request body
			log.Println(reqBody)
			requestChan <- reqBody

			// wait for response from ubirch backend to be forwarded
			select {
			case resp := <-responseChan:
				w.WriteHeader(resp.code)
				w.Write(resp.content)
			}
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message": "http method not implemented"}`))
		}
	}
}

type HTTPServer struct {
	SigningRequestChan       chan []byte
	SigningResponseChan      chan Response
	VerificationRequestChan  chan []byte
	VerificationResponseChan chan Response
}

type Response struct {
	code    int
	content []byte
}

func (srv *HTTPServer) Listen(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	s := &http.Server{Addr: ":8080"}
	http.HandleFunc("/sign", handleRequest(srv.SigningRequestChan, srv.SigningResponseChan))
	http.HandleFunc("/verify", handleRequest(srv.VerificationRequestChan, srv.VerificationResponseChan))

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
