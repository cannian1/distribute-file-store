package p2p

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync"
)

// TCPPeer 代表一个 TCP 连接的远端节点
type TCPPeer struct {
	net.Conn // 对端的连接

	// 如果是 true, 则是发起连接的一方
	// 如果是 false, 则是接收连接的一方
	outbound bool

	wg *sync.WaitGroup
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		Conn:     conn,
		outbound: outbound,
		wg:       &sync.WaitGroup{},
	}
}

// CloseStream 实现 TCPPeer 接口，关闭流
func (p *TCPPeer) CloseStream() {
	p.wg.Done()
}

// Send 实现 TCPPeer 接口，发送数据
func (p *TCPPeer) Send(b []byte) error {
	_, err := p.Conn.Write(b)
	return err
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
		rpcChan:          make(chan RPC, 1024),
	}
}

// Addr 实现 Transport 的接口，返回监听地址
func (t *TCPTransport) Addr() string {
	return t.ListenAddr
}

// Consume 实现 Transport 的接口，返回一个只读 channel 用于接收即将到来的网络上的对端的消息
func (t *TCPTransport) Consume() <-chan RPC {
	return t.rpcChan
}

// Close 实现 Transport 的接口，关闭监听
func (t *TCPTransport) Close() error {
	return t.listener.Close()
}

// Dial 实现 Transport 的接口，发起连接
func (t *TCPTransport) Dial(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}

	go t.handleConn(conn, true)
	return nil
}

// ListenAndAccept 实现 Transport 的接口，监听并接受连接
func (t *TCPTransport) ListenAndAccept() (err error) {
	t.listener, err = net.Listen("tcp", t.ListenAddr)
	if err != nil {
		return err
	}

	go t.startAcceptLoop()

	slog.Info("TCP transport listening on port", "port", t.ListenAddr)

	return
}

func (t *TCPTransport) startAcceptLoop() {
	for {
		conn, err := t.listener.Accept()
		if errors.Is(err, net.ErrClosed) {
			return
		}
		if err != nil {
			slog.Error("TCPTransport accept error", "error", err)
			return
		}

		go t.handleConn(conn, false)
	}
}

func (t *TCPTransport) handleConn(conn net.Conn, outbound bool) {
	var err error

	defer func() {
		slog.Debug("dropping peer connection", "error", err)
		conn.Close()
	}()

	peer := NewTCPPeer(conn, outbound)

	if err = t.HandshakeFunc(peer); err != nil {
		return
	}

	if t.OnPeer != nil {
		if err = t.OnPeer(peer); err != nil {
			return
		}
	}

	// 循环读取数据
	for {
		rpc := RPC{}
		err = t.Decoder.Decode(peer.Conn, &rpc)
		if err != nil {
			slog.Error("TCPTransport read error", "error", err)
			return
		}

		rpc.From = conn.RemoteAddr().String()
		if rpc.Stream {
			peer.wg.Add(1)
			fmt.Printf("[%s] incoming stream, waiting...\n", conn.RemoteAddr())
			peer.wg.Wait()
			fmt.Printf("[%s] stream closed, resuming read loop\n", conn.RemoteAddr())
			continue
		}

		t.rpcChan <- rpc
		slog.Debug("TCPTransport received message", "rpc", rpc)
	}
}
