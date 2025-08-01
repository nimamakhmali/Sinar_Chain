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

	// Event synchronization
	eventQueue chan *Event
	syncQueue  chan SyncRequest

	// Peer discovery
	discovery *PeerDiscovery
}

type PeerInfo struct {
	ID          peer.ID
	Address     string
	LastSeen    time.Time
	Events      map[EventID]bool
	IsValidator bool
	Stake       uint64
}

type EventMessage struct {
	Type      string    `json:"type"`
	Event     *Event    `json:"event,omitempty"`
	EventID   EventID   `json:"event_id,omitempty"`
	From      peer.ID   `json:"from"`
	Timestamp time.Time `json:"timestamp"`
	Round     uint64    `json:"round,omitempty"`
}

type SyncRequest struct {
	From      peer.ID   `json:"from"`
	FromRound uint64    `json:"from_round"`
	ToRound   uint64    `json:"to_round"`
	Events    []EventID `json:"events,omitempty"`
}

type SyncResponse struct {
	Events []*Event `json:"events"`
	Round  uint64   `json:"round"`
}

type PeerDiscovery struct {
	host host.Host
	ctx  context.Context
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
		host:       h,
		dag:        dag,
		peers:      make(map[peer.ID]*PeerInfo),
		ctx:        ctx,
		cancel:     cancel,
		eventQueue: make(chan *Event, 1000),
		syncQueue:  make(chan SyncRequest, 100),
		discovery:  &PeerDiscovery{host: h, ctx: ctx},
	}

	// تنظیم stream handlers
	h.SetStreamHandler("/lachesis/events/1.0.0", nm.handleEventStream)
	h.SetStreamHandler("/lachesis/sync/1.0.0", nm.handleSyncStream)
	h.SetStreamHandler("/lachesis/consensus/1.0.0", nm.handleConsensusStream)

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
		fmt.Printf("🚀 Sinar Chain Node listening on: %s/p2p/%s\n", addr, nm.host.ID())
	}

	// شروع event processing
	go nm.processEventQueue()

	// شروع sync processing
	go nm.processSyncQueue()

	// شروع peer discovery (placeholder)
	// go nm.discovery.startDiscovery()

	return nil
}

func (nm *NetworkManager) Stop() {
	nm.cancel()
	nm.host.Close()
}

// processEventQueue پردازش queue events
func (nm *NetworkManager) processEventQueue() {
	for {
		select {
		case event := <-nm.eventQueue:
			if event != nil {
				// بررسی اینکه آیا event قبلاً وجود دارد
				if _, exists := nm.dag.Events[event.Hash()]; exists {
					// Event قبلاً وجود دارد، نادیده گرفتن
					continue
				}

				// اضافه کردن event به DAG
				if err := nm.dag.AddEvent(event); err != nil {
					fmt.Printf("Failed to add event from queue: %v\n", err)
					continue
				}

				// Gossip به سایر peers
				go nm.GossipEvent(event)
			}
		case <-nm.ctx.Done():
			return
		}
	}
}

// processSyncQueue پردازش sync requests
func (nm *NetworkManager) processSyncQueue() {
	for {
		select {
		case syncReq := <-nm.syncQueue:
			go nm.handleSyncRequest(syncReq)
		case <-nm.ctx.Done():
			return
		}
	}
}

// GossipEvent ارسال event به تمام peers
func (nm *NetworkManager) GossipEvent(event *Event) error {
	eventMsg := EventMessage{
		Type:      "new_event",
		Event:     event,
		From:      nm.host.ID(),
		Timestamp: time.Now(),
		Round:     event.Round,
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
			// اضافه کردن event به queue برای پردازش
			select {
			case nm.eventQueue <- msg.Event:
			default:
				fmt.Printf("Event queue full, dropping event\n")
			}
		}
	case "sync_request":
		// درخواست sync
		var syncReq SyncRequest
		if err := json.Unmarshal([]byte(msg.EventID[:]), &syncReq); err != nil {
			fmt.Printf("Failed to decode sync request: %v\n", err)
			return
		}
		select {
		case nm.syncQueue <- syncReq:
		default:
			fmt.Printf("Sync queue full, dropping sync request\n")
		}
	}
}

