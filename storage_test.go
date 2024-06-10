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
	expectedFilename := "9c6a869ebfa9c237594def249adac0b2c4582781"
	expectedPathname := "9c6a8/69ebf/a9c23/7594d/ef249/adac0/b2c45/82781"
	assert.Equal(t, pathKey.PathName, expectedPathname)
	assert.Equal(t, pathKey.Filename, expectedFilename)
}

func TestStore(t *testing.T) {
	s := newStore()
	id := generateID()

	for i := range 10 {
		key := fmt.Sprintf("foo_%d", i)
		data := []byte("some jpg bytes")
		if _, err := s.writeStream(id, key, bytes.NewReader(data)); err != nil {
			t.Error(err)
		}

		_, r, err := s.Read(id, key)
		if err != nil {
			t.Error(err)
		}

		b, _ := io.ReadAll(r)

		assert.Equal(t, string(b), string(data))
		assert.Equal(t, s.Has(id, key), true)

		//teardown(t, s)
	}

}

func newStore() *Store {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
	}

	return NewStore(opts)
}

func teardown(t *testing.T, s *Store) {
	if err := s.Clear(); err != nil {
		t.Error(err)
	}
}
