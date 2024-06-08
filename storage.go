package main

import (
	"io"
	"log"
	"os"
)

// PathTransformFunc 用于将一个key转换为一个路径
type PathTransformFunc func(string) string

// StoreOpts 保存了一个 Store 的配置
type StoreOpts struct {
	PathTransformFunc PathTransformFunc
}

// DefaultPathTransformFunc 是一个默认的 PathTransformFunc
var DefaultPathTransformFunc = func(key string) string {
	return key
}

type Store struct {
	StoreOpts
}

func NewStore(opt StoreOpts) *Store {
	return &Store{
		StoreOpts: opt,
	}
}

// writeStream 将一个流写入到磁盘上
func (s *Store) writeStream(key string, r io.Reader) error {
	pathName := key

	if err := os.MkdirAll(pathName, os.ModePerm); err != nil {
		return err
	}

	fileName := "some-file-name"
	pathAndFileName := pathName + "/" + fileName

	f, err := os.Create(pathAndFileName)
	if err != nil {
		return err
	}

	n, err := io.Copy(f, r)
	if err != nil {
		return err
	}
	log.Printf("written (%d bytes) to dist:%s\n", n, pathAndFileName)

	return nil
}
