package main

import (
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

func (s *Store) Has(id string, key string) bool {
	pathKey := s.PathTransformFunc(key)
	fullPathWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, pathKey.FullPath())

	_, err := os.Stat(fullPathWithRoot)
	return !errors.Is(err, os.ErrNotExist)
}

func (s *Store) Clear() error {
	return os.RemoveAll(s.Root)
}

// Delete 从磁盘上删除一个 key 对应的文件
func (s *Store) Delete(id string, key string) error {
	pathKey := s.PathTransformFunc(key)

	defer func() {
		log.Printf("deleted [%s] from disk\n", pathKey.FullPath())
	}()

	firstPathNameWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, pathKey.FirstPathName())

	// todo: 这种删除方式可能导致其他通过散列函数生成文件名前面的pad与想要删除的文件相同的文件被删除
	return os.RemoveAll(firstPathNameWithRoot)
}

func (s *Store) Write(id string, key string, r io.Reader) (int64, error) {
	return s.writeStream(id, key, r)
}

func (s *Store) WriteDecrypt(encKey []byte, id string, key string, r io.Reader) (int64, error) {
	f, err := s.openFileForWriting(id, key)
	if err != nil {
		return 0, err
	}
	n, err := copyDecrypt(encKey, r, f)
	return int64(n), err
}

func (s *Store) Read(id string, key string) (int64, io.Reader, error) {
	return s.readStream(id, key)
}

func (s *Store) readStream(id string, key string) (int64, io.ReadCloser, error) {
	pathKey := s.PathTransformFunc(key)
	fullPathWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, pathKey.FullPath())

	file, err := os.Open(fullPathWithRoot)
	if err != nil {
		return 0, nil, err
	}

	fi, err := file.Stat()
	if err != nil {
		return 0, nil, err
	}

	return fi.Size(), file, nil
}

func (s *Store) openFileForWriting(id string, key string) (*os.File, error) {
	pathKey := s.PathTransformFunc(key) // 通过传入的规则函数将 key 转化为路径
	pathNameWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, pathKey.PathName)
	if err := os.MkdirAll(pathNameWithRoot, os.ModePerm); err != nil {
		return nil, err
	}

	fullPathWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, pathKey.FullPath())

	return os.Create(fullPathWithRoot)
}

// writeStream 将一个流写入到磁盘上
func (s *Store) writeStream(id string, key string, r io.Reader) (int64, error) {
	f, err := s.openFileForWriting(id, key)
	if err != nil {
		return 0, err
	}
	return io.Copy(f, r)
}
