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

func handleRequest(srv *HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// only accept POST requests
		if r.Method != "POST" {
			http.Error(w, fmt.Sprintf("%s not implemented", r.Method), http.StatusNotImplemented)
			return
		}

		// make sure request body is of type json
		if strings.ToLower(r.Header.Get("Content-Type")) != "application/json" {
			http.Error(w, "Wrong request body type", http.StatusBadRequest)
			return
		}

		// get UUID from URL path
		id, err := uuid.Parse(strings.TrimPrefix(r.URL.Path, srv.Endpoint))
		if err != nil {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		// check if UUID is known
		idAuth, exists := srv.Auth[id.String()]
		if !exists {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		// check authorization
		reqAuth := r.Header.Get("X-Auth-Token")
		if reqAuth != idAuth {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		// read request body
		reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error reading request body: %v", err), http.StatusBadRequest)
			return
		}

		// generate a sorted compact rendering of the json formatted request body
		var reqDump interface{}
		var compactSortedJson bytes.Buffer

		err = json.Unmarshal(reqBody, &reqDump)
		if err != nil {
			http.Error(w, fmt.Sprintf("error parsing request body: %v", err), http.StatusBadRequest)
			return
		}

		// json.Marshal sorts the keys
		sortedJson, _ := json.Marshal(reqDump)
		_ = json.Compact(&compactSortedJson, sortedJson)

		respChan := make(chan HTTPResponse)
		srv.MessageHandler <- HTTPMessage{ID: id, Msg: compactSortedJson.Bytes(), Response: respChan}

		// wait for response from ubirch backend to be forwarded
		resp := <-respChan
		w.WriteHeader(resp.Code)
		for k, v := range resp.Header {
			w.Header().Set(k, v[0])
		}
		w.Write(resp.Content)
	}
}

type HTTPServer struct {
	MessageHandler chan HTTPMessage
	Endpoint       string
	Auth           map[string]string
}

type HTTPMessage struct {
	ID       uuid.UUID
	Msg      []byte
	Response chan HTTPResponse
}

type HTTPResponse struct {
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
