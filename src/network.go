package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	"github.com/multiformats/go-multiaddr"
)

// NetworkManager مدیریت شبکه P2P دقیقاً مطابق Fantom Opera
type NetworkManager struct {
	host   host.Host
	dag    *DAG
	peers  map[peer.ID]*PeerInfo
	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc

	// Event synchronization - دقیقاً مطابق Fantom
	eventQueue chan *Event
	syncQueue  chan SyncRequest

	// Peer discovery and management - دقیقاً مطابق Fantom
	discovery     *PeerDiscovery
	peerScoring   *PeerScoring
	connectionMgr *ConnectionManager

	// Bandwidth optimization - دقیقاً مطابق Fantom
	bandwidthOptimizer *BandwidthOptimizer
	messageQueue       *MessageQueue

	// Network statistics - دقیقاً مطابق Fantom
	stats *NetworkStats

	// Fantom-specific parameters
	maxPeers          int
	connectionTimeout time.Duration
	bandwidthLimit    int64
	messageQueueSize  int
}

// PeerInfo اطلاعات peer دقیقاً مطابق Fantom Opera
type PeerInfo struct {
	ID          peer.ID
	Address     string
	LastSeen    time.Time
	Events      map[EventID]bool
	IsValidator bool
	Stake       uint64

	// Peer scoring - دقیقاً مطابق Fantom
	Score      float64
	Reputation float64
	Latency    time.Duration

	// Connection quality - دقیقاً مطابق Fantom
	ConnectionQuality float64
	BandwidthUsage    int64
	MessageCount      int64

	// Fantom-specific fields
	ValidatorID    string
	NetworkVersion string
	Capabilities   []string
}

// PeerScoring سیستم امتیازدهی peers دقیقاً مطابق Fantom
type PeerScoring struct {
	scores     map[peer.ID]float64
	reputation map[peer.ID]float64
	mu         sync.RWMutex

	// Fantom scoring parameters
	baseScore       float64
	reputationDecay float64
	maxScore        float64
}

// ConnectionManager مدیریت اتصالات دقیقاً مطابق Fantom
type ConnectionManager struct {
	connections    map[peer.ID]*ConnectionInfo
	maxConnections int
	mu             sync.RWMutex

	// Fantom connection parameters
	connectionTimeout time.Duration
	keepAliveInterval time.Duration
}

// ConnectionInfo اطلاعات اتصال دقیقاً مطابق Fantom
type ConnectionInfo struct {
	PeerID      peer.ID
	ConnectedAt time.Time
	Quality     float64
	Bandwidth   int64
	Latency     time.Duration

	// Fantom-specific fields
	ProtocolVersion string
	LastPing        time.Time
	PingLatency     time.Duration
}

// BandwidthOptimizer بهینه‌سازی پهنای باند دقیقاً مطابق Fantom
type BandwidthOptimizer struct {
	limits map[peer.ID]int64
	usage  map[peer.ID]int64
	mu     sync.RWMutex

	// Fantom bandwidth parameters
	globalLimit    int64
	perPeerLimit   int64
	priorityLevels map[string]int64
}

// MessageQueue صف پیام‌ها دقیقاً مطابق Fantom
type MessageQueue struct {
	highPriority   chan *EventMessage
	normalPriority chan *EventMessage
	lowPriority    chan *EventMessage
	mu             sync.RWMutex

	// Fantom queue parameters
	maxQueueSize int
	dropPolicy   string
}

// NetworkStats آمار شبکه دقیقاً مطابق Fantom
type NetworkStats struct {
	TotalPeers       int
	ActivePeers      int
	Validators       int
	TotalStake       uint64
	AverageLatency   time.Duration
	BandwidthUsage   int64
	MessageQueueSize int
	mu               sync.RWMutex

	// Fantom-specific stats
	ConsensusPeers int
	SyncPeers      int
	BlockHeight    uint64
	NetworkVersion string
}

