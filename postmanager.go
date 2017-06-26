package main

import (
	"encoding/hex"
	"errors"
	"math"
	"os"
	"path/filepath"
	"sync"
)

var (
	ErrPayloadExists    error = errors.New("payloadcache: writing payload that already exists")
	ErrPayloadMissing   error = errors.New("payloadcache: fetching payload that does not exist")
	ErrPayloadOversized error = errors.New("payloadcache: reading a payload that is too large (payload > 2^16 bytes)")

	ErrPostMissing error = errors.New("postcache: fetching post that does not exist")
)

type PostManager struct {
	PayloadBacking PayloadCache
	PostBacking    PostCache
}

type PayloadCache interface {
	GetPayload(PayloadHash) (*Payload, error)
	HasPayload(PayloadHash) bool
	WritePayload(PayloadHash, *Payload) error
}

type PostCache interface {
	GetPost(PayloadHash) (*Post, error)
	WritePost(PayloadHash, *Post) error
}

func (manager *PostManager) FetchPayload(post *Post) error {
	post.lock.Lock()
	defer post.lock.Unlock()
	if post.Payload != nil {
		return nil
	}
	payload, err := manager.PayloadBacking.GetPayload(post.PayloadHash)
	if err != nil {
		return err
	}
	post.Payload = payload
	return nil
}

type MemoryPostCache struct {
	lock    sync.RWMutex
	content map[PayloadHash]*Post
}

func (cache *MemoryPostCache) GetPost(hash PayloadHash) (*Post, error) {
	cache.lock.RLock()
	defer cache.lock.RUnlock()
	post, ok := cache.content[hash]
	if !ok {
		return post, ErrPostMissing
	}
	return post, nil
}

func (cache *MemoryPostCache) WritePost(hash PayloadHash, post *Post) error {
	cache.lock.Lock()
	defer cache.lock.Unlock()
	cache.content[hash] = post
	return nil
}

type MemoryPayloadCache struct {
	lock    sync.RWMutex
	content map[PayloadHash]*Payload
}

func (cache *MemoryPayloadCache) GetPayload(hash PayloadHash) (*Payload, error) {
	cache.lock.RLock()
	defer cache.lock.RUnlock()
	payload, ok := cache.content[hash]
	if !ok {
		return nil, ErrPayloadMissing
	}
	return payload, nil
}

func (cache *MemoryPayloadCache) HasPayload(hash PayloadHash) bool {
	cache.lock.RLock()
	defer cache.lock.RUnlock()
	_, ok := cache.content[hash]
	return ok
}

func (cache *MemoryPayloadCache) WritePayload(hash PayloadHash, payload *Payload) error {
	cache.lock.Lock()
	defer cache.lock.Unlock()
	if _, ok := cache.content[hash]; ok {
		return ErrPayloadExists
	}
	cache.content[hash] = payload
	return nil
}

type DiskPayloadCache struct {
	lock    sync.RWMutex
	content map[PayloadHash]*Payload
}

func GetHashPath(hash PayloadHash) string {
	return filepath.Join(hex.EncodeToString(hash[:1]), hex.EncodeToString(hash[1:]))
}

func (cache *DiskPayloadCache) GetPayload(hash PayloadHash) (*Payload, error) {
	if !cache.HasPayload(hash) {
		return nil, ErrPayloadMissing
	}
	cache.lock.Lock()
	defer cache.lock.Unlock()
	if payload, ok := cache.content[hash]; ok {
		return payload, nil
	}
	file, err := os.Open(GetHashPath(hash))
	if err != nil {
		return nil, err
	}
	defer func(file *os.File) {
		if err := file.Close(); err != nil {
			Error.Println(err)
			return
		}
		// TODO: Maybe add some extra shutdown logic.
	}(file)
	payload := Payload{}
	read, err := file.Read(payload)
	if err != nil {
		return nil, err
	}
	if read > math.MaxUint16 {
		return nil, ErrPayloadOversized
	}
	cache.content[hash] = &payload
	return &payload, nil
}

func (cache *DiskPayloadCache) HasPayload(hash PayloadHash) bool {
	cache.lock.RLock()
	defer cache.lock.RUnlock()
	_, ok := cache.content[hash]
	if ok {
		return ok
	}
	if _, err := os.Stat(GetHashPath(hash)); err == nil {
		return true
	}
	return false
}

func (cache *DiskPayloadCache) WritePayload(hash PayloadHash, payload *Payload) error {
	if cache.HasPayload(hash) {
		return ErrPayloadExists
	}
	file, err := os.OpenFile(GetHashPath(hash), os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		if err := file.Close(); err != nil {
			Error.Println(err)
			return
		}
		// TODO: Maybe add some extra shutdown logic.
	}(file)
	_, err = file.Write(*payload)
	if err != nil {
		return err
	}
	return nil
}
