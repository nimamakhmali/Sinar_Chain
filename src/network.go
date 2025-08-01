package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	"github.com/multiformats/go-multiaddr"
)

// NetworkManager مدیریت شبکه P2P
type NetworkManager struct {
	host   host.Host
	dag    *DAG
	peers  map[peer.ID]*PeerInfo
	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
}

type PeerInfo struct {
	ID       peer.ID
	Address  string
	LastSeen time.Time
	Events   map[EventID]bool
}

type EventMessage struct {
	Type      string    `json:"type"`
	Event     *Event    `json:"event,omitempty"`
	EventID   EventID   `json:"event_id,omitempty"`
	From      peer.ID   `json:"from"`
	Timestamp time.Time `json:"timestamp"`
}

func NewNetworkManager(dag *DAG) (*NetworkManager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// ایجاد host با libp2p
	h, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"),
		libp2p.Security(noise.ID, noise.New),
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create host: %v", err)
	}

	nm := &NetworkManager{
		host:   h,
		dag:    dag,
		peers:  make(map[peer.ID]*PeerInfo),
		ctx:    ctx,
		cancel: cancel,
	}

	// تنظیم stream handlers
	h.SetStreamHandler("/lachesis/events/1.0.0", nm.handleEventStream)
	h.SetStreamHandler("/lachesis/sync/1.0.0", nm.handleSyncStream)

	// تنظیم connection handlers
	h.Network().Notify(&network.NotifyBundle{
		ConnectedF:    nm.handlePeerConnected,
		DisconnectedF: nm.handlePeerDisconnected,
	})

	return nm, nil
}

func (nm *NetworkManager) Start() error {
	// نمایش آدرس‌های listening
	addrs := nm.host.Addrs()
	for _, addr := range addrs {
		fmt.Printf("Listening on: %s/p2p/%s\n", addr, nm.host.ID())
	}
	return nil
}

func (nm *NetworkManager) Stop() {
	nm.cancel()
	nm.host.Close()
}

// GossipEvent ارسال event به تمام peers
func (nm *NetworkManager) GossipEvent(event *Event) error {
	eventMsg := EventMessage{
		Type:      "new_event",
		Event:     event,
		From:      nm.host.ID(),
		Timestamp: time.Now(),
	}

	nm.mu.RLock()
	peers := make([]peer.ID, 0, len(nm.peers))
	for pid := range nm.peers {
		peers = append(peers, pid)
	}
	nm.mu.RUnlock()

	for _, pid := range peers {
		go nm.sendEventToPeer(pid, eventMsg)
	}

	return nil
}

func (nm *NetworkManager) sendEventToPeer(peerID peer.ID, msg EventMessage) {
	stream, err := nm.host.NewStream(nm.ctx, peerID, "/lachesis/events/1.0.0")
	if err != nil {
		fmt.Printf("Failed to create stream to %s: %v\n", peerID, err)
		return
	}
	defer stream.Close()

	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(msg); err != nil {
		fmt.Printf("Failed to encode message: %v\n", err)
		return
	}
}

func (nm *NetworkManager) handleEventStream(stream network.Stream) {
	defer stream.Close()

	var msg EventMessage
	decoder := json.NewDecoder(stream)
	if err := decoder.Decode(&msg); err != nil {
		fmt.Printf("Failed to decode message: %v\n", err)
		return
	}

	switch msg.Type {
	case "new_event":
		if msg.Event != nil {
			// اضافه کردن event به DAG
			if err := nm.dag.AddEvent(msg.Event); err != nil {
				fmt.Printf("Failed to add event: %v\n", err)
				return
			}

			// Gossip به سایر peers
			go nm.GossipEvent(msg.Event)
		}
	}
}

func (nm *NetworkManager) handleSyncStream(stream network.Stream) {
	defer stream.Close()
	// پیاده‌سازی sync logic
}

func (nm *NetworkManager) handlePeerConnected(n network.Network, conn network.Conn) {
	peerID := conn.RemotePeer()
	nm.mu.Lock()
	defer nm.mu.Unlock()

	nm.peers[peerID] = &PeerInfo{
		ID:       peerID,
		Address:  conn.RemoteMultiaddr().String(),
		LastSeen: time.Now(),
		Events:   make(map[EventID]bool),
	}

	fmt.Printf("Peer connected: %s\n", peerID)
}

func (nm *NetworkManager) handlePeerDisconnected(n network.Network, conn network.Conn) {
	peerID := conn.RemotePeer()
	nm.mu.Lock()
	defer nm.mu.Unlock()

	delete(nm.peers, peerID)
	fmt.Printf("Peer disconnected: %s\n", peerID)
}

// ConnectToPeer اتصال به یک peer جدید
func (nm *NetworkManager) ConnectToPeer(addr string) error {
	maddr, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		return fmt.Errorf("invalid multiaddr: %v", err)
	}

	peer, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return fmt.Errorf("failed to parse peer info: %v", err)
	}

	if err := nm.host.Connect(nm.ctx, *peer); err != nil {
		return fmt.Errorf("failed to connect to peer: %v", err)
	}

	return nil
}

// GetPeers دریافت لیست peers
func (nm *NetworkManager) GetPeers() []*PeerInfo {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	peers := make([]*PeerInfo, 0, len(nm.peers))
	for _, p := range nm.peers {
		peers = append(peers, p)
	}
	return peers
}
