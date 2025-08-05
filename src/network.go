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

// NetworkManager مدیریت شبکه P2P مشابه Fantom Opera
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

	// Peer discovery and management
	discovery     *PeerDiscovery
	peerScoring   *PeerScoring
	connectionMgr *ConnectionManager

	// Bandwidth optimization
	bandwidthOptimizer *BandwidthOptimizer
	messageQueue       *MessageQueue

	// Network statistics
	stats *NetworkStats
}

// PeerInfo اطلاعات peer مشابه Fantom Opera
type PeerInfo struct {
	ID          peer.ID
	Address     string
	LastSeen    time.Time
	Events      map[EventID]bool
	IsValidator bool
	Stake       uint64

	// Peer scoring
	Score      float64
	Reputation float64
	Latency    time.Duration

	// Connection quality
	ConnectionQuality float64
	BandwidthUsage    int64
	MessageCount      int64
}

// PeerScoring سیستم امتیازدهی peers
type PeerScoring struct {
	scores     map[peer.ID]float64
	reputation map[peer.ID]float64
	mu         sync.RWMutex
}

// ConnectionManager مدیریت اتصالات
type ConnectionManager struct {
	connections    map[peer.ID]*ConnectionInfo
	maxConnections int
	mu             sync.RWMutex
}

// ConnectionInfo اطلاعات اتصال
type ConnectionInfo struct {
	PeerID      peer.ID
	ConnectedAt time.Time
	Quality     float64
	Bandwidth   int64
	Latency     time.Duration
}

// BandwidthOptimizer بهینه‌سازی پهنای باند
type BandwidthOptimizer struct {
	limits map[peer.ID]int64
	usage  map[peer.ID]int64
	mu     sync.RWMutex
}

// MessageQueue صف پیام‌ها
type MessageQueue struct {
	highPriority   chan *EventMessage
	normalPriority chan *EventMessage
	lowPriority    chan *EventMessage
	mu             sync.RWMutex
}

// NetworkStats آمار شبکه
type NetworkStats struct {
	TotalPeers       int
	ActivePeers      int
	Validators       int
	TotalStake       uint64
	AverageLatency   time.Duration
	BandwidthUsage   int64
	MessageQueueSize int
	mu               sync.RWMutex
}

// EventMessage پیام event مشابه Fantom Opera
type EventMessage struct {
	Type      string    `json:"type"`
	Event     *Event    `json:"event,omitempty"`
	EventID   EventID   `json:"event_id,omitempty"`
	From      peer.ID   `json:"from"`
	Timestamp time.Time `json:"timestamp"`
	Round     uint64    `json:"round,omitempty"`
	Priority  int       `json:"priority,omitempty"`
}

// SyncRequest درخواست sync
type SyncRequest struct {
	From      peer.ID   `json:"from"`
	FromRound uint64    `json:"from_round"`
	ToRound   uint64    `json:"to_round"`
	Events    []EventID `json:"events,omitempty"`
}

// SyncResponse پاسخ sync
type SyncResponse struct {
	Events []*Event `json:"events"`
	Round  uint64   `json:"round"`
}

// PeerDiscovery کشف peers
type PeerDiscovery struct {
	host host.Host
	ctx  context.Context
}

// NewNetworkManager ایجاد NetworkManager جدید
func NewNetworkManager(dag *DAG) (*NetworkManager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// ایجاد host با تنظیمات بهینه مشابه Fantom Opera
	h, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"),
		libp2p.Security(noise.ID, noise.New),
		libp2p.EnableAutoRelay(),
		libp2p.EnableHolePunching(),
		libp2p.EnableNATService(),
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
		peerScoring: &PeerScoring{
			scores:     make(map[peer.ID]float64),
			reputation: make(map[peer.ID]float64),
		},
		connectionMgr: &ConnectionManager{
			connections:    make(map[peer.ID]*ConnectionInfo),
			maxConnections: 100,
		},
		bandwidthOptimizer: &BandwidthOptimizer{
			limits: make(map[peer.ID]int64),
			usage:  make(map[peer.ID]int64),
		},
		messageQueue: &MessageQueue{
			highPriority:   make(chan *EventMessage, 100),
			normalPriority: make(chan *EventMessage, 500),
			lowPriority:    make(chan *EventMessage, 200),
		},
		stats: &NetworkStats{},
	}

	// تنظیم stream handlers مشابه Fantom Opera
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

// Start شروع NetworkManager
func (nm *NetworkManager) Start() error {
	// شروع peer discovery
	go nm.startPeerDiscovery()

	// شروع message processing
	go nm.processMessageQueue()

	// شروع peer scoring updates
	go nm.updatePeerScores()

	// شروع bandwidth monitoring
	go nm.monitorBandwidth()

	// شروع connection management
	go nm.manageConnections()

	fmt.Printf("🌐 Network Manager started on %s\n", nm.host.Addrs()[0])
	return nil
}

// Stop توقف NetworkManager
func (nm *NetworkManager) Stop() {
	if nm.cancel != nil {
		nm.cancel()
	}

	// بستن تمام اتصالات
	for peerID := range nm.peers {
		nm.host.Network().ClosePeer(peerID)
	}

	close(nm.eventQueue)
	close(nm.syncQueue)
}