func (nm *NetworkManager) handleSyncStream(stream network.Stream) {
	defer stream.Close()

	var syncReq SyncRequest
	decoder := json.NewDecoder(stream)
	if err := decoder.Decode(&syncReq); err != nil {
		fmt.Printf("Failed to decode sync request: %v\n", err)
		return
	}

	// پردازش sync request
	nm.handleSyncRequest(syncReq)
}

func (nm *NetworkManager) handleConsensusStream(stream network.Stream) {
	defer stream.Close()
	// پردازش consensus messages
}

func (nm *NetworkManager) handleSyncRequest(syncReq SyncRequest) {
	// پیدا کردن events مورد نیاز
	var events []*Event
	for _, eventID := range syncReq.Events {
		if event, exists := nm.dag.GetEvent(eventID); exists {
			events = append(events, event)
		}
	}

	// ارسال response
	response := SyncResponse{
		Events: events,
		Round:  syncReq.ToRound,
	}

	// ارسال response به peer
	go nm.sendSyncResponse(syncReq.From, response)
}

func (nm *NetworkManager) sendSyncResponse(peerID peer.ID, response SyncResponse) {
	stream, err := nm.host.NewStream(nm.ctx, peerID, "/lachesis/sync/1.0.0")
	if err != nil {
		fmt.Printf("Failed to create sync stream to %s: %v\n", peerID, err)
		return
	}
	defer stream.Close()

	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(response); err != nil {
		fmt.Printf("Failed to encode sync response: %v\n", err)
		return
	}
}

func (nm *NetworkManager) handlePeerConnected(n network.Network, conn network.Conn) {
	peerID := conn.RemotePeer()
	nm.mu.Lock()
	defer nm.mu.Unlock()

	nm.peers[peerID] = &PeerInfo{
		ID:          peerID,
		Address:     conn.RemoteMultiaddr().String(),
		LastSeen:    time.Now(),
		Events:      make(map[EventID]bool),
		IsValidator: false, // Will be updated later
		Stake:       0,
	}

	fmt.Printf("✅ Peer connected: %s (%s)\n", peerID, conn.RemoteMultiaddr())
}

func (nm *NetworkManager) handlePeerDisconnected(n network.Network, conn network.Conn) {
	peerID := conn.RemotePeer()
	nm.mu.Lock()
	defer nm.mu.Unlock()

	delete(nm.peers, peerID)
	fmt.Printf("❌ Peer disconnected: %s\n", peerID)
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

	fmt.Printf("🔗 Connected to peer: %s\n", peer.ID)
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

// RequestSync درخواست sync از peer
func (nm *NetworkManager) RequestSync(peerID peer.ID, fromRound, toRound uint64, events []EventID) error {
	syncReq := SyncRequest{
		From:      nm.host.ID(),
		FromRound: fromRound,
		ToRound:   toRound,
		Events:    events,
	}

	stream, err := nm.host.NewStream(nm.ctx, peerID, "/lachesis/sync/1.0.0")
	if err != nil {
		return fmt.Errorf("failed to create sync stream: %v", err)
	}
	defer stream.Close()

	encoder := json.NewEncoder(stream)
	return encoder.Encode(syncReq)
}

// GetNetworkStats آمار شبکه
func (nm *NetworkManager) GetNetworkStats() map[string]interface{} {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_peers"] = len(nm.peers)
	stats["validators"] = 0
	stats["total_stake"] = uint64(0)

	for _, peer := range nm.peers {
		if peer.IsValidator {
			stats["validators"] = stats["validators"].(int) + 1
			stats["total_stake"] = stats["total_stake"].(uint64) + peer.Stake
		}
	}

	return stats
}
