package main

type PacketType uint8

// Update ids sent to peers
const (
	PacketNonceUpdate PacketType = iota
	PacketPayload
	PacketPayloadRequest
	PacketGetNonce
	PacketMultiNonce
)

var packetHandlers = map[PacketType]func(*Peer) error{
	PacketNonceUpdate:    readPacketMultiNonce,
	PacketPayload:        readPacketPayload,
	PacketPayloadRequest: readPacketPayloadRequest,
	PacketGetNonce:       readPacketGetNonce,
	PacketMultiNonce:     readPacketMultiNonce,
}
