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
	"strings"
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

		// check if request body is a json object
		if stringInList("application/json", r.Header["Content-Type"]) {
			// get UUID from URL path
			pathID := strings.TrimPrefix(r.URL.Path, srv.Endpoint)
			id, err := uuid.Parse(pathID)
			if err != nil {
				returnErrorResponse(w, http.StatusNotFound, http.StatusText(http.StatusNotFound))
				return
			}

			// check if UUID is known
			idAuth, exists := srv.Auth[id.String()]
			if !exists {
				returnErrorResponse(w, http.StatusNotFound, http.StatusText(http.StatusNotFound))
				return
			}

			// check authorization
			reqAuth := r.Header.Get("X-Auth-Token")
			if reqAuth != idAuth {
				returnErrorResponse(w, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
				return
			}

			// generate a sorted compact rendering of the json formatted request body before forwarding it to the signer
			var reqDump interface{}
			var compactSortedJson bytes.Buffer

			err = json.Unmarshal(reqBody, &reqDump)
			if err != nil {
				returnErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("error parsing request body: %v", err))
				return
			}
			// json.Marshal sorts the keys
			sortedJson, _ := json.Marshal(reqDump)
			_ = json.Compact(&compactSortedJson, sortedJson)

			srv.ReceiveHandler <- append(id[:], compactSortedJson.Bytes()...)

		} else {
			srv.ReceiveHandler <- reqBody
		}

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
	Auth            map[string]string
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