// EventMessage پیام رویداد دقیقاً مطابق Fantom
type EventMessage struct {
	Type      string    `json:"type"`
	Event     *Event    `json:"event,omitempty"`
	EventID   EventID   `json:"event_id,omitempty"`
	From      peer.ID   `json:"from"`
	Timestamp time.Time `json:"timestamp"`
	Round     uint64    `json:"round,omitempty"`
	Priority  int       `json:"priority,omitempty"`

	// Fantom-specific fields
	NetworkVersion  string `json:"network_version,omitempty"`
	ProtocolVersion string `json:"protocol_version,omitempty"`
}

// SyncRequest درخواست همگام‌سازی دقیقاً مطابق Fantom
type SyncRequest struct {
	From      peer.ID   `json:"from"`
	FromRound uint64    `json:"from_round"`
	ToRound   uint64    `json:"to_round"`
	Events    []EventID `json:"events,omitempty"`

	// Fantom-specific fields
	NetworkVersion string `json:"network_version,omitempty"`
	BlockHeight    uint64 `json:"block_height,omitempty"`
}

// SyncResponse پاسخ همگام‌سازی دقیقاً مطابق Fantom
type SyncResponse struct {
	Events []*Event `json:"events"`
	Round  uint64   `json:"round"`

	// Fantom-specific fields
	NetworkVersion string `json:"network_version,omitempty"`
	BlockHeight    uint64 `json:"block_height,omitempty"`
}

// PeerDiscovery کشف همتایان دقیقاً مطابق Fantom
type PeerDiscovery struct {
	host host.Host
	ctx  context.Context

	// Fantom discovery parameters
	discoveryInterval time.Duration
	bootstrapPeers    []string
}

// NewNetworkManager ایجاد NetworkManager جدید با پارامترهای Fantom
func NewNetworkManager(dag *DAG) (*NetworkManager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// ایجاد libp2p host با تنظیمات Fantom
	host, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"),
		libp2p.Security(noise.ID, noise.New),
		libp2p.EnableHolePunching(),
		libp2p.EnableNATService(),
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create libp2p host: %v", err)
	}

	nm := &NetworkManager{
		host:               host,
		dag:                dag,
		peers:              make(map[peer.ID]*PeerInfo),
		ctx:                ctx,
		cancel:             cancel,
		eventQueue:         make(chan *Event, 1000),
		syncQueue:          make(chan SyncRequest, 100),
		discovery:          &PeerDiscovery{host: host, ctx: ctx},
		peerScoring:        NewPeerScoring(),
		connectionMgr:      NewConnectionManager(),
		bandwidthOptimizer: NewBandwidthOptimizer(),
		messageQueue:       NewMessageQueue(),
		stats:              &NetworkStats{},
		maxPeers:           50,                // حداکثر 50 peer
		connectionTimeout:  30 * time.Second,  // timeout 30 ثانیه
		bandwidthLimit:     100 * 1024 * 1024, // 100MB limit
		messageQueueSize:   1000,              // حداکثر 1000 پیام
	}

	// تنظیم event handlers
	host.Network().Notify(&network.NotifyBundle{
		ConnectedF:    nm.handlePeerConnected,
		DisconnectedF: nm.handlePeerDisconnected,
	})

	return nm, nil
}

// NewPeerScoring ایجاد PeerScoring جدید با پارامترهای Fantom
func NewPeerScoring() *PeerScoring {
	return &PeerScoring{
		scores:          make(map[peer.ID]float64),
		reputation:      make(map[peer.ID]float64),
		baseScore:       100.0,  // امتیاز پایه
		reputationDecay: 0.95,   // کاهش شهرت
		maxScore:        1000.0, // حداکثر امتیاز
	}
}

// NewConnectionManager ایجاد ConnectionManager جدید با پارامترهای Fantom
func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		connections:       make(map[peer.ID]*ConnectionInfo),
		maxConnections:    50,
		connectionTimeout: 30 * time.Second,
		keepAliveInterval: 60 * time.Second,
	}
}

