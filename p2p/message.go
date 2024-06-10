package p2p

const (
	IncomingMessage = 0x1
	IncomingStream  = 0x2
)

// RPC 保存网络中两个节点间正在传输的任何消息
type RPC struct {
	From    string
	Payload []byte
	Stream  bool
}
