package p2p

import (
	"fmt"
	"net"
	"sync"
)

// TCPPeer 代表一个 TCP 连接的远端节点
type TCPPeer struct {
	conn net.Conn // 对端的连接

	// 如果是 true, 则是发起连接的一方
	// 如果是 false, 则是接收连接的一方
	outbound bool
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		conn:     conn,
		outbound: outbound,
	}
}

type TCPTransport struct {
	listenAddress string
	listener      net.Listener
	handshakeFunc HandshakeFunc
	decoder       Decoder

	mu    sync.RWMutex // 把互斥锁放置在需要保护的字段上
	peers map[net.Addr]Peer
}

// NewTCPTransport 创建一个新的 TCPTransport
func NewTCPTransport(listenAddress string) *TCPTransport {
	return &TCPTransport{
		handshakeFunc: NOPHandshakeFun,
		listenAddress: listenAddress,
		peers:         make(map[net.Addr]Peer),
	}
}

func (t *TCPTransport) ListenAndAccept() (err error) {
	t.listener, err = net.Listen("tcp", t.listenAddress)
	if err != nil {
		return err
	}

	go t.startAcceptLoop()

	return
}

func (t *TCPTransport) startAcceptLoop() {
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			fmt.Println("TCPTransport accept error: ", err)
			return
		}

		fmt.Printf("new incoming connection: %+v\n, peer addr is %s", conn)
		go t.handleConn(conn)
	}
}

type Temp struct {
}

func (t *TCPTransport) handleConn(conn net.Conn) {
	peer := NewTCPPeer(conn, true)

	if err := t.handshakeFunc(peer); err != nil {
		fmt.Println("TCPTransport handshake error: ", err)
		return
	}

	// 循环读取数据
	msg := &Temp{}
	for {
		if err := t.decoder.Decode(peer.conn, msg); err != nil {
			fmt.Println("TCPTransport decode error: ", err)
			continue
		}
	}
}