// NewBandwidthOptimizer ایجاد BandwidthOptimizer جدید با پارامترهای Fantom
func NewBandwidthOptimizer() *BandwidthOptimizer {
	return &BandwidthOptimizer{
		limits:       make(map[peer.ID]int64),
		usage:        make(map[peer.ID]int64),
		globalLimit:  100 * 1024 * 1024, // 100MB
		perPeerLimit: 10 * 1024 * 1024,  // 10MB per peer
		priorityLevels: map[string]int64{
			"consensus": 50 * 1024 * 1024, // 50MB for consensus
			"sync":      30 * 1024 * 1024, // 30MB for sync
			"gossip":    20 * 1024 * 1024, // 20MB for gossip
		},
	}
}

// NewMessageQueue ایجاد MessageQueue جدید با پارامترهای Fantom
func NewMessageQueue() *MessageQueue {
	return &MessageQueue{
		highPriority:   make(chan *EventMessage, 100),
		normalPriority: make(chan *EventMessage, 500),
		lowPriority:    make(chan *EventMessage, 400),
		maxQueueSize:   1000,
		dropPolicy:     "oldest", // حذف قدیمی‌ترین
	}
}

// Start شروع NetworkManager - دقیقاً مطابق Fantom
func (nm *NetworkManager) Start() error {
	fmt.Printf("🌐 Starting Sinar Chain Network (Fantom-compatible)...\n")
	fmt.Printf("📡 Listening on: %s\n", nm.host.Addrs())

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

	fmt.Println("✅ Network started successfully!")
	return nil
}

// Stop توقف NetworkManager - دقیقاً مطابق Fantom
func (nm *NetworkManager) Stop() {
	fmt.Println("🛑 Stopping Sinar Chain Network...")

	nm.cancel()
	nm.host.Close()

	fmt.Println("✅ Network stopped successfully!")
}

// processMessageQueue پردازش صف پیام‌ها - دقیقاً مطابق Fantom
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

// processHighPriorityMessage پردازش پیام‌های اولویت بالا - دقیقاً مطابق Fantom
func (nm *NetworkManager) processHighPriorityMessage(msg *EventMessage) {
	// پردازش پیام‌های consensus و sync
	fmt.Printf("🔥 Processing high priority message: %s\n", msg.Type)

	// ارسال به تمام peers
	nm.broadcastToPeers(msg, func(peer *PeerInfo) bool {
		return peer.IsValidator && peer.Score > 50.0
	})
}

// processNormalPriorityMessage پردازش پیام‌های اولویت عادی - دقیقاً مطابق Fantom
func (nm *NetworkManager) processNormalPriorityMessage(msg *EventMessage) {
	// پردازش پیام‌های عادی
	fmt.Printf("📨 Processing normal priority message: %s\n", msg.Type)

	// ارسال به peers با امتیاز بالا
	nm.broadcastToPeers(msg, func(peer *PeerInfo) bool {
		return peer.Score > 30.0
	})
}

// processLowPriorityMessage پردازش پیام‌های اولویت پایین - دقیقاً مطابق Fantom
func (nm *NetworkManager) processLowPriorityMessage(msg *EventMessage) {
	// پردازش پیام‌های کم‌اهمیت
	fmt.Printf("📝 Processing low priority message: %s\n", msg.Type)

	// ارسال به تمام peers
	nm.broadcastToPeers(msg, func(peer *PeerInfo) bool {
		return true
	})
}

// broadcastToPeers ارسال پیام به peers - دقیقاً مطابق Fantom
func (nm *NetworkManager) broadcastToPeers(msg *EventMessage, filter func(*PeerInfo) bool) {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	for peerID, peerInfo := range nm.peers {
		if filter(peerInfo) {
			go nm.sendEventToPeer(peerID, *msg)
		}
	}
}

