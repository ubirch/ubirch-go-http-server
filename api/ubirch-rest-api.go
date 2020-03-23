package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
)

// helper function to determine if a list contains a certain string
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
			log.Printf("recieved %s request. (not implemented)", r.Method)
			returnErrorResponse(w, http.StatusNotFound, "http method not implemented")
			return
		}

		// read request body
		reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("error reading http request body: %v", err)
			returnErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("error reading request body: %v", err))
			return
		}

		// check if request body is a json object
		if stringInList("application/json", r.Header["Content-Type"]) {
			// get UUID from header
			uuidString := r.Header.Get("UUID")
			if uuidString == "" {
				log.Printf("missing UUID header")
				returnErrorResponse(w, http.StatusBadRequest, "missing UUID")
				return
			}
			id, err := uuid.Parse(uuidString)
			if err != nil {
				log.Printf("error parsing UUID: %v", err)
				returnErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("error parsing UUID: %v", err))
				return
			}

			// generate a sorted compact rendering of the json formatted request body before forwarding it to the signer
			var reqDump interface{}
			var compactSortedJson bytes.Buffer

			err = json.Unmarshal(reqBody, &reqDump)
			if err != nil {
				log.Printf("error parsing http request body to json: %v", err)
				returnErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("error parsing request body: %v", err))
				return
			}
			// json.Marshal sorts the keys
			sortedJson, _ := json.Marshal(reqDump)
			_ = json.Compact(&compactSortedJson, sortedJson)

			requestChan <- append(id[:], compactSortedJson.Bytes()...)

		} else {
			requestChan <- reqBody
		}

		// wait for response from ubirch backend to be forwarded
		//todo check performance
		select {
		case resp := <-responseChan:
			w.WriteHeader(resp.Code)
			for k, v := range resp.Header {
				w.Header().Set(k, v[0])
			}
			w.Write(resp.Content)
		}
	}
}

func returnErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(message))
}

type HTTPServer struct {
	ReceiveHandler  chan []byte
	ResponseHandler chan Response
}

type Response struct {
	Code    int
	Header  map[string][]string
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
