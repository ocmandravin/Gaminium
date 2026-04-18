package network

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/ocamndravin/gaminium/config"
	"github.com/ocamndravin/gaminium/internal/blockchain"
	"github.com/ocamndravin/gaminium/internal/crypto"
	"github.com/ocamndravin/gaminium/internal/wallet"
)

// MessageType defines the P2P message protocol.
type MessageType uint8

const (
	MsgVersion    MessageType = 1
	MsgVerAck     MessageType = 2
	MsgPing       MessageType = 3
	MsgPong       MessageType = 4
	MsgGetBlocks  MessageType = 5
	MsgBlock      MessageType = 6
	MsgTx         MessageType = 7
	MsgGetPeers   MessageType = 8
	MsgPeers      MessageType = 9
	MsgInv        MessageType = 10
	MsgGetData    MessageType = 11
)

// Message is the base P2P network message.
type Message struct {
	Type    MessageType
	Payload []byte
}

// Peer represents a connected network peer.
type Peer struct {
	conn       net.Conn
	addr       *PeerAddr
	version    uint32
	lastPing   time.Time
	mu         sync.Mutex
}

// Node is the GAMINIUM P2P network node.
type Node struct {
	mu        sync.RWMutex
	chain     *blockchain.Chain
	mempool   *Mempool
	discovery *Discovery
	peers     map[string]*Peer

	listener  net.Listener
	port      int
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewNode creates a new P2P node.
func NewNode(chain *blockchain.Chain, mempool *Mempool, port int) *Node {
	ctx, cancel := context.WithCancel(context.Background())
	disc := NewDiscovery(
		config.MaxPeers,
		MainnetDNSSeeds,
		MainnetBootstraps,
	)
	return &Node{
		chain:     chain,
		mempool:   mempool,
		discovery: disc,
		peers:     make(map[string]*Peer),
		port:      port,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Start begins listening for connections and discovers peers.
func (n *Node) Start() error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", n.port))
	if err != nil {
		return fmt.Errorf("node: listen: %w", err)
	}
	n.listener = ln

	go n.acceptLoop()
	go n.discoverPeers()
	go n.pingLoop()
	go n.mempoolCleanup()

	return nil
}

// Stop shuts down the node.
func (n *Node) Stop() {
	n.cancel()
	if n.listener != nil {
		n.listener.Close()
	}
	n.mu.Lock()
	for _, p := range n.peers {
		p.conn.Close()
	}
	n.mu.Unlock()
}

// BroadcastTx broadcasts a transaction to all connected peers.
func (n *Node) BroadcastTx(tx *wallet.Transaction) {
	data, err := json.Marshal(tx)
	if err != nil {
		return
	}
	n.broadcast(Message{Type: MsgTx, Payload: data})
}

// BroadcastBlock broadcasts a block to all connected peers.
func (n *Node) BroadcastBlock(block *blockchain.Block) {
	// Minimal inv message with block hash
	hash := block.Header.Hash()
	n.broadcast(Message{Type: MsgInv, Payload: hash[:]})
}

// PeerCount returns the number of connected peers.
func (n *Node) PeerCount() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return len(n.peers)
}

func (n *Node) acceptLoop() {
	for {
		select {
		case <-n.ctx.Done():
			return
		default:
		}

		conn, err := n.listener.Accept()
		if err != nil {
			select {
			case <-n.ctx.Done():
				return
			default:
				continue
			}
		}

		go n.handleConnection(conn)
	}
}

func (n *Node) handleConnection(conn net.Conn) {
	peer := &Peer{
		conn: conn,
		addr: &PeerAddr{
			IP:   net.ParseIP(conn.RemoteAddr().String()),
			Port: uint16(config.DefaultPort),
		},
		lastPing: time.Now(),
	}

	key := conn.RemoteAddr().String()
	n.mu.Lock()
	if len(n.peers) >= config.MaxPeers {
		n.mu.Unlock()
		conn.Close()
		return
	}
	n.peers[key] = peer
	n.mu.Unlock()

	defer func() {
		n.mu.Lock()
		delete(n.peers, key)
		n.mu.Unlock()
		conn.Close()
	}()

	// Send version message
	n.sendVersion(peer)

	// Read loop
	for {
		select {
		case <-n.ctx.Done():
			return
		default:
		}

		msg, err := readMessage(conn)
		if err != nil {
			return
		}
		n.handleMessage(peer, msg)
	}
}

func (n *Node) handleMessage(peer *Peer, msg Message) {
	switch msg.Type {
	case MsgVersion:
		sendMsg(peer.conn, Message{Type: MsgVerAck})
	case MsgPing:
		sendMsg(peer.conn, Message{Type: MsgPong, Payload: msg.Payload})
	case MsgGetPeers:
		n.sendPeers(peer)
	case MsgTx:
		n.handleTx(msg.Payload)
	case MsgInv:
		// Request full data for unknown items
		sendMsg(peer.conn, Message{Type: MsgGetData, Payload: msg.Payload})
	case MsgBlock:
		n.handleBlock(msg.Payload)
	case MsgGetBlocks:
		n.sendBlocks(peer, msg.Payload)
	}
}

func (n *Node) handleTx(payload []byte) {
	var tx wallet.Transaction
	if err := json.Unmarshal(payload, &tx); err != nil {
		return
	}
	n.mempool.Add(&tx) //nolint: errcheck
}

func (n *Node) handleBlock(payload []byte) {
	// Block processing handled by sync module
	_ = payload
}

func (n *Node) sendBlocks(peer *Peer, payload []byte) {
	// Send blocks from the given hash forward
	if len(payload) < crypto.HashSize {
		return
	}
	var fromHash crypto.Hash
	copy(fromHash[:], payload[:crypto.HashSize])
	_ = fromHash
}

func (n *Node) sendPeers(peer *Peer) {
	peers := n.discovery.RandomPeers(25)
	data, _ := json.Marshal(peers)
	sendMsg(peer.conn, Message{Type: MsgPeers, Payload: data})
}

func (n *Node) sendVersion(peer *Peer) {
	payload := make([]byte, 8)
	binary.BigEndian.PutUint32(payload[0:4], 1) // protocol version
	binary.BigEndian.PutUint32(payload[4:8], uint32(n.chain.Height()))
	sendMsg(peer.conn, Message{Type: MsgVersion, Payload: payload})
}

func (n *Node) broadcast(msg Message) {
	n.mu.RLock()
	peers := make([]*Peer, 0, len(n.peers))
	for _, p := range n.peers {
		peers = append(peers, p)
	}
	n.mu.RUnlock()

	for _, p := range peers {
		sendMsg(p.conn, msg)
	}
}

func (n *Node) discoverPeers() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	// Discover immediately on start
	n.discovery.Discover()
	n.connectToNewPeers()

	for {
		select {
		case <-n.ctx.Done():
			return
		case <-ticker.C:
			n.discovery.Discover()
			n.connectToNewPeers()
		}
	}
}

