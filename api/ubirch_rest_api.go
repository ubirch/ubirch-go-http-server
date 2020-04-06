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

// helper function to get "content-type" from headers
func ContentType(r *http.Request) string {
	return strings.ToLower(r.Header.Get("content-type"))
}

// helper function to get "x-auth-token" from headers
func XAuthToken(r *http.Request) string {
	return r.Header.Get("x-auth-token")
}

// blocks until response is received and forwards it to sender
func forwardResponse(respChan chan HTTPResponse, w http.ResponseWriter) {
	resp := <-respChan
	w.WriteHeader(resp.Code)
	for k, v := range resp.Header {
		w.Header().Set(k, v[0])
	}
	_, _ = w.Write(resp.Content)
}

func handleRequestHash(srv *HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// only accept POST requests
		if r.Method != "POST" {
			http.Error(w, fmt.Sprintf("%s not implemented", r.Method), http.StatusNotImplemented)
			return
		}

		// get UUID from URL path
		id, err := uuid.Parse(strings.TrimPrefix(r.URL.Path, srv.Endpoint+"-hash/"))
		if err != nil {
			http.NotFound(w, r)
			return
		}

		// check authorization
		if XAuthToken(r) != srv.Auth {
			http.Error(w, "invalid \"X-Auth-Token\"", http.StatusUnauthorized)
			return
		}

		// make sure request body is of correct type
		expectedType := "application/octet-stream"
		if ContentType(r) != expectedType {
			http.Error(w, fmt.Sprintf("Wrong content-type. Expected \"%s\"", expectedType), http.StatusBadRequest)
			return
		}

		// read request body
		message, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error reading request body: %v", err), http.StatusBadRequest)
			return
		}

		respChan := make(chan HTTPResponse)
		srv.MessageHandler <- HTTPMessage{ID: id, Msg: message, IsAlreadyHashed: true, Response: respChan}

		// wait for response from ubirch backend to be forwarded
		forwardResponse(respChan, w)
	}
}

func handleRequestData(srv *HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// only accept POST requests
		if r.Method != "POST" {
			http.Error(w, fmt.Sprintf("%s not implemented", r.Method), http.StatusNotImplemented)
			return
		}

		// get UUID from URL path
		id, err := uuid.Parse(strings.TrimPrefix(r.URL.Path, srv.Endpoint+"/"))
		if err != nil {
			http.NotFound(w, r)
			return
		}

		// check authorization
		if XAuthToken(r) != srv.Auth {
			http.Error(w, "invalid \"X-Auth-Token\"", http.StatusUnauthorized)
			return
		}

		// make sure request body is of correct type
		expectedType := "application/json"
		if ContentType(r) != expectedType {
			http.Error(w, fmt.Sprintf("Wrong content-type. Expected \"%s\"", expectedType), http.StatusBadRequest)
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
		message := compactSortedJson.Bytes()

		// create HTTPMessage with individual response channel for each request
		respChan := make(chan HTTPResponse)
		srv.MessageHandler <- HTTPMessage{ID: id, Msg: message, IsAlreadyHashed: false, Response: respChan}

		// wait for response from ubirch backend to be forwarded
		forwardResponse(respChan, w)
	}
}

type HTTPServer struct {
	MessageHandler chan HTTPMessage
	Endpoint       string
	Auth           string
}

type HTTPMessage struct {
	ID              uuid.UUID
	Msg             []byte
	IsAlreadyHashed bool
	Response        chan HTTPResponse
}

type HTTPResponse struct {
	Code    int
	Header  map[string][]string
	Content []byte
}

func (srv *HTTPServer) Serve(ctx context.Context, wg *sync.WaitGroup) {
	s := &http.Server{Addr: ":8080"}
	http.HandleFunc(srv.Endpoint+"/", handleRequestData(srv))
	http.HandleFunc(srv.Endpoint+"-hash/", handleRequestHash(srv))

	go func() {
		<-ctx.Done()
		log.Printf("shutting down http service (%s)", srv.Endpoint)
		_ = s.Shutdown(ctx)
	}()

	go func() {
		defer wg.Done()
		err := s.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Printf("error starting http service: %v", err)
		}
	}()
}
