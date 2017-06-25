package main

import (
	"errors"
	"sync"
)

var (
	ErrPayloadExists  error = errors.New("payloadcache: writing payload that already exists")
	ErrPayloadMissing error = errors.New("payloadcache: fetching payload that does not exist")

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
	Lock    sync.RWMutex
	Content map[PayloadHash]*Post
}

func (cache *MemoryPostCache) GetPost(hash PayloadHash) (*Post, error) {
	cache.Lock.RLock()
	defer cache.Lock.RUnlock()
	post, ok := cache.Content[hash]
	if !ok {
		return post, ErrPostMissing
	}
	return post, nil
}

func (cache *MemoryPostCache) WritePost(hash PayloadHash, post *Post) error {
	cache.Lock.Lock()
	defer cache.Lock.Unlock()
	cache.Content[hash] = post
	return nil
}

type MemoryPayloadCache struct {
	Lock    sync.RWMutex
	Content map[PayloadHash]*Payload
}

func (cache *MemoryPayloadCache) GetPayload(hash PayloadHash) (*Payload, error) {
	cache.Lock.RLock()
	defer cache.Lock.RUnlock()
	payload, ok := cache.Content[hash]
	if !ok {
		return nil, ErrPayloadMissing
	}
	return payload, nil
}

func (cache *MemoryPayloadCache) HasPayload(hash PayloadHash) bool {
	cache.Lock.RLock()
	defer cache.Lock.RUnlock()
	_, ok := cache.Content[hash]
	return ok
}

func (cache *MemoryPayloadCache) WritePayload(hash PayloadHash, payload *Payload) error {
	cache.Lock.Lock()
	defer cache.Lock.Unlock()
	if _, ok := cache.Content[hash]; ok {
		return ErrPayloadExists
	}
	cache.Content[hash] = payload
	return nil
}
