package p2p

// Peer 是一个代表远端节点的接口
type Peer interface {
	Close() error
}

// Transport 是处理任何远端网络之间节点通信的接口
// 可以是 TCP, UDP, Websocket 等
type Transport interface {
	ListenAndAccept() error
	Consume() <-chan RPC
	Close() error
}
