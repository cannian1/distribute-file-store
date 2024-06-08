package p2p

import (
	"fmt"
	"net"
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

// Close 实现 TCPPeer 接口，关闭连接
func (p *TCPPeer) Close() error {
	return p.conn.Close()
}

type TCPTransportOpts struct {
	ListenAddr    string
	HandshakeFunc HandshakeFunc
	Decoder       Decoder
	OnPeer        func(Peer) error
}

type TCPTransport struct {
	TCPTransportOpts
	listener net.Listener
	rpcChan  chan RPC
}

// NewTCPTransport 创建一个新的 TCPTransport
func NewTCPTransport(Ops TCPTransportOpts) *TCPTransport {
	return &TCPTransport{
		TCPTransportOpts: Ops,
		rpcChan:          make(chan RPC),
	}
}

// Consume 实现 Transport 的接口，返回一个只读 channel 用于接收即将到来的网络上的对端的消息
func (t *TCPTransport) Consume() <-chan RPC {
	return t.rpcChan
}

func (t *TCPTransport) ListenAndAccept() (err error) {
	t.listener, err = net.Listen("tcp", t.ListenAddr)
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

		fmt.Printf("new incoming connection: %+v\n", conn)
		go t.handleConn(conn)
	}
}

func (t *TCPTransport) handleConn(conn net.Conn) {
	var err error

	defer func() {
		fmt.Printf("dropping peer connection :%v\n", err)
		conn.Close()
	}()

	peer := NewTCPPeer(conn, true)

	if err = t.HandshakeFunc(peer); err != nil {
		return
	}

	if t.OnPeer != nil {
		if err = t.OnPeer(peer); err != nil {
			return
		}
	}

	// 循环读取数据
	rpc := RPC{}
	//buf := make([]byte, 1024)
	for {
		err = t.Decoder.Decode(peer.conn, &rpc)

		if err != nil {
			fmt.Println("TCPTransport read error: ", err)
			return
		}

		rpc.From = conn.RemoteAddr()
		t.rpcChan <- rpc
		fmt.Printf("receive message: %+v\n", rpc)
	}
}
