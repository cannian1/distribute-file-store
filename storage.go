package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

// CASPathTransformFunc 是将一个 key 通过散列转化为一个路径的函数
func CASPathTransformFunc(key string) PathKey {
	hash := sha1.Sum([]byte(key)) // [20]byte -> []byte
	hashStr := hex.EncodeToString(hash[:])

	blockSize := 5 // 转化后的路径名每层最大长度
	sliceLen := len(hashStr) / 5

	paths := make([]string, sliceLen)

	for i := range sliceLen {
		from, to := i*blockSize, (i+1)*blockSize
		paths[i] = hashStr[from:to]
	}
	return PathKey{
		PathName: strings.Join(paths, "/"),
		Filename: hashStr,
	}
}

// PathTransformFunc 用于将一个key转换为一个路径
type PathTransformFunc func(string) PathKey

type PathKey struct {
	PathName string
	Filename string
}

func (p PathKey) FirstPathName() string {
	paths := strings.Split(p.PathName, "/")
	if len(paths) == 0 {
		return ""
	}
	return paths[0]
}

func (p PathKey) FullPath() string {
	return fmt.Sprintf("%s/%s", p.PathName, p.Filename)
}

// StoreOpts 保存了一个 Store 的配置
type StoreOpts struct {
	PathTransformFunc PathTransformFunc
}

// DefaultPathTransformFunc 是一个默认的 PathTransformFunc
var DefaultPathTransformFunc = func(key string) PathKey {
	return PathKey{
		PathName: key,
		Filename: key,
	}
}

type Store struct {
	StoreOpts
}

func NewStore(opt StoreOpts) *Store {
	if opt.PathTransformFunc == nil {
		opt.PathTransformFunc = DefaultPathTransformFunc
	}
	return &Store{
		StoreOpts: opt,
	}
}

func (s *Store) Has(key string) bool {
	pathKey := s.PathTransformFunc(key)
	_, err := os.Stat(pathKey.FullPath())
	return !errors.Is(err, os.ErrNotExist)
}

// Delete 从磁盘上删除一个 key 对应的文件
func (s *Store) Delete(key string) error {
	pathKey := s.PathTransformFunc(key)

	defer func() {
		log.Printf("deleted [%s] from disk\n", pathKey.FullPath())
	}()

	// todo: 这种删除方式可能导致其他通过散列函数生成文件名前面的pad与想要删除的文件相同的文件被删除
	return os.RemoveAll(pathKey.FirstPathName())
}

func (s *Store) Read(key string) (io.Reader, error) {
	f, err := s.readStream(key)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, f)

	return buf, nil
}

func (s *Store) readStream(key string) (io.ReadCloser, error) {
	pathKey := s.PathTransformFunc(key)
	return os.Open(pathKey.FullPath())
}

// writeStream 将一个流写入到磁盘上
func (s *Store) writeStream(key string, r io.Reader) error {
	pathKey := s.PathTransformFunc(key) // 通过传入的规则函数将 key 转化为路径

	if err := os.MkdirAll(pathKey.PathName, os.ModePerm); err != nil {
		return err
	}

	fullPath := pathKey.FullPath()
	fmt.Println(fullPath)

	f, err := os.Create(fullPath)
	defer f.Close()
	if err != nil {
		return err
	}

	n, err := io.Copy(f, r)
	if err != nil {
		return err
	}
	log.Printf("written (%d bytes) to dist:%s\n", n, fullPath)

	return nil
}