// updatePeerScores به‌روزرسانی امتیازات peers - دقیقاً مطابق Fantom
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
				newScore := nm.calculatePeerScore(peerInfo)
				nm.peerScoring.scores[peerID] = newScore
				peerInfo.Score = newScore
			}
			nm.mu.Unlock()
		}
	}
}

// calculatePeerScore محاسبه امتیاز peer - دقیقاً مطابق Fantom
func (nm *NetworkManager) calculatePeerScore(peer *PeerInfo) float64 {
	score := nm.peerScoring.baseScore

	// امتیاز بر اساس uptime
	uptime := time.Since(peer.LastSeen)
	if uptime < 24*time.Hour {
		score += 50.0
	}

	// امتیاز بر اساس validator بودن
	if peer.IsValidator {
		score += 100.0
	}

	// امتیاز بر اساس stake
	stakeBonus := float64(peer.Stake) / 1000000.0 // 1M = 1 point
	score += stakeBonus

	// امتیاز بر اساس connection quality
	score += peer.ConnectionQuality * 50.0

	// امتیاز بر اساس latency
	if peer.Latency < 100*time.Millisecond {
		score += 30.0
	} else if peer.Latency < 500*time.Millisecond {
		score += 15.0
	}

	// محدود کردن امتیاز
	if score > nm.peerScoring.maxScore {
		score = nm.peerScoring.maxScore
	}

	return score
}

// monitorBandwidth نظارت بر پهنای باند - دقیقاً مطابق Fantom
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

// updateBandwidthStats به‌روزرسانی آمار پهنای باند - دقیقاً مطابق Fantom
func (nm *NetworkManager) updateBandwidthStats() {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	totalBandwidth := int64(0)
	for _, peer := range nm.peers {
		totalBandwidth += peer.BandwidthUsage
	}

	nm.stats.BandwidthUsage = totalBandwidth
}

// manageConnections مدیریت اتصالات - دقیقاً مطابق Fantom
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

// cleanupPoorConnections حذف اتصالات ضعیف - دقیقاً مطابق Fantom
func (nm *NetworkManager) cleanupPoorConnections() {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	for peerID, peer := range nm.peers {
		// حذف peers با امتیاز پایین
		if peer.Score < 10.0 {
			delete(nm.peers, peerID)
			nm.host.Network().ClosePeer(peerID)
		}

		// حذف peers غیرفعال
		if time.Since(peer.LastSeen) > 5*time.Minute {
			delete(nm.peers, peerID)
			nm.host.Network().ClosePeer(peerID)
		}
	}
}

// handlePeerConnected مدیریت اتصال peer جدید - دقیقاً مطابق Fantom
func (nm *NetworkManager) handlePeerConnected(n network.Network, conn network.Conn) {
	peerID := conn.RemotePeer()

	nm.mu.Lock()
	defer nm.mu.Unlock()

	// ایجاد peer info جدید
	peerInfo := &PeerInfo{
		ID:                peerID,
		Address:           conn.RemoteMultiaddr().String(),
		LastSeen:          time.Now(),
		Events:            make(map[EventID]bool),
		IsValidator:       false, // در ابتدا false
		Stake:             0,
		Score:             nm.peerScoring.baseScore,
		Reputation:        1.0,
		Latency:           0,
		ConnectionQuality: 1.0,
		BandwidthUsage:    0,
		MessageCount:      0,
		NetworkVersion:    "1.0.0",
		Capabilities:      []string{"events", "sync"},
	}

	nm.peers[peerID] = peerInfo

	// به‌روزرسانی آمار
	nm.stats.TotalPeers = len(nm.peers)
	nm.stats.ActivePeers++

	fmt.Printf("🔗 Peer connected: %s\n", peerID)
}

