package main

import (
	"distributed-file-store/p2p"
	"fmt"
	"log"
	"time"
)

func OnPeer(peer p2p.Peer) error {
	fmt.Println("doing some logic with peer outside TCPTransport")
	return nil
	//peer.Close()
	//return nil
}

func main() {
	tcpTransportOpts := p2p.TCPTransportOpts{
		ListenAddr:    ":3000",
		HandshakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
		// todo: OnPeer func
	}
	tcpTransport := p2p.NewTCPTransport(tcpTransportOpts)

	fileServerOpts := FileServerOpts{
		StorageRoot:       "3000_network",
		PathTransformFunc: CASPathTransformFunc,
		Transport:         tcpTransport,
	}

	s := NewFileServer(fileServerOpts)

	go func() {
		time.Sleep(3 * time.Second)
		s.Stop()
	}()

	if err := s.Start(); err != nil {
		log.Fatalln(err)
	}

}
