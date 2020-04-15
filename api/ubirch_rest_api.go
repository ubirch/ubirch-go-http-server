package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
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
	_, err := w.Write(resp.Content)
	if err != nil {
		log.Printf("http server encountered error writing response: %s", err)
	}
}

type HTTPServer struct {
	MessageHandler chan HTTPMessage
	AuthTokens     map[string]string
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

func (srv *HTTPServer) handleRequestHash(w http.ResponseWriter, r *http.Request) {
	// get UUID from URL
	id, err := uuid.Parse(chi.URLParam(r, "uuid"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// check if UUID is known
	idAuthToken, exists := srv.AuthTokens[id.String()]
	if !exists {
		http.NotFound(w, r)
		return
	}

	// check authorization
	if XAuthToken(r) != idAuthToken {
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

func (srv *HTTPServer) handleRequestData(w http.ResponseWriter, r *http.Request) {
	// get UUID from URL
	id, err := uuid.Parse(chi.URLParam(r, "uuid"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// check if UUID is known
	idAuthToken, exists := srv.AuthTokens[id.String()]
	if !exists {
		http.NotFound(w, r)
		return
	}

	// check authorization
	if XAuthToken(r) != idAuthToken {
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

func (srv *HTTPServer) Serve(ctx context.Context, wg *sync.WaitGroup) {
	router := chi.NewMux()
	router.Post("/{uuid}", srv.handleRequestData)
	router.Post("/{uuid}/hash", srv.handleRequestHash)

	server := &http.Server{
		Addr:         ":8080",
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  75 * time.Second,
	}

	go func() {
		<-ctx.Done()
		log.Printf("shutting down http server")
		server.SetKeepAlivesEnabled(false) // disallow clients to create new long-running conns
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Failed to gracefully shutdown server: %s", err)
		}
	}()

	go func() {
		defer wg.Done()
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("error starting http service: %v", err)
		}
	}()
}
