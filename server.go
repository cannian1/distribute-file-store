package main

import (
	"bytes"
	"distributed-file-store/p2p"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"log/slog"
	"sync"
	"time"
)

func init() {
	gob.Register(MessageStoreFile{})
	gob.Register(MessageGetFile{})
}

type FileServerOpts struct {
	ID                string
	EncKey            []byte
	StorageRoot       string
	PathTransformFunc PathTransformFunc
	Transport         p2p.Transport
	BootstrapNodes    []string
}

// FileServer 是一个简单的文件服务器，它可以接收来自网络上的对端的文件请求
type FileServer struct {
	FileServerOpts

	peerLock sync.Mutex
	peers    map[string]p2p.Peer

	store  *Store
	quitCh chan struct{}
}

// NewFileServer 创建一个新的文件服务器
func NewFileServer(opts FileServerOpts) *FileServer {
	storeOpts := StoreOpts{
		Root:              opts.StorageRoot,
		PathTransformFunc: opts.PathTransformFunc,
	}

	if len(opts.ID) == 0 {
		opts.ID = generateID()
	}

	return &FileServer{
		FileServerOpts: opts,
		store:          NewStore(storeOpts),
		quitCh:         make(chan struct{}),
		peers:          make(map[string]p2p.Peer),
	}
}

// Start 启动文件服务器
func (s *FileServer) Start() error {
	fmt.Printf("[%s] starting fileserver...\n", s.Transport.Addr())
	if err := s.Transport.ListenAndAccept(); err != nil {
		return err
	}

	s.bootstrapNetwork()
	s.loop()
	return nil
}

// broadcast 广播消息给所有的对端
func (s *FileServer) broadcast(msg *Message) error {
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		return err
	}

	for _, peer := range s.peers {
		peer.Send([]byte{p2p.IncomingMessage})
		if err := peer.Send(buf.Bytes()); err != nil {
			return err
		}
	}

	return nil
}

type Message struct {
	Payload any
}

type MessageStoreFile struct {
	ID   string
	Key  string
	Size int64
}

type MessageGetFile struct {
	ID  string
	Key string
}

// Get 从本地磁盘获取文件
func (s *FileServer) Get(key string) (io.Reader, error) {
	if s.store.Has(s.ID, key) {
		fmt.Printf("[%s] serving file (%s) from local disk\n", s.Transport.Addr(), key)
		_, r, err := s.store.Read(s.ID, key)
		return r, err
	}

	fmt.Printf("[%s] dont have file (%s) locally, fetching from network...\n", s.Transport.Addr(), key)

	msg := Message{
		Payload: MessageGetFile{
			ID:  s.ID,
			Key: hashKey(key),
		},
	}

	if err := s.broadcast(&msg); err != nil {
		return nil, err
	}

	time.Sleep(time.Millisecond * 500)

	for _, peer := range s.peers {
		// 首先读取文件大小，这样就可以限制从连接读取的字节数，这样它就不会一直挂起
		var fileSize int64
		binary.Read(peer, binary.LittleEndian, &fileSize)

		n, err := s.store.WriteDecrypt(s.EncKey, s.ID, key, io.LimitReader(peer, fileSize))
		if err != nil {
			return nil, err
		}

		fmt.Printf("[%s] received (%d) bytes over the network from (%s)", s.Transport.Addr(), n, peer.RemoteAddr())

		peer.CloseStream()
	}

	_, r, err := s.store.Read(s.ID, key)
	return r, err
}

