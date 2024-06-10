package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"io"
)

// generateID 生成一个随机的 ID
func generateID() string {
	buf := make([]byte, 32)
	io.ReadFull(rand.Reader, buf)
	return hex.EncodeToString(buf)
}

// hashKey 生成一个 key 的 hash
func hashKey(key string) string {
	hash := md5.Sum([]byte(key))
	return hex.EncodeToString(hash[:])
}

// newEncryptionKey 生成一个新的加密 key
func newEncryptionKey() []byte {
	keyBuf := make([]byte, 32)
	io.ReadFull(rand.Reader, keyBuf)
	return keyBuf
}

// copyStream 从 src 中读取数据，加密后写入到 dst 中
func copyStream(stream cipher.Stream, blockSize int, src io.Reader, dst io.Writer) (int, error) {
	var (
		buf = make([]byte, 32*1024)
		nw  = blockSize
	)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			stream.XORKeyStream(buf, buf[:n])
			nn, err := dst.Write(buf[:n])
			if err != nil {
				return 0, err
			}
			nw += nn
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
	}
	return nw, nil
}

// copyDecrypt 从 src 中读取数据，解密后写入到 dst 中
func copyDecrypt(key []byte, src io.Reader, dst io.Writer) (int, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return 0, err
	}

	// 从给定的 io.Reader 读取 IV，在示例中，它应该是读取的块.BlockSize()字节
	iv := make([]byte, block.BlockSize())
	if _, err := src.Read(iv); err != nil {
		return 0, err
	}

	stream := cipher.NewCTR(block, iv)
	return copyStream(stream, block.BlockSize(), src, dst)
}

func copyEncrypt(key []byte, src io.Reader, dst io.Writer) (int, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return 0, err
	}

	iv := make([]byte, block.BlockSize()) // 16 bytes
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return 0, err
	}

	// 在 dst 中写入 IV，这样我们就可以在解密时读取它
	if _, err := dst.Write(iv); err != nil {
		return 0, err
	}

	stream := cipher.NewCTR(block, iv)
	return copyStream(stream, block.BlockSize(), src, dst)
}