// processMessageQueue پردازش صف پیام‌ها
func (nm *NetworkManager) processMessageQueue() {
	for {
		select {
		case <-nm.ctx.Done():
			return
		case msg := <-nm.messageQueue.highPriority:
			nm.processHighPriorityMessage(msg)
		case msg := <-nm.messageQueue.normalPriority:
			nm.processNormalPriorityMessage(msg)
		case msg := <-nm.messageQueue.lowPriority:
			nm.processLowPriorityMessage(msg)
		}
	}
}

// processHighPriorityMessage پردازش پیام‌های با اولویت بالا
func (nm *NetworkManager) processHighPriorityMessage(msg *EventMessage) {
	// پیام‌های consensus و events مهم
	nm.broadcastToPeers(msg, func(peer *PeerInfo) bool {
		return peer.IsValidator && peer.Score > 0.7
	})
}

// processNormalPriorityMessage پردازش پیام‌های عادی
func (nm *NetworkManager) processNormalPriorityMessage(msg *EventMessage) {
	// پیام‌های events عادی
	nm.broadcastToPeers(msg, func(peer *PeerInfo) bool {
		return peer.Score > 0.5
	})
}

// processLowPriorityMessage پردازش پیام‌های با اولویت پایین
func (nm *NetworkManager) processLowPriorityMessage(msg *EventMessage) {
	// پیام‌های sync و stats
	nm.broadcastToPeers(msg, func(peer *PeerInfo) bool {
		return peer.Score > 0.3
	})
}

// broadcastToPeers ارسال به peers با فیلتر
func (nm *NetworkManager) broadcastToPeers(msg *EventMessage, filter func(*PeerInfo) bool) {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	for peerID, peerInfo := range nm.peers {
		if filter(peerInfo) {
			go nm.sendEventToPeer(peerID, *msg)
		}
	}
}

// updatePeerScores به‌روزرسانی امتیازات peers
func (nm *NetworkManager) updatePeerScores() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-nm.ctx.Done():
			return
		case <-ticker.C:
			nm.mu.Lock()
			for peerID, peerInfo := range nm.peers {
				// محاسبه امتیاز جدید بر اساس performance
				newScore := nm.calculatePeerScore(peerInfo)
				nm.peerScoring.scores[peerID] = newScore
				peerInfo.Score = newScore
			}
			nm.mu.Unlock()
		}
	}
}

// calculatePeerScore محاسبه امتیاز peer
func (nm *NetworkManager) calculatePeerScore(peer *PeerInfo) float64 {
	score := 1.0

	// کاهش امتیاز بر اساس latency
	if peer.Latency > 100*time.Millisecond {
		score -= 0.2
	}

	// کاهش امتیاز بر اساس bandwidth usage
	if peer.BandwidthUsage > 1024*1024 { // 1MB
		score -= 0.1
	}

	// افزایش امتیاز برای validators
	if peer.IsValidator {
		score += 0.3
	}

	// افزایش امتیاز بر اساس reputation
	score += peer.Reputation * 0.2

	return score
}

// monitorBandwidth نظارت بر پهنای باند
func (nm *NetworkManager) monitorBandwidth() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-nm.ctx.Done():
			return
		case <-ticker.C:
			nm.updateBandwidthStats()
		}
	}
}

// updateBandwidthStats به‌روزرسانی آمار پهنای باند
func (nm *NetworkManager) updateBandwidthStats() {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	totalBandwidth := int64(0)
	for _, peer := range nm.peers {
		totalBandwidth += peer.BandwidthUsage
	}

	nm.stats.mu.Lock()
	nm.stats.BandwidthUsage = totalBandwidth
	nm.stats.mu.Unlock()
}

// manageConnections مدیریت اتصالات
func (nm *NetworkManager) manageConnections() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-nm.ctx.Done():
			return
		case <-ticker.C:
			nm.cleanupPoorConnections()
		}
	}
}

// cleanupPoorConnections حذف اتصالات ضعیف
func (nm *NetworkManager) cleanupPoorConnections() {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	for peerID, peerInfo := range nm.peers {
		if peerInfo.Score < 0.3 && !peerInfo.IsValidator {
			// حذف peer با امتیاز پایین
			delete(nm.peers, peerID)
			nm.host.Network().ClosePeer(peerID)
		}
	}
}

// handlePeerConnected مدیریت اتصال peer جدید
func (nm *NetworkManager) handlePeerConnected(n network.Network, conn network.Conn) {
	peerID := conn.RemotePeer()
	nm.mu.Lock()
	defer nm.mu.Unlock()

	nm.peers[peerID] = &PeerInfo{
		ID:                peerID,
		Address:           conn.RemoteMultiaddr().String(),
		LastSeen:          time.Now(),
		Events:            make(map[EventID]bool),
		IsValidator:       false,
		Stake:             0,
		Score:             1.0,
		Reputation:        0.5,
		Latency:           0,
		ConnectionQuality: 1.0,
		BandwidthUsage:    0,
		MessageCount:      0,
	}

	// اضافه کردن به connection manager
	nm.connectionMgr.connections[peerID] = &ConnectionInfo{
		PeerID:      peerID,
		ConnectedAt: time.Now(),
		Quality:     1.0,
		Bandwidth:   0,
		Latency:     0,
	}

	fmt.Printf("✅ Peer connected: %s (%s)\n", peerID, conn.RemoteMultiaddr())
}

