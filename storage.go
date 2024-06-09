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

const defaultRootFoldName = "cannian1"

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
	// Root 是存储文件的根目录，包含系统所有的文件夹/文件
	Root              string
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

func NewStore(opts StoreOpts) *Store {
	if opts.PathTransformFunc == nil {
		opts.PathTransformFunc = DefaultPathTransformFunc
	}

	if len(opts.Root) == 0 {
		opts.Root = defaultRootFoldName
	}

	return &Store{
		StoreOpts: opts,
	}
}

func (s *Store) Has(key string) bool {
	pathKey := s.PathTransformFunc(key)
	fullPathWithRoot := fmt.Sprintf("%s/%s", s.Root, pathKey.FullPath())

	_, err := os.Stat(fullPathWithRoot)
	return !errors.Is(err, os.ErrNotExist)
}

// Delete 从磁盘上删除一个 key 对应的文件
func (s *Store) Delete(key string) error {
	pathKey := s.PathTransformFunc(key)

	defer func() {
		log.Printf("deleted [%s] from disk\n", pathKey.FullPath())
	}()

	firstPathNameWithRoot := fmt.Sprintf("%s/%s", s.Root, pathKey.FirstPathName())

	// todo: 这种删除方式可能导致其他通过散列函数生成文件名前面的pad与想要删除的文件相同的文件被删除
	return os.RemoveAll(firstPathNameWithRoot)
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
	pathPathWithRoot := fmt.Sprintf("%s/%s", s.Root, pathKey.FullPath())
	return os.Open(pathPathWithRoot)
}

// writeStream 将一个流写入到磁盘上
func (s *Store) writeStream(key string, r io.Reader) error {
	pathKey := s.PathTransformFunc(key) // 通过传入的规则函数将 key 转化为路径
	pathNameWithRoot := fmt.Sprintf("%s/%s", s.Root, pathKey.PathName)

	if err := os.MkdirAll(pathNameWithRoot, os.ModePerm); err != nil {
		return err
	}

	fullPathWithRoot := fmt.Sprintf("%s/%s", s.Root, pathKey.FullPath())

	f, err := os.Create(fullPathWithRoot)
	defer f.Close()
	if err != nil {
		return err
	}

	n, err := io.Copy(f, r)
	if err != nil {
		return err
	}
	log.Printf("written (%d bytes) to dist:%s\n", n, fullPathWithRoot)

	return nil
}
