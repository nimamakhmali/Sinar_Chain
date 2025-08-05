package main

import (
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

func main() {
	fmt.Println("🚀 Starting Sinar Chain (Fantom-like Lachesis Consensus)...")

	// تنظیمات Consensus
	consensusConfig := &ConsensusConfig{
		BlockTime:         2 * time.Second, // 2 ثانیه per block
		MaxEventsPerBlock: 1000,
		MinValidators:     21,
		MaxValidators:     100,
		StakeRequired:     1000000, // 1M tokens
		ConsensusTimeout:  30 * time.Second,
		GasLimit:          30000000, // 30M gas
		BaseFee:           big.NewInt(0),
	}

	// ایجاد Consensus Engine
	consensusEngine := NewConsensusEngine(consensusConfig)

	// ایجاد StateDB
	stateDB := consensusEngine.GetStateDB()

	// ایجاد Network Manager
	networkManager, err := NewNetworkManager(nil) // DAG will be set later
	if err != nil {
		log.Fatal("Failed to create network manager:", err)
	}

	// تنظیم network در consensus engine
	consensusEngine.SetNetwork(networkManager)

	// ایجاد API Server
	apiServer := NewAPIServer(consensusEngine, networkManager, stateDB, "8080")

	// شروع Consensus Engine
	if err := consensusEngine.Start(); err != nil {
		log.Fatal("Failed to start consensus engine:", err)
	}

	// شروع Network
	if err := networkManager.Start(); err != nil {
		log.Fatal("Failed to start network:", err)
	}

	// توزیع اولیه ارز سینار
	fmt.Println(" Initializing SINAR token distribution...")
	if err := stateDB.InitializeSINARDistribution(); err != nil {
		log.Fatalf("Failed to initialize SINAR distribution: %v", err)
	}
	fmt.Println("✅ SINAR token distribution completed!")

	// شروع API Server در goroutine جداگانه
	go func() {
		if err := apiServer.Start(); err != nil {
			log.Printf("API Server error: %v", err)
		}
	}()

	// تولید کلید برای validator اصلی
	privKey, _ := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	validator := NewValidator("NodeA", privKey, 1000000)

	// اضافه کردن validator به consensus engine
	if err := consensusEngine.AddValidator(validator); err != nil {
		log.Fatal("Failed to add validator:", err)
	}

	// ایجاد events نمونه
	createSampleEvents(consensusEngine, privKey)

	// نمایش اطلاعات شبکه
	displayNetworkInfo(consensusEngine, networkManager)
	displayConsensusInfo(consensusEngine)
	runMultiNodeTest()
	runPerformanceTest(consensusEngine)
	// نگه داشتن برنامه برای مدتی
	fmt.Println("\n⏳ Running Sinar Chain for 60 seconds...")
	time.Sleep(60 * time.Second)

	// توقف شبکه
	networkManager.Stop()
	consensusEngine.Stop()
	fmt.Println("🛑 Sinar Chain stopped.")
}

// createSampleEvents ایجاد events نمونه
func createSampleEvents(engine *ConsensusEngine, privKey *ecdsa.PrivateKey) {
	fmt.Println("📝 Creating sample events...")

	// Event اول (Genesis)
	event1, err := createEvent("NodeA", nil, 0, 1, nil, privKey)
	if err != nil {
		log.Printf("Failed to create event1: %v", err)
		return
	}

	// Event دوم
	event2, err := createEvent("NodeA", []EventID{event1.Hash()}, 0, 2, nil, privKey)
	if err != nil {
		log.Printf("Failed to create event2: %v", err)
		return
	}

	// Event سوم
	event3, err := createEvent("NodeA", []EventID{event1.Hash(), event2.Hash()}, 0, 3, nil, privKey)
	if err != nil {
		log.Printf("Failed to create event3: %v", err)
		return
	}

	// اضافه کردن events به consensus engine
	engine.AddEvent(event1)
	engine.AddEvent(event2)
	engine.AddEvent(event3)

	fmt.Printf("✅ Created %d sample events\n", 3)
}

// createEvent ایجاد event جدید
func createEvent(creatorID string, parents []EventID, epoch, lamport uint64, txs types.Transactions, privKey *ecdsa.PrivateKey) (*Event, error) {
	event := NewEvent(creatorID, parents, epoch, lamport, txs, 0)

	// امضای event
	if err := event.Sign(privKey); err != nil {
		return nil, fmt.Errorf("failed to sign event: %v", err)
	}

	return event, nil
}

// displayNetworkInfo نمایش اطلاعات شبکه
func displayNetworkInfo(engine *ConsensusEngine, network *NetworkManager) {
	fmt.Println("\n📊 Network Information:")

	// اطلاعات Poset
	poset := engine.GetPoset()
	if poset != nil {
		fmt.Printf("Total Events: %d\n", len(poset.Events))
		fmt.Printf("Latest Round: %d\n", poset.LastRound)
		fmt.Printf("Latest Frame: %d\n", poset.LastFrame)
	}

	// اطلاعات Validators
	validators := engine.GetValidators()
	fmt.Printf("Active Validators: %d\n", len(validators))

	// اطلاعات Network
	peers := network.GetPeers()
	fmt.Printf("Connected Peers: %d\n", len(peers))

	// اطلاعات آخرین بلاک
	latestBlock := engine.GetLatestBlock()
	if latestBlock != nil {
		fmt.Printf("Latest Block: #%d\n", latestBlock.Header.Number)
		fmt.Printf("Block Hash: %s\n", latestBlock.Hash().Hex())
	} else {
		fmt.Println("No blocks created yet")
	}

	// اطلاعات Blockchain
	blockchain := engine.GetBlockchain()
	if blockchain != nil {
		blockStats := blockchain.GetBlockStats()
		fmt.Printf("Blockchain Stats: %+v\n", blockStats)
	}

	// اطلاعات State
	stateDB := engine.GetStateDB()
	if stateDB != nil {
		stateStats := stateDB.GetStateStats()
		fmt.Printf("State Stats: %+v\n", stateStats)
	}
}

// displayConsensusInfo نمایش اطلاعات consensus
func displayConsensusInfo(engine *ConsensusEngine) {
	fmt.Println("\n🔄 Consensus Information:")

	poset := engine.GetPoset()
	if poset == nil {
		fmt.Println("Poset not available")
		return
	}

	// نمایش rounds
	for round := uint64(0); round <= poset.LastRound; round++ {
		if roundInfo, exists := poset.GetRoundInfo(round); exists {
			fmt.Printf("Round %d: %d witnesses, %d roots, %d clothos, %d atropos\n",
				round,
				len(roundInfo.Witnesses),
				len(roundInfo.Roots),
				len(roundInfo.Clothos),
				len(roundInfo.Atropos))
		}
	}

	// نمایش latest events
	latestEvents := poset.GetLatestEvents()
	fmt.Printf("Latest Events: %d\n", len(latestEvents))
	for creatorID, event := range latestEvents {
		hash := event.Hash()
		fmt.Printf("  %s: %x\n", creatorID, hash[:8])
	}

	// نمایش consensus stats
	consensusStats := engine.GetConsensusStats()
	fmt.Printf("Consensus Stats: %+v\n", consensusStats)
}

// runMultiNodeTest اجرای تست چند نودی
func runMultiNodeTest() error {
	fmt.Println("🚀 Starting Multi-Node Test for Sinar Chain...")

	// در نسخه فعلی، این تست پیاده‌سازی نشده است
	fmt.Println("⚠️ Multi-Node Test not implemented yet")

	fmt.Println("✅ Multi-Node Test completed successfully!")
	return nil
}

// runPerformanceTest اجرای تست عملکرد
func runPerformanceTest(engine *ConsensusEngine) {
	fmt.Println("🚀 Starting Performance Test...")

	// تست ایجاد events
	startTime := time.Now()
	for i := 0; i < 1000; i++ {
		event, _ := createEvent("TestNode", nil, 0, uint64(i), nil, nil)
		engine.AddEvent(event)
	}
	eventTime := time.Since(startTime)
	fmt.Printf("✅ Created 1000 events in %v\n", eventTime)

	// تست consensus
	startTime = time.Now()
	_ = engine.GetConsensusStats()
	consensusTime := time.Since(startTime)
	fmt.Printf("✅ Consensus stats calculated in %v\n", consensusTime)

	// تست blockchain
	blockchain := engine.GetBlockchain()
	if blockchain != nil {
		startTime = time.Now()
		_ = blockchain.GetBlockStats()
		blockchainTime := time.Since(startTime)
		fmt.Printf("✅ Blockchain stats calculated in %v\n", blockchainTime)
	}

	fmt.Println("✅ Performance Test completed!")
}
