package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
)

func stringInList(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func handleRequest(requestChan chan []byte, responseChan chan Response) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// only accept POST requests
		if r.Method != "POST" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message": "http method not implemented"}`))
		}

		// read request body
		reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("error reading http request body: %v", err)
			return
		}

		//
		if stringInList("application/json", r.Header["Content-Type"]) {
			// make a sorted compact rendering of the json formatted request body before forwarding it to the signer
			var reqDump interface{}
			err = json.Unmarshal(reqBody, &reqDump)
			if err != nil {
				log.Printf("error parsing http request body to json: %v", err)
				return
			}
			// json.Marshal sorts the keys
			sortedJson, _ := json.Marshal(reqDump)
			var compactSortedJson bytes.Buffer
			err = json.Compact(&compactSortedJson, sortedJson)

			//requestChan <- append(compactSortedJson.Bytes())

		} else {
			requestChan <- reqBody
		}

		// wait for response from ubirch backend to be forwarded
		// TODO check performance
		select {
		case resp := <-responseChan:
			w.WriteHeader(resp.Code)
			w.Write(resp.Content)
		}
	}
}

type HTTPServer struct {
	ReceiveHandler  chan []byte
	ResponseHandler chan Response
}

type Response struct {
	Code    int
	Content []byte
}

func (srv *HTTPServer) Listen(endpoint string, ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	s := &http.Server{Addr: ":8080"}
	http.HandleFunc(endpoint, handleRequest(srv.ReceiveHandler, srv.ResponseHandler))

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
