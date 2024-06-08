package main

import (
	"distributed-file-store/p2p"
	"fmt"
	"log"
)

func OnPeer(peer p2p.Peer) error {
	fmt.Println("doing some logic with peer outside TCPTransport")
	return nil
	//peer.Close()
	//return nil
}

func main() {
	tcpOps := p2p.TCPTransportOpts{
		ListenAddr:    ":3000",
		HandshakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
		OnPeer:        OnPeer,
	}
	tr := p2p.NewTCPTransport(tcpOps)
	if err := tr.ListenAndAccept(); err != nil {
		log.Fatal(err)
	}

	go func() {
		for rpc := range tr.Consume() {
			log.Printf("Received RPC: %+v\n", rpc)
		}
	}()

	ch := make(chan struct{})
	<-ch
}
