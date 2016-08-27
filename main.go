package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"time"
)

const (
	PacketNonceUpdate         uint8 = 4
	PacketPostContents        uint8 = 7
	PacketPostContentsRequest uint8 = 9
)

type Post struct {
	payloadRaw            []byte
	nonce                 [32]byte
	firstReceived         time.Time
	mostRecentNonceUpdate time.Time
}

type Peer struct {
	conn      net.Conn
	writeLock sync.Mutex
}

var peers []*Peer
var peersLock sync.Mutex

var posts = make(map[[32]byte]*Post)
var postsLock sync.Mutex

func (post *Post) payloadHash() [32]byte {
	return sha256.Sum256(post.payloadRaw)
}
func (post *Post) hash() [32]byte {
	combined := append(post.payloadRaw, post.nonce[:]...)
	return sha256.Sum256(combined)
}
func (post *Post) checkPossibleNonce(newNonce [32]byte) int {
	if bytes.Equal(newNonce[:], post.nonce[:]) {
		return 0
	}
	newHash := sha256.Sum256(append(post.payloadRaw, newNonce[:]...))
	oldHash := post.hash()
	comparison := bytes.Compare(newHash[:], oldHash[:])
	return comparison
}
func randomNonce() [32]byte {
	rand.Seed(time.Now().UTC().UnixNano())
	fixedLengthIsBS := make([]byte, 32)
	rand.Read(fixedLengthIsBS)
	var nonce [32]byte
	copy(nonce[:], fixedLengthIsBS)
	return nonce
}
func (post *Post) insert() {
	postsLock.Lock()
	posts[post.payloadHash()] = post
	postsLock.Unlock()
}
func (post *Post) mine(count int) {
	currentHash := post.hash()
	nonce := randomNonce()
	for i := 0; i < count; i++ {
		newHash := sha256.Sum256(append(post.payloadRaw, nonce[:]...))
		if bytes.Compare(newHash[:], currentHash[:]) < 0 {
			currentHash = newHash
			postsLock.Lock()
			post.nonce = nonce
			post.mostRecentNonceUpdate = time.Now()
			postsLock.Unlock()
			fmt.Println("Nonce improvement, hash is now ", newHash)
			post.broadcastNonceUpdate()
		}
		nonce[31]++
		if nonce[31] == 0 {
			nonce[30]++
			if nonce[30] == 0 {
				for i := 29; i >= 0; i-- {
					nonce[i]++
					if nonce[i] != 0 {
						break
					}
				}
			}
		}
	}
}
func onNonceUpdateReceived(postPayloadHash [32]byte, newNonce [32]byte, peerFrom *Peer) {
	postsLock.Lock()
	post, ok := posts[postPayloadHash]

	if ok {
		comparison := post.checkPossibleNonce(newNonce)
		if comparison < 0 {
			oldNonce := post.nonce
			post.nonce = newNonce
			post.mostRecentNonceUpdate = time.Now()
			postsLock.Unlock()
			fmt.Println("Updating post nonce from", oldNonce[:], "to", newNonce[:])
			post.broadcastNonceUpdate()
		} else {
			postsLock.Unlock()
			if comparison != 0 { //its not equal, they actaully just proudly gave us a nonce that's WORSE
				//wow you're behind the times. let's help you out
				fmt.Println("Helping out peer", peerFrom, " that's behind the times")
				go peerFrom.send(append(append([]byte{PacketNonceUpdate}, postPayloadHash[:]...), post.nonce[:]...))
			}
		}
	} else {
		postsLock.Unlock()
		//ask the peer we received this from for the contents of the post because we don't have it
		peerFrom.send(append([]byte{PacketPostContentsRequest}, postPayloadHash[:]...))
	}

}
func (post *Post) broadcastNonceUpdate() {
	postPayloadHash := post.payloadHash()
	newNonce := post.nonce
	message := append(append([]byte{PacketNonceUpdate}, postPayloadHash[:]...), newNonce[:]...)
	peersLock.Lock()
	fmt.Println("Sending nonce update")
	for _, peer := range peers {
		go peer.send(message)
	}
	peersLock.Unlock()
}
func onPostContentsReceived(payloadRaw []byte, nonce [32]byte, peerFrom *Peer) {
	fmt.Println("Post contents:", payloadRaw)
	payloadHash := sha256.Sum256(payloadRaw)
	postsLock.Lock()
	_, ok := posts[payloadHash]
	if ok {
		fmt.Println("Already have it")
		postsLock.Unlock()
		//already have it. let's just check if the nonce is better
		onNonceUpdateReceived(payloadHash, nonce, peerFrom)
	} else {
		//dont have it, lets add it
		now := time.Now()
		post := Post{
			payloadRaw:            payloadRaw,
			nonce:                 nonce,
			firstReceived:         now,
			mostRecentNonceUpdate: now,
		}
		posts[payloadHash] = &post
		postsLock.Unlock()
		fmt.Println("Added post with payload hash", post.payloadHash(), "and normal hash", post.hash())
	}
}
func onPostContentsRequested(payloadHash [32]byte, peerFrom *Peer) {
	postsLock.Lock()
	post, ok := posts[payloadHash]
	postsLock.Unlock()
	if ok {
		payloadLenBytes := make([]byte, 2)
		binary.LittleEndian.PutUint16(payloadLenBytes, uint16(len(post.payloadRaw)))
		fmt.Println("Sending contents of ", payloadHash)
		//man I wish go was better at appending mulitple arrays. lol im probbaly doing something wrong here. BUT HEY, IT WORKS
		message := append(append(append([]byte{PacketPostContents}, payloadLenBytes...), post.payloadRaw...), post.nonce[:]...)
		fmt.Println("data:", message)
		peerFrom.send(message)
	} else {
		//um idk we don't have it. just ignore lol
	}
}
func (peer *Peer) send(msg []byte) error {
	peer.writeLock.Lock()
	defer peer.writeLock.Unlock()
	_, err := peer.conn.Write(msg)
	return err
}
func (peer *Peer) listen() error {
	for {
		msgType := make([]byte, 1)
		_, err := peer.conn.Read(msgType)
		if err != nil {
			return err
		}
		switch msgType[0] {
		case PacketNonceUpdate:
			payloadHash, err := read32(peer.conn)
			if err != nil {
				return err
			}
			newNonce, err := read32(peer.conn)
			if err != nil {
				return err
			}
			fmt.Println("Someone gave us a new nonce", newNonce, "for", payloadHash)
			go onNonceUpdateReceived(payloadHash, newNonce, peer)
		case PacketPostContents:
			payloadLenBytes := make([]byte, 2)
			_, err := io.ReadFull(peer.conn, payloadLenBytes)
			if err != nil {
				return err
			}
			payloadLen := int(binary.LittleEndian.Uint16(payloadLenBytes))
			fmt.Println("Reading payload with len", payloadLen)
			payload := make([]byte, payloadLen)
			_, err = io.ReadFull(peer.conn, payload)
			if err != nil {
				return err
			}
			nonce, err := read32(peer.conn)
			if err != nil {
				return err
			}
			fmt.Println("Someone gave us post contents")
			go onPostContentsReceived(payload, nonce, peer)
		case PacketPostContentsRequest:
			requestedPayloadHash, err := read32(peer.conn)
			if err != nil {
				return err
			}
			fmt.Println("Someone just asked us for post contents for payload hash", requestedPayloadHash)
			go onPostContentsRequested(requestedPayloadHash, peer)
		default:
			peer.conn.Close()
			fmt.Println("Unexpected prefix byte", msgType)
		}
	}
}
func read32(conn io.Reader) ([32]byte, error) {
	var data [32]byte
	_, err := io.ReadFull(conn, data[:])
	return data, err
}
func addPeer(conn net.Conn) {
	peer := Peer{conn: conn}
	peersLock.Lock()
	peers = append(peers, &peer)
	peersLock.Unlock()
	go peer.listen()
}
func listen(port int) {
	listen, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		panic(err)
	}
	fmt.Println("Listening on", port)
	for {
		conn, err := listen.Accept()
		if err != nil {
			panic(err)
		}
		fmt.Println("Connection from ", conn)
		addPeer(conn)
	}
}
func connect(port int) {
	fmt.Println("Connecting to", port)
	conn, err := net.Dial("tcp", "localhost:"+strconv.Itoa(port))
	if err != nil {
		panic(err)
	}
	addPeer(conn)
}
func main() {
	listenPort := flag.Int("listen", -1, "port to listen on")
	connectPort := flag.Int("connect", -1, "port to connect to")
	createAndMine := flag.Bool("create", false, "create and mine a post, as a test")
	flag.Parse()
	if *connectPort != -1 {
		connect(*connectPort)
	}
	if *createAndMine {
		go func() {
			post := Post{
				payloadRaw: []byte{5, 0, 2, 1},
			}
			post.insert()
			post.mine(20000000)

		}()
	}
	listen(*listenPort) //this goes last because it blocks
}
