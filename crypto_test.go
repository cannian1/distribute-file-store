package main

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCopyEncryptDecrypt(t *testing.T) {
	payload := "Foo not bar"
	src := bytes.NewReader([]byte(payload))
	dst := new(bytes.Buffer)
	key := newEncryptionKey()
	_, err := copyEncrypt(key, src, dst)
	assert.Nil(t, err)

	fmt.Println(len(payload))
	fmt.Println(len(dst.String()))

	out := new(bytes.Buffer)
	nw, err := copyDecrypt(key, dst, out)
	assert.Nil(t, err)
	assert.Equal(t, nw, 16+len(payload))
	assert.Equal(t, out.String(), payload)
}

func TestGId(t *testing.T) {
	id := generateID()
	fmt.Println(id)
}
