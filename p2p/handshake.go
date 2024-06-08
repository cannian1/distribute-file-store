package p2p

// HandshakeFunc 是一个握手函数
type HandshakeFunc func(peer Peer) error

func NOPHandshakeFun(peer Peer) error {
	return nil
}
