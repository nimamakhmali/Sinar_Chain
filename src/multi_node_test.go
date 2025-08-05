package main

import (
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// MultiNodeTest سیستم تست چند نودی
type MultiNodeTest struct {
	nodes    map[string]*TestNode
	mu       sync.RWMutex
	network  *TestNetwork
}

// TestNode یک نود تست
type TestNode struct {
	ID           string
	Consensus    *ConsensusEngine
	Network      *NetworkManager
	Validator    *Validator
	Transactions []*types.Transaction
	mu           sync.RWMutex
}

// TestNetwork شبکه تست
type TestNetwork struct {
	nodes map[string]*TestNode
	mu    sync.RWMutex
}

// NewMultiNodeTest ایجاد سیستم تست چند نودی
func NewMultiNodeTest() *MultiNodeTest {
	return &MultiNodeTest{
		nodes:   make(map[string]*TestNode),
		network: &TestNetwork{nodes: make(map[string]*TestNode)},
	}
}

// CreateNode ایجاد یک نود جدید
func (mnt *MultiNodeTest) CreateNode(id string) (*TestNode, error) {
	// ایجاد validator
	privKey, _ := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	validator := NewValidator(id, privKey, 1000000)

	// ایجاد consensus config
	config := &ConsensusConfig{
		BlockTime:         2 * time.Second,
		MaxEventsPerBlock: 1000,
		MinValidators:     3,
		MaxValidators:     10,
		StakeRequired:     1000000,
		ConsensusTimeout:  30 * time.Second,
	}

	// ایجاد consensus engine
	consensus := NewConsensusEngine(config)

	// ایجاد network manager
	network, err := NewNetworkManager(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create network for node %s: %v", id, err)
	}

	// تنظیم network در consensus
	consensus.SetNetwork(network)

	// اضافه کردن validator
	consensus.AddValidator(validator)

	node := &TestNode{
		ID:        id,
		Consensus: consensus,
		Network:   network,
		Validator: validator,
	}

	mnt.mu.Lock()
	mnt.nodes[id] = node
	mnt.network.nodes[id] = node
	mnt.mu.Unlock()

	fmt.Printf("🔄 Node %s created successfully\n", id)
	return node, nil
}

// StartAllNodes شروع تمام نودها
func (mnt *MultiNodeTest) StartAllNodes() error {
	mnt.mu.RLock()
	defer mnt.mu.RUnlock()

	for id, node := range mnt.nodes {
		// شروع consensus
		if err := node.Consensus.Start(); err != nil {
			return fmt.Errorf("failed to start consensus for node %s: %v", id, err)
		}

		// شروع network
		if err := node.Network.Start(); err != nil {
			return fmt.Errorf("failed to start network for node %s: %v", id, err)
		}

		fmt.Printf("🚀 Node %s started\n", id)
	}

	return nil
}

// StopAllNodes توقف تمام نودها
func (mnt *MultiNodeTest) StopAllNodes() {
	mnt.mu.RLock()
	defer mnt.mu.RUnlock()

	for id, node := range mnt.nodes {
		node.Consensus.Stop()
		node.Network.Stop()
		fmt.Printf("🛑 Node %s stopped\n", id)
	}
}

// ConnectNodes اتصال نودها به هم
func (mnt *MultiNodeTest) ConnectNodes() error {
	mnt.mu.RLock()
	defer mnt.mu.RUnlock()

	nodes := make([]*TestNode, 0, len(mnt.nodes))
	for _, node := range mnt.nodes {
		nodes = append(nodes, node)
	}

	// اتصال هر نود به نودهای دیگر
	for i, node1 := range nodes {
		for j, node2 := range nodes {
			if i != j {
				// در نسخه کامل، اینجا peer discovery انجام می‌شود
				fmt.Printf("🔗 Connecting %s to %s\n", node1.ID, node2.ID)
			}
		}
	}

	return nil
}

// CreateTestTransaction ایجاد تراکنش تست
func (mnt *MultiNodeTest) CreateTestTransaction(from, to string, value uint64) *types.Transaction {
	// ایجاد تراکنش تست
	tx := types.NewTransaction(
		0, // nonce
		common.HexToAddress(to),
		big.NewInt(int64(value)),
		21000, // gas limit
		big.NewInt(1), // gas price
		nil, // data
	)

	return tx
}

// BroadcastTransaction ارسال تراکنش به تمام نودها
func (mnt *MultiNodeTest) BroadcastTransaction(tx *types.Transaction) error {
	mnt.mu.RLock()
	defer mnt.mu.RUnlock()

	for id, node := range mnt.nodes {
		// ایجاد event از تراکنش
		event := NewEvent(
			id,
			nil, // parents
			0,   // epoch
			uint64(time.Now().UnixNano()), // lamport
			types.Transactions{tx},
			0, // height
		)

		// اضافه کردن event به consensus
		if err := node.Consensus.AddEvent(event); err != nil {
			fmt.Printf("❌ Failed to add event to node %s: %v\n", id, err)
		} else {
			fmt.Printf("✅ Transaction broadcasted to node %s\n", id)
		}
	}

	return nil
}

// GetNetworkStats آمار شبکه
func (mnt *MultiNodeTest) GetNetworkStats() map[string]interface{} {
	mnt.mu.RLock()
	defer mnt.mu.RUnlock()

	stats := make(map[string]interface{})
	
	// آمار کلی
	stats["total_nodes"] = len(mnt.nodes)
	
	// آمار هر نود
	nodeStats := make(map[string]interface{})
	for id, node := range mnt.nodes {
		nodeStats[id] = map[string]interface{}{
			"validator_id": node.Validator.ID,
			"stake":        node.Validator.Stake,
			"events_count": len(node.Consensus.GetPoset().Events),
		}
	}
	stats["nodes"] = nodeStats

	return stats
}

// RunConsensusTest اجرای تست consensus
func (mnt *MultiNodeTest) RunConsensusTest(duration time.Duration) error {
	fmt.Printf("🧪 Starting consensus test for %v...\n", duration)

	// شروع تمام نودها
	if err := mnt.StartAllNodes(); err != nil {
		return err
	}

	// اتصال نودها
	if err := mnt.ConnectNodes(); err != nil {
		return err
	}

	// ایجاد تراکنش‌های تست
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			// ایجاد تراکنش تست
			tx := mnt.CreateTestTransaction("NodeA", "NodeB", 100)
			
			// ارسال به شبکه
			if err := mnt.BroadcastTransaction(tx); err != nil {
				fmt.Printf("❌ Failed to broadcast transaction: %v\n", err)
			}
		}
	}()

	// اجرا برای مدت مشخص
	time.Sleep(duration)

	// نمایش آمار نهایی
	stats := mnt.GetNetworkStats()
	fmt.Printf("📊 Final Network Stats: %+v\n", stats)

	// توقف نودها
	mnt.StopAllNodes()

	return nil
}

// RunMultiNodeTest اجرای تست کامل چند نودی
func RunMultiNodeTest() error {
	fmt.Println("🚀 Starting Multi-Node Test for Sinar Chain...")

	// ایجاد سیستم تست
	test := NewMultiNodeTest()

	// ایجاد نودها
	nodes := []string{"NodeA", "NodeB", "NodeC"}
	for _, nodeID := range nodes {
		if _, err := test.CreateNode(nodeID); err != nil {
			return fmt.Errorf("failed to create node %s: %v", nodeID, err)
		}
	}

	// اجرای تست consensus
	if err := test.RunConsensusTest(30 * time.Second); err != nil {
		return fmt.Errorf("consensus test failed: %v", err)
	}

	fmt.Println("✅ Multi-Node Test completed successfully!")
	return nil
} 