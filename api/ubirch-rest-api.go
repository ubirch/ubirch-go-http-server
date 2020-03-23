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
				w.WriteHeader(resp.Code)
				w.Write(resp.Content)
			}
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message": "http method not implemented"}`))
		}
	}
}

type HTTPServer struct {
	RequestChan  chan []byte
	ResponseChan chan Response
}

type Response struct {
	Code    int
	Content []byte
}

func (srv *HTTPServer) Listen(endpoint string, ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	s := &http.Server{Addr: ":8080"}
	http.HandleFunc(endpoint, handleRequest(srv.RequestChan, srv.ResponseChan))

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
