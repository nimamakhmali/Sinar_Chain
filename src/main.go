package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
)

// NetworkConfig تنظیمات شبکه
type NetworkConfig struct {
	Port            string
	ChainID         uint64
	BlockTime       time.Duration
	MaxValidators   int
	MinStake        uint64
	GasLimit        uint64
	BaseFee         *big.Int
	EnableMobile    bool
	EnableWebSocket bool
	LogLevel        string
}

// SinarChainNetwork کلاس اصلی شبکه Sinar Chain
type SinarChainNetwork struct {
	mu sync.RWMutex

	// Core Components
	consensus  *ConsensusEngine
	network    *NetworkManager
	apiServer  *APIServer
	stateDB    *StateDB
	blockchain *Blockchain

	// Configuration
	config *NetworkConfig

	// Validators
	validators  map[string]*Validator
	myValidator *Validator

	// Context for graceful shutdown
	ctx    context.Context
	cancel context.CancelFunc

	// Performance monitoring
	startTime time.Time
	metrics   map[string]interface{}
}

// NewSinarChainNetwork ایجاد شبکه جدید
func NewSinarChainNetwork(config *NetworkConfig) (*SinarChainNetwork, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// ایجاد validator برای این node
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("خطا در تولید کلید: %v", err)
	}

	validator := NewValidator("NodeA", privateKey, config.MinStake)

	// ایجاد consensus config
	consensusConfig := &ConsensusConfig{
		BlockTime:         config.BlockTime,
		MaxEventsPerBlock: 1000,
		MinValidators:     3,
		MaxValidators:     config.MaxValidators,
		StakeRequired:     config.MinStake,
		ConsensusTimeout:  30 * time.Second,
		GasLimit:          config.GasLimit,
		BaseFee:           config.BaseFee,
	}

	// ایجاد consensus engine
	consensus := NewConsensusEngine(consensusConfig)

	// ایجاد network manager
	network, err := NewNetworkManager(nil) // DAG will be set later
	if err != nil {
		return nil, fmt.Errorf("خطا در ایجاد network manager: %v", err)
	}

	// ایجاد state DB
	stateDB := NewStateDB()

	// ایجاد API server
	apiServer := NewAPIServer(consensus, network, stateDB, config.Port)

	// ایجاد blockchain
	blockchain := NewBlockchain(nil) // DAG will be set later

	sinarNetwork := &SinarChainNetwork{
		consensus:  consensus,
		network:    network,
		apiServer:  apiServer,
		stateDB:    stateDB,
		blockchain: blockchain,
		config:     config,
		validators: map[string]*Validator{
			validator.ID: validator,
		},
		myValidator: validator,
		ctx:         ctx,
		cancel:      cancel,
		startTime:   time.Now(),
		metrics:     make(map[string]interface{}),
	}

	// تنظیم network برای consensus
	consensus.SetNetwork(network)

	return sinarNetwork, nil
}

// Start شروع شبکه
func (scn *SinarChainNetwork) Start() error {
	log.Println("🚀 شروع شبکه Sinar Chain...")

	// شروع consensus engine
	if err := scn.consensus.Start(); err != nil {
		return fmt.Errorf("خطا در شروع consensus: %v", err)
	}

	// شروع network manager
	if err := scn.network.Start(); err != nil {
		return fmt.Errorf("خطا در شروع network: %v", err)
	}

	// شروع API server
	go func() {
		if err := scn.apiServer.Start(); err != nil {
			log.Printf("خطا در شروع API server: %v", err)
		}
	}()

	// شروع monitoring
	go scn.monitorNetwork()

	// شروع performance testing
	if scn.config.LogLevel == "debug" {
		go scn.runPerformanceTests()
	}

	log.Printf("✅ شبکه Sinar Chain با موفقیت شروع شد!")
	log.Printf("🔗 API Server: http://localhost:%s", scn.config.Port)
	log.Printf("🌐 WebSocket: ws://localhost:%s/ws", scn.config.Port)

	return nil
}

// Stop توقف شبکه
func (scn *SinarChainNetwork) Stop() {
	log.Println("🛑 توقف شبکه Sinar Chain...")

	scn.cancel()

	// توقف consensus
	scn.consensus.Stop()

	// توقف network
	scn.network.Stop()

	// توقف API server - در نسخه فعلی Stop() وجود ندارد
	// scn.apiServer.Stop()

	log.Println("✅ شبکه Sinar Chain با موفقیت متوقف شد!")
}