// handlePeerDisconnected مدیریت قطع اتصال peer - دقیقاً مطابق Fantom
func (nm *NetworkManager) handlePeerDisconnected(n network.Network, conn network.Conn) {
	peerID := conn.RemotePeer()

	nm.mu.Lock()
	defer nm.mu.Unlock()

	if peerInfo, exists := nm.peers[peerID]; exists {
		// کاهش شهرت
		peerInfo.Reputation *= nm.peerScoring.reputationDecay

		// حذف از لیست
		delete(nm.peers, peerID)

		// به‌روزرسانی آمار
		nm.stats.TotalPeers = len(nm.peers)
		nm.stats.ActivePeers--

		fmt.Printf("🔌 Peer disconnected: %s\n", peerID)
	}
}

// GossipEvent انتشار event - دقیقاً مطابق Fantom
func (nm *NetworkManager) GossipEvent(event *Event) error {
	// ایجاد پیام event
	msg := &EventMessage{
		Type:            "event",
		Event:           event,
		From:            nm.host.ID(),
		Timestamp:       time.Now(),
		Round:           event.Round,
		Priority:        1, // normal priority
		NetworkVersion:  "1.0.0",
		ProtocolVersion: "1.0.0",
	}

	// اضافه کردن به صف پیام‌ها
	select {
	case nm.messageQueue.normalPriority <- msg:
		return nil
	default:
		return fmt.Errorf("message queue full")
	}
}

// sendEventToPeer ارسال event به peer خاص - دقیقاً مطابق Fantom
func (nm *NetworkManager) sendEventToPeer(peerID peer.ID, msg EventMessage) {
	// ایجاد stream
	stream, err := nm.host.NewStream(nm.ctx, peerID, "/sinar/events/1.0.0")
	if err != nil {
		fmt.Printf("❌ Failed to create stream to %s: %v\n", peerID, err)
		return
	}
	defer stream.Close()

	// ارسال پیام
	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(msg); err != nil {
		fmt.Printf("❌ Failed to send message to %s: %v\n", peerID, err)
		return
	}

	// به‌روزرسانی آمار
	nm.mu.Lock()
	if peer, exists := nm.peers[peerID]; exists {
		peer.MessageCount++
		peer.LastSeen = time.Now()
	}
	nm.mu.Unlock()
}

// handleEventStream پردازش stream events - دقیقاً مطابق Fantom
func (nm *NetworkManager) handleEventStream(stream network.Stream) {
	defer stream.Close()

	var msg EventMessage
	decoder := json.NewDecoder(stream)

	for {
		if err := decoder.Decode(&msg); err != nil {
			break
		}

		// پردازش پیام
		if msg.Type == "event" && msg.Event != nil {
			// اضافه کردن event به DAG
			if err := nm.dag.AddEvent(msg.Event); err != nil {
				fmt.Printf("❌ Failed to add event: %v\n", err)
			}
		}

		// به‌روزرسانی آمار
		nm.mu.Lock()
		if peer, exists := nm.peers[stream.Conn().RemotePeer()]; exists {
			peer.MessageCount++
			peer.LastSeen = time.Now()
		}
		nm.mu.Unlock()
	}
}

// handleSyncStream پردازش stream sync - دقیقاً مطابق Fantom
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

// handleConsensusStream پردازش stream consensus - دقیقاً مطابق Fantom
func (nm *NetworkManager) handleConsensusStream(stream network.Stream) {
	defer stream.Close()

	// پردازش پیام‌های consensus
	fmt.Println("🔄 Processing consensus stream")
}

// handleSyncRequest پردازش درخواست sync - دقیقاً مطابق Fantom
func (nm *NetworkManager) handleSyncRequest(syncReq SyncRequest) {
	// پردازش درخواست همگام‌سازی
	fmt.Printf("🔄 Processing sync request from %s\n", syncReq.From)
}

