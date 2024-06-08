package p2p

import "net"

// RPC 保存网络中两个节点间正在传输的任何消息
type RPC struct {
	From    net.Addr
	Payload []byte
}
