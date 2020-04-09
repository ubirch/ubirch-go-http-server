package main

import (
	"context"
	"github.com/ubirch/ubirch-go-http-server/api"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// handle graceful shutdown
func shutdown(signals chan os.Signal, wg *sync.WaitGroup, cancel context.CancelFunc) {
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	// block until we receive a SIGINT or SIGTERM
	sig := <-signals
	log.Printf("shutting down after receiving: %v", sig)

	// wait for all go routings to end, cancels the go routines contexts
	// and waits for the wait group
	cancel()
	wg.Wait()

	log.Println("clean exit")
	os.Exit(0)
}

func main() {
	whitelist := map[string]string{"825255ef-a9cf-42e9-8839-ada9a81f99cd": "1234567890_password"}

	// create a waitgroup that contains all asynchronous operations
	// a cancellable context is used to stop the operations gracefully
	wg := sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())

	// set up graceful shutdown handling
	signals := make(chan os.Signal, 1)
	go shutdown(signals, &wg, cancel)

	// create a messages channel that parses the http message and creates UPPs
	msgsToSign := make(chan api.HTTPMessage, 100)

	// listen to messages
	httpSrvSign := api.HTTPServer{MessageHandler: msgsToSign, AuthTokens: whitelist}
	httpSrvSign.Serve(ctx, &wg)
	wg.Add(1)

	for {
		sMsg := <-msgsToSign
		log.Println("received message:")
		log.Println("UUID: " + sMsg.ID.String())
		log.Println("Message: " + string(sMsg.Msg))
	}
}
