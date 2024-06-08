package main

import (
	"distributed-file-store/p2p"
	"log"
)

func main() {
	tr := p2p.NewTCPTransport(":3000")
	if err := tr.ListenAndAccept(); err != nil {
		log.Fatal(err)
	}

	ch := make(chan struct{})
	<-ch
}
