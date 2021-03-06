// The MIT License (MIT)
//
// Copyright (c) 2016 RapidLoop
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

// Package skv provides a simple persistent key-value store for Go values. It
// can store a mapping of string to any gob-encodable Go value. It is
// lightweight and performant, and ideal for use in low-traffic websites,
// utilities and the like.
//
// The API is very simple - you can Put(), Get() or Delete() entries. These
// methods are goroutine-safe.
//
// skv uses BoltDB for storage and the encoding/gob package for encoding and
// decoding values. There are no other dependencies.

// Modified: 2018-02-14
// Updated to accept a bucket name on Open(), Get(), Put(), and Delete()
package main

import (
	"bytes"
	"encoding/gob"
	"errors"
	"time"

	"github.com/boltdb/bolt"
)

// KVStore represents the key value store. Use the Open() method to create
// one, and Close() it when done.
type KVStore struct {
	db *bolt.DB
}

var (
	// ErrNotFound is returned when the key supplied to a Get or Delete
	// method does not exist in the database.
	ErrNotFound = errors.New("skv: key not found")

	// ErrBadValue is returned when the value supplied to the Put method
	// is nil.
	ErrBadValue = errors.New("skv: bad value")
)

// Open a key-value store. "path" is the full path to the database file, any
// leading directories must have been created already. File is created with
// mode 0640 if needed.
//
// Because of BoltDB restrictions, only one process may open the file at a
// time. Attempts to open the file from another process will fail with a
// timeout error.
func NewKVStore(path string) (*KVStore, error) {
	opts := &bolt.Options{
		Timeout: 50 * time.Millisecond,
	}

	db, err := bolt.Open(path, 0640, opts)
	if err != nil {
		return nil, err
	}

	return &KVStore{db: db}, nil
}

// AddBucket creates a new bucket if it does not exist. AddBucket returns an
// error, which needs to be checked.
func (kvs *KVStore) AddBucket(bucket string) error {
	err := kvs.db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucket))
		return err
	})
	if err != nil {
		return err
	}

	return nil
}

// Put an entry into the store. The passed value is gob-encoded and stored.
// The key can be an empty string, but the value cannot be nil - if it is,
// Put() returns ErrBadValue.
//
//	err := store.Put("key42", 156)
//	err := store.Put("key42", "this is a string")
//	m := map[string]int{
//	    "harry": 100,
//	    "emma":  101,
//	}
//	err := store.Put("key43", m)
func (kvs *KVStore) Put(bucket, key string, value interface{}) error {
	if value == nil {
		return ErrBadValue
	}

	if err := kvs.AddBucket(bucket); err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(value); err != nil {
		return nil
	}
	return kvs.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte(bucket)).Put([]byte(key), buf.Bytes())
	})
}

// Get an entry from the store. "value" must be a pointer-typed. If the key
// is not present in the store, Get returns ErrNotFound.
//
//	type MyStruct struct {
//	    Numbers []int
//	}
//	var val MyStruct
//	if err := store.Get("key42", &val); err == skv.ErrNotFound {
//	    // "key42" not found
//	} else if err != nil {
//	    // an error occurred
//	} else {
//	    // ok
//	}
//
// The value passed to Get() can be nil, in which case any value read from
// the store is silently discarded.
//
//  if err := store.Get("key42", nil); err == nil {
//      fmt.Println("entry is present")
//  }
func (kvs *KVStore) Get(bucket, key string, value interface{}) error {
	return kvs.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte(bucket)).Cursor()

		k, v := c.Seek([]byte(key))
		if k == nil || string(k) != key {
			return ErrNotFound
		}

		if v == nil {
			return nil
		}

		d := gob.NewDecoder(bytes.NewReader(v))

		return d.Decode(value)
	})
}

// Delete the entry with the given key. If no such key is present in the store,
// it returns ErrNotFound.
//
//	store.Delete("key42")
func (kvs *KVStore) Delete(bucket, key string) error {
	return kvs.db.Update(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte(bucket)).Cursor()
		if k, _ := c.Seek([]byte(key)); k == nil || string(k) != key {
			return ErrNotFound
		} else {
			return c.Delete()
		}
	})
}

// Close closes the key-value store file.
func (kvs *KVStore) Close() error {
	return kvs.db.Close()
}
