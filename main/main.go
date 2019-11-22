package main

import "log"

func main() {
	// create a messages channel that parses the http message and creates UPPs
	msgsToSign := make(chan []byte, 100)

	// listen to messages to ubirch (sign)
	httpSrvSign := HTTPServer{signHandler: msgsToSign, verifyHandler: nil}
	err := httpSrvSign.Listen()
	if err != nil {
		log.Fatalf("error starting signing service: %v", err)
	}

	// create a messages channel that hashes messages and fetches the UPP to verify
	msgsToVrfy := make(chan []byte, 100)

	// listen to messages to verify
	httpSrvVrfy := HTTPServer{verifyHandler: msgsToVrfy, signHandler: nil}
	err = httpSrvVrfy.Listen()
	if err != nil {
		log.Fatalf("error starting verification service: %v", err)
	}

	select {
	case vMsg := <-msgsToVrfy:
		log.Println("msgsToVrfy:")
		log.Println(vMsg)
	case sMsg := <-msgsToSign:
		log.Println("msgsToSign:")
		log.Println(sMsg)
	}
}