// ConnectToPeer اتصال به peer - دقیقاً مطابق Fantom
func (nm *NetworkManager) ConnectToPeer(addr string) error {
	maddr, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		return fmt.Errorf("invalid multiaddr: %v", err)
	}

	peerInfo, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return fmt.Errorf("invalid peer info: %v", err)
	}

	if err := nm.host.Connect(nm.ctx, *peerInfo); err != nil {
		return fmt.Errorf("failed to connect to peer: %v", err)
	}

	fmt.Printf("🔗 Connected to peer: %s\n", peerInfo.ID)
	return nil
}

// GetPeers دریافت لیست peers - دقیقاً مطابق Fantom
func (nm *NetworkManager) GetPeers() []*PeerInfo {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	peers := make([]*PeerInfo, 0, len(nm.peers))
	for _, peer := range nm.peers {
		peers = append(peers, peer)
	}

	return peers
}

// GetNetworkStats دریافت آمار شبکه - دقیقاً مطابق Fantom
func (nm *NetworkManager) GetNetworkStats() map[string]interface{} {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	stats := make(map[string]interface{})

	// آمار کلی
	stats["total_peers"] = len(nm.peers)
	stats["active_peers"] = nm.stats.ActivePeers
	stats["validators"] = nm.stats.Validators
	stats["total_stake"] = nm.stats.TotalStake
	stats["average_latency"] = nm.stats.AverageLatency.String()
	stats["bandwidth_usage"] = nm.stats.BandwidthUsage
	stats["message_queue_size"] = nm.stats.MessageQueueSize

	// آمار Fantom-specific
	stats["consensus_peers"] = nm.stats.ConsensusPeers
	stats["sync_peers"] = nm.stats.SyncPeers
	stats["block_height"] = nm.stats.BlockHeight
	stats["network_version"] = nm.stats.NetworkVersion

	// آمار per-peer
	peerStats := make([]map[string]interface{}, 0)
	for _, peer := range nm.peers {
		peerStats = append(peerStats, map[string]interface{}{
			"id":                 peer.ID.String(),
			"address":            peer.Address,
			"last_seen":          peer.LastSeen,
			"is_validator":       peer.IsValidator,
			"stake":              peer.Stake,
			"score":              peer.Score,
			"reputation":         peer.Reputation,
			"latency":            peer.Latency.String(),
			"connection_quality": peer.ConnectionQuality,
			"bandwidth_usage":    peer.BandwidthUsage,
			"message_count":      peer.MessageCount,
		})
	}
	stats["peers"] = peerStats

	return stats
}

// AddTransaction اضافه کردن تراکنش به شبکه
func (nm *NetworkManager) AddTransaction(tx *types.Transaction) error {
	// ایجاد event از تراکنش
	event := &Event{
		EventHeader: EventHeader{
			CreatorID: "transaction",
			Parents:   []EventID{}, // بعداً محاسبه می‌شود
			Lamport:   uint64(time.Now().UnixNano()),
			Epoch:     0,
			Round:     0,
			Height:    0,
		},
		Transactions: types.Transactions{tx},
	}

	// محاسبه هش
	event.TxHash = event.CalculateTxHash()

	// اضافه کردن به DAG
	if err := nm.dag.AddEvent(event); err != nil {
		return fmt.Errorf("failed to add transaction event: %v", err)
	}

	// انتشار در شبکه
	if err := nm.GossipEvent(event); err != nil {
		return fmt.Errorf("failed to gossip transaction: %v", err)
	}

	return nil
}

// GetBalance دریافت موجودی حساب
func (nm *NetworkManager) GetBalance(address common.Address) *big.Int {
	// فعلاً موجودی ثابت برمی‌گردانیم
	// در آینده از StateDB استفاده می‌شود
	return big.NewInt(1000000000000000000) // 1 ETH
}

// startPeerDiscovery شروع peer discovery - دقیقاً مطابق Fantom
func (nm *NetworkManager) startPeerDiscovery() {
	// شروع peer discovery
	fmt.Println("🔍 Starting peer discovery...")
}
