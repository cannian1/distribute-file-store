package main

import (
	"distributed-file-store/p2p"
	"fmt"
	"log"
)

type FileServerOpts struct {
	StorageRoot       string
	PathTransformFunc PathTransformFunc
	Transport         p2p.Transport
}

// FileServer 是一个简单的文件服务器，它可以接收来自网络上的对端的文件请求
type FileServer struct {
	FileServerOpts
	store  *Store
	quitCh chan struct{}
}

// NewFileServer 创建一个新的文件服务器
func NewFileServer(opts FileServerOpts) *FileServer {
	storeOpts := StoreOpts{
		Root:              opts.StorageRoot,
		PathTransformFunc: opts.PathTransformFunc,
	}

	return &FileServer{
		FileServerOpts: opts,
		store:          NewStore(storeOpts),
		quitCh:         make(chan struct{}),
	}
}

// Start 启动文件服务器
func (s *FileServer) Start() error {
	if err := s.Transport.ListenAndAccept(); err != nil {
		return err
	}

	s.loop()
	return nil
}

// Stop 停止文件服务器
func (s *FileServer) Stop() {
	close(s.quitCh)
}

func (s *FileServer) loop() {
	// 可以在循环外面做 defer 操作或者别的逻辑
	defer func() {
		log.Println("file server stopped due to user quit action")
		s.Transport.Close()
	}()

	for {
		select {
		case msg := <-s.Transport.Consume():
			fmt.Println(msg)
		case <-s.quitCh:
			return
		}
	}
}
