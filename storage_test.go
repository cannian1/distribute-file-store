package main

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPathTransformFunc(t *testing.T) {
	key := "momsbestrecipe"
	pathName := CASPathTransformFunc(key)
	expectedPathname := "9c6a8/69ebf/a9c23/7594d/ef249/adac0/b2c45/82781"
	assert.Equal(t, pathName, expectedPathname)
}

func TestNewStore(t *testing.T) {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
	}

	s := NewStore(opts)

	data := bytes.NewReader([]byte("some jpg bytes"))
	if err := s.writeStream("myspecialpicture", data); err != nil {
		t.Error(err)
	}

}