// Store 存储文件
func (s *FileServer) Store(key string, r io.Reader) error {
	// 1.将文件存到磁盘
	// 2.将文件广播给其他节点

	fileBuffer := new(bytes.Buffer)
	tee := io.TeeReader(r, fileBuffer)

	size, err := s.store.Write(s.ID, key, tee)
	if err != nil {
		return err
	}

	log.Printf("written (%d bytes) to dist\n", size)

	msg := Message{
		Payload: MessageStoreFile{
			ID:   s.ID,
			Key:  hashKey(key),
			Size: size + 16,
		},
	}

	if err := s.broadcast(&msg); err != nil {
		return err
	}

	time.Sleep(time.Millisecond * 5)

	peers := make([]io.Writer, 0, len(s.peers))
	for _, peer := range s.peers {
		peers = append(peers, peer)
	}
	mw := io.MultiWriter(peers...) // 将数据写入多个 writer
	mw.Write([]byte{p2p.IncomingStream})
	n, err := copyEncrypt(s.EncKey, fileBuffer, mw)
	if err != nil {
		return err
	}

	fmt.Printf("[%s] received and written (%d) bytes to disk\n", s.Transport.Addr(), n)
	return nil
}

// Stop 停止文件服务器
func (s *FileServer) Stop() {
	close(s.quitCh)
}

// OnPeer 是一个回调函数，当有新的对端连接时会被调用
func (s *FileServer) OnPeer(p p2p.Peer) error {
	s.peerLock.Lock()
	defer s.peerLock.Unlock()

	s.peers[p.RemoteAddr().String()] = p

	slog.Info("connected with remote", "remote addr", p.RemoteAddr())

	return nil
}

// loop 是一个无限循环，用于处理来自网络上的对端的消息
func (s *FileServer) loop() {
	// 可以在循环外面做 defer 操作或者别的逻辑
	defer func() {
		slog.Info("file server stopped due to user quit action")
		s.Transport.Close()
	}()

	for {
		select {
		case rpc := <-s.Transport.Consume():

			var msg Message
			if err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&msg); err != nil {
				slog.Error("failed to decode message", "error", err)
			}
			if err := s.handleMessage(rpc.From, &msg); err != nil {
				slog.Error("handle message error: ", "error", err)
			}
		case <-s.quitCh:
			return
		}
	}
}

func (s *FileServer) handleMessage(from string, msg *Message) error {
	switch v := msg.Payload.(type) {
	case MessageStoreFile:
		return s.handleMessageStoreFile(from, v)
	case MessageGetFile:
		return s.handleMessageGetFile(from, v)
	}

	return nil
}

func (s *FileServer) handleMessageGetFile(from string, msg MessageGetFile) error {
	if !s.store.Has(msg.ID, msg.Key) {
		return fmt.Errorf("[%s] need to serve file (%s) but it does not exist on disk", s.Transport.Addr(), msg.Key)
	}

	fmt.Printf("[%s] serving file (%s) over the network\n", s.Transport.Addr(), msg.Key)

	fileSize, r, err := s.store.Read(msg.ID, msg.Key)
	if err != nil {
		return err
	}

	if rc, ok := r.(io.ReadCloser); ok {
		fmt.Println("closing readCloser")
		defer rc.Close()
	}

	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("peer %s not in map", from)
	}

	peer.Send([]byte{p2p.IncomingStream})
	binary.Write(peer, binary.LittleEndian, fileSize)
	n, err := io.Copy(peer, r)
	if err != nil {
		return err
	}

	fmt.Printf("[%s] written (%d) bytes over the network to %s\n", s.Transport.Addr(), n, from)

	return nil
}

func (s *FileServer) handleMessageStoreFile(from string, msg MessageStoreFile) error {
	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("peer (%s) could not be found in the peer list", from)
	}

	n, err := s.store.Write(msg.ID, msg.Key, io.LimitReader(peer, msg.Size))
	if err != nil {
		return err
	}

	fmt.Printf("[%s] written %d bytes to disk\n", s.Transport.Addr(), n)

	peer.CloseStream()

	return nil
}

// bootstrapNetwork 启动网络
func (s *FileServer) bootstrapNetwork() error {
	for _, addr := range s.BootstrapNodes {
		if len(addr) == 0 {
			continue
		}

		go func(addr string) {
			slog.Info("attempting connect with remote", "addr", addr)
			if err := s.Transport.Dial(addr); err != nil {
				slog.Info("failed to dial", "err", err)
			}
		}(addr)
	}
	return nil
}