func (n *Node) connectToNewPeers() {
	n.mu.RLock()
	currentCount := len(n.peers)
	n.mu.RUnlock()

	if currentCount >= config.MaxPeers {
		return
	}

	needed := config.MaxPeers - currentCount
	candidates := n.discovery.RandomPeers(needed * 2)

	for _, addr := range candidates {
		n.mu.RLock()
		_, connected := n.peers[addr.String()]
		n.mu.RUnlock()
		if connected {
			continue
		}

		go func(a *PeerAddr) {
			conn, err := net.DialTimeout("tcp", a.String(), 10*time.Second)
			if err != nil {
				return
			}
			n.handleConnection(conn)
		}(addr)
	}
}

func (n *Node) pingLoop() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-n.ctx.Done():
			return
		case <-ticker.C:
			n.mu.RLock()
			for _, peer := range n.peers {
				sendMsg(peer.conn, Message{Type: MsgPing})
			}
			n.mu.RUnlock()
		}
	}
}

func (n *Node) mempoolCleanup() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-n.ctx.Done():
			return
		case <-ticker.C:
			n.mempool.ExpireOld()
		}
	}
}

func sendMsg(conn net.Conn, msg Message) {
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	// 5 byte header: 1 type + 4 length
	header := make([]byte, 5)
	header[0] = byte(msg.Type)
	binary.BigEndian.PutUint32(header[1:], uint32(len(msg.Payload)))
	conn.Write(header)
	if len(msg.Payload) > 0 {
		conn.Write(msg.Payload)
	}
}

func readMessage(conn net.Conn) (Message, error) {
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	header := make([]byte, 5)
	if _, err := io.ReadFull(conn, header); err != nil {
		return Message{}, err
	}
	msgType := MessageType(header[0])
	length := binary.BigEndian.Uint32(header[1:])

	if length > 32*1024*1024 { // 32MB max message
		return Message{}, fmt.Errorf("message too large: %d bytes", length)
	}

	payload := make([]byte, length)
	if length > 0 {
		if _, err := io.ReadFull(conn, payload); err != nil {
			return Message{}, err
		}
	}

	return Message{Type: msgType, Payload: payload}, nil
}
