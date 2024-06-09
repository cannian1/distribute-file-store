package main

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"io"
	"log"
	"os"
	"strings"
)

// CASPathTransformFunc 是将一个 key 通过散列转化为一个路径的函数
func CASPathTransformFunc(key string) string {
	hash := sha1.Sum([]byte(key)) // [20]byte -> []byte
	hashStr := hex.EncodeToString(hash[:])

	blockSize := 5 // 转化后的路径名每层最大长度
	sliceLen := len(hashStr) / 5

	paths := make([]string, sliceLen)

	for i := range sliceLen {
		from, to := i*blockSize, (i+1)*blockSize
		paths[i] = hashStr[from:to]
	}
	return strings.Join(paths, "/")
}

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
	pathName := s.PathTransformFunc(key) // 通过传入的规则函数将 key 转化为路径

	if err := os.MkdirAll(pathName, os.ModePerm); err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	io.Copy(buf, r)

	fileNameBytes := md5.Sum(buf.Bytes())
	fileName := hex.EncodeToString(fileNameBytes[:])
	pathAndFileName := pathName + "/" + fileName

	f, err := os.Create(pathAndFileName)
	defer f.Close()
	if err != nil {
		return err
	}

	n, err := io.Copy(f, buf)
	if err != nil {
		return err
	}
	log.Printf("written (%d bytes) to dist:%s\n", n, pathAndFileName)

	return nil
}
