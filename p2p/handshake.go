package p2p

// HandshakeFunc 是一个握手函数
type HandshakeFunc func(peer Peer) error

func NOPHandshakeFunc(peer Peer) error {
	return nil
}
