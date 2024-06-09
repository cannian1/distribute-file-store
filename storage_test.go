package main

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"testing"
)

func TestPathTransformFunc(t *testing.T) {
	key := "momsbestrecipe"
	pathKey := CASPathTransformFunc(key)
	expectedOriginalKey := "9c6a869ebfa9c237594def249adac0b2c4582781"
	expectedPathname := "9c6a8/69ebf/a9c23/7594d/ef249/adac0/b2c45/82781"
	assert.Equal(t, pathKey.PathName, expectedPathname)
	assert.Equal(t, pathKey.Filename, expectedOriginalKey)
}

func TestStoreDeleteKey(t *testing.T) {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
	}

	s := NewStore(opts)

	key := "momsbestrecipe"
	data := []byte("some jpg bytes")
	if err := s.writeStream(key, bytes.NewReader(data)); err != nil {
		t.Error(err)
	}

	if err := s.Delete(key); err != nil {
		t.Error(err)
	}

	if _, err := s.Read(key); err == nil {
		t.Error(err)
	}
}

func TestNewStore(t *testing.T) {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
	}

	s := NewStore(opts)

	key := "momsbestrecipe"
	data := []byte("some jpg bytes")
	if err := s.writeStream(key, bytes.NewReader(data)); err != nil {
		t.Error(err)
	}

	r, err := s.Read(key)
	if err != nil {
		t.Error(err)
	}

	b, _ := io.ReadAll(r)

	fmt.Println(string(b))

	assert.Equal(t, string(b), string(data))
	assert.Equal(t, s.Has(key), true)
}
