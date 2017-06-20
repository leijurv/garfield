package main

import (
	"encoding/json"
	"io"
)

type Meta struct {
	raw  []byte
	data map[string]interface{}
}

func (meta Meta) GetData(key string) (interface{}, bool) {
	a, b := meta.data[key] //I wish I could do return meta.data[key] but go is stupid =(
	return a, b
}
func (meta Meta) Verify() bool {
	//TODO
	return true
}
func readMeta(peer *Peer) (*Meta, error) {
	metaLenBytes := make([]byte, 1)
	_, err := io.ReadFull(peer.Conn, metaLenBytes)
	if err != nil {
		return nil, err
	}
	metaLen := int(uint8(metaLenBytes[0]))

	metaBytes := make([]byte, metaLen)
	_, err = io.ReadFull(peer.Conn, metaBytes)
	if err != nil {
		return nil, err
	}
	meta := Meta{raw: metaBytes}
	err = json.Unmarshal(metaBytes, &(meta.data))
	if err != nil {
		return nil, err
	}
	return &meta, nil
}
func (meta Meta) Write() []byte {
	if len(meta.raw) > 255 {
		panic("not long enough")
	}
	return append([]byte{uint8(len(meta.raw))}, meta.raw...)
}