// handlePeerDisconnected مدیریت قطع اتصال peer
func (nm *NetworkManager) handlePeerDisconnected(n network.Network, conn network.Conn) {
	peerID := conn.RemotePeer()
	nm.mu.Lock()
	defer nm.mu.Unlock()

	delete(nm.peers, peerID)
	delete(nm.connectionMgr.connections, peerID)
	fmt.Printf("❌ Peer disconnected: %s\n", peerID)
}

// GossipEvent انتشار event مشابه Fantom Opera
func (nm *NetworkManager) GossipEvent(event *Event) error {
	msg := &EventMessage{
		Type:      "event",
		Event:     event,
		EventID:   event.Hash(),
		From:      nm.host.ID(),
		Timestamp: time.Now(),
		Round:     event.Round,
		Priority:  1, // High priority for events
	}

	// اضافه کردن به صف پیام‌های با اولویت بالا
	select {
	case nm.messageQueue.highPriority <- msg:
	default:
		// اگر صف پر است، به صف عادی اضافه کن
		select {
		case nm.messageQueue.normalPriority <- msg:
		default:
			return fmt.Errorf("message queue full")
		}
	}

	return nil
}

// sendEventToPeer ارسال event به peer خاص
func (nm *NetworkManager) sendEventToPeer(peerID peer.ID, msg EventMessage) {
	stream, err := nm.host.NewStream(nm.ctx, peerID, "/lachesis/events/1.0.0")
	if err != nil {
		return
	}
	defer stream.Close()

	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(msg); err != nil {
		return
	}

	// به‌روزرسانی آمار peer
	nm.mu.Lock()
	if peerInfo, exists := nm.peers[peerID]; exists {
		peerInfo.MessageCount++
		peerInfo.LastSeen = time.Now()
	}
	nm.mu.Unlock()
}

// handleEventStream مدیریت stream events
func (nm *NetworkManager) handleEventStream(stream network.Stream) {
	defer stream.Close()

	var msg EventMessage
	decoder := json.NewDecoder(stream)
	if err := decoder.Decode(&msg); err != nil {
		return
	}

	// پردازش event
	if msg.Event != nil {
		select {
		case nm.eventQueue <- msg.Event:
		default:
			// اگر صف پر است، event را نادیده بگیر
		}
	}

	// به‌روزرسانی آمار peer
	nm.mu.Lock()
	if peerInfo, exists := nm.peers[stream.Conn().RemotePeer()]; exists {
		peerInfo.MessageCount++
		peerInfo.LastSeen = time.Now()
	}
	nm.mu.Unlock()
}

// handleSyncStream مدیریت stream sync
func (nm *NetworkManager) handleSyncStream(stream network.Stream) {
	defer stream.Close()

	var syncReq SyncRequest
	decoder := json.NewDecoder(stream)
	if err := decoder.Decode(&syncReq); err != nil {
		return
	}

	// پردازش درخواست sync
	nm.handleSyncRequest(syncReq)
}

// handleConsensusStream مدیریت stream consensus
func (nm *NetworkManager) handleConsensusStream(stream network.Stream) {
	defer stream.Close()

	// پردازش پیام‌های consensus
	// این بخش در نسخه کامل پیاده‌سازی می‌شود
}

// handleSyncRequest پردازش درخواست sync
func (nm *NetworkManager) handleSyncRequest(syncReq SyncRequest) {
	// پیاده‌سازی sync logic
	// این بخش در نسخه کامل پیاده‌سازی می‌شود
}

// ConnectToPeer اتصال به peer جدید
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

// GetNetworkStats آمار شبکه
func (nm *NetworkManager) GetNetworkStats() map[string]interface{} {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_peers"] = len(nm.peers)
	stats["validators"] = 0
	stats["total_stake"] = uint64(0)
	stats["average_score"] = 0.0
	stats["total_bandwidth"] = int64(0)

	totalScore := 0.0
	for _, peer := range nm.peers {
		if peer.IsValidator {
			stats["validators"] = stats["validators"].(int) + 1
			stats["total_stake"] = stats["total_stake"].(uint64) + peer.Stake
		}
		totalScore += peer.Score
		stats["total_bandwidth"] = stats["total_bandwidth"].(int64) + peer.BandwidthUsage
	}

	if len(nm.peers) > 0 {
		stats["average_score"] = totalScore / float64(len(nm.peers))
	}

	return stats
}

// startPeerDiscovery شروع peer discovery
func (nm *NetworkManager) startPeerDiscovery() {
	// پیاده‌سازی peer discovery
	// این بخش در نسخه کامل پیاده‌سازی می‌شود
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-nm.ctx.Done():
			return
		case <-ticker.C:
			// Peer discovery logic
		}
	}
}

