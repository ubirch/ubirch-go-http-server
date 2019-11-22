package main

import (
	"context"
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

	os.Exit(0)
}

func main() {

	// create a waitgroup that contains all asynchronous operations
	// a cancellable context is used to stop the operations gracefully
	wg := sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())

	// set up graceful shutdown handling
	signals := make(chan os.Signal, 1)
	go shutdown(signals, &wg, cancel)

	// create a messages channel that parses the http message and creates UPPs
	msgsToSign := make(chan []byte, 100)

	// create a messages channel that hashes messages and fetches the UPP to verify
	msgsToVrfy := make(chan []byte, 100)

	// listen to messages
	server := HTTPServer{signHandler: msgsToSign, verifyHandler: msgsToVrfy}
	err := server.Listen(ctx, &wg)
	if err != nil {
		log.Fatalf("error starting service: %v", err)
	}
	wg.Add(1)

	select {
	case vMsg := <-msgsToVrfy:
		log.Println("msgsToVrfy:")
		log.Println(vMsg)
	case sMsg := <-msgsToSign:
		log.Println("msgsToSign:")
		log.Println(sMsg)
	}
}