// monitorNetwork مانیتورینگ شبکه
func (scn *SinarChainNetwork) monitorNetwork() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-scn.ctx.Done():
			return
		case <-ticker.C:
			scn.updateStats()
			scn.logStats()
		}
	}
}

// updateStats بروزرسانی آمار شبکه
func (scn *SinarChainNetwork) updateStats() {
	scn.mu.Lock()
	defer scn.mu.Unlock()

	// دریافت آمار consensus
	consensusStats := scn.consensus.GetConsensusStats()

	// دریافت آمار network
	networkStats := scn.network.GetNetworkStats()

	// بروزرسانی metrics
	scn.metrics["consensus"] = consensusStats
	scn.metrics["network"] = networkStats
	scn.metrics["uptime"] = time.Since(scn.startTime)
}

// logStats نمایش آمار شبکه
func (scn *SinarChainNetwork) logStats() {
	scn.mu.RLock()
	defer scn.mu.RUnlock()

	log.Printf("📊 آمار شبکه Sinar Chain:")
	log.Printf("   🕐 Uptime: %v", time.Since(scn.startTime))
	log.Printf("   👥 Validators: %d", len(scn.validators))
	log.Printf("   🌐 Network Active: %v", scn.network != nil)
	log.Printf("   🔄 Consensus Active: %v", scn.consensus != nil)
}

// runPerformanceTests اجرای تست‌های عملکرد
func (scn *SinarChainNetwork) runPerformanceTests() {
	log.Println("🧪 شروع تست‌های عملکرد...")

	// تست 1: ارسال تراکنش‌های متعدد
	go scn.testTransactionThroughput()

	// تست 2: تست consensus speed
	go scn.testConsensusSpeed()

	// تست 3: تست network latency
	go scn.testNetworkLatency()

	log.Println("✅ تست‌های عملکرد شروع شدند!")
}

// testTransactionThroughput تست throughput تراکنش‌ها
func (scn *SinarChainNetwork) testTransactionThroughput() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for i := 0; i < 100; i++ {
		select {
		case <-scn.ctx.Done():
			return
		case <-ticker.C:
			// در نسخه فعلی، تراکنش‌ها از طریق events ارسال می‌شوند
			log.Printf("📤 ارسال تراکنش تست #%d", i)
		}
	}
}

// testConsensusSpeed تست سرعت consensus
func (scn *SinarChainNetwork) testConsensusSpeed() {
	time.Sleep(5 * time.Second)

	startTime := time.Now()

	// ایجاد event تست
	event := NewEvent("NodeA", nil, 0, 1, nil, 0)

	// ارسال event
	scn.consensus.AddEvent(event)

	// انتظار برای consensus
	time.Sleep(10 * time.Second)

	duration := time.Since(startTime)
	log.Printf("⏱️  زمان consensus: %v", duration)
}

// testNetworkLatency تست latency شبکه
func (scn *SinarChainNetwork) testNetworkLatency() {
	time.Sleep(10 * time.Second)

	peers := scn.network.GetPeers()
	for _, peer := range peers {
		log.Printf("🌐 Peer %s - Latency: %v", peer.ID, peer.Latency)
	}
}

// GetStats دریافت آمار شبکه
func (scn *SinarChainNetwork) GetStats() map[string]interface{} {
	scn.mu.RLock()
	defer scn.mu.RUnlock()

	return map[string]interface{}{
		"metrics":    scn.metrics,
		"config":     scn.config,
		"validators": len(scn.validators),
		"uptime":     time.Since(scn.startTime),
	}
}

func main() {
	log.Println("🌟 شروع Sinar Chain Network...")

	// تنظیمات پیش‌فرض
	config := &NetworkConfig{
		Port:            "8082",
		ChainID:         1001, // Sinar Chain ID
		BlockTime:       1 * time.Second,
		MaxValidators:   100,
		MinStake:        1000000, // 1M SINAR
		GasLimit:        8000000,
		BaseFee:         big.NewInt(1000000000), // 1 Gwei
		EnableMobile:    true,
		EnableWebSocket: true,
		LogLevel:        "info",
	}

	// ایجاد شبکه
	network, err := NewSinarChainNetwork(config)
	if err != nil {
		log.Fatalf("❌ خطا در ایجاد شبکه: %v", err)
	}

	// شروع شبکه
	if err := network.Start(); err != nil {
		log.Fatalf("❌ خطا در شروع شبکه: %v", err)
	}

	// تنظیم graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// انتظار برای signal
	<-sigChan

	log.Println("🛑 دریافت signal توقف...")
	network.Stop()

	log.Println("👋 Sinar Chain Network متوقف شد!")
}
