package main

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"golang.org/x/crypto/sha3"
)

// Consensus Engine مشابه Fantom
type ConsensusEngine struct {
	mu sync.RWMutex

	// Poset
	poset *Poset

	// Validators
	validators   map[common.Address]*Validator
	validatorSet *ValidatorSet

	// Block creation
	blockTime time.Duration
	lastBlock time.Time

	// Consensus state
	currentRound uint64
	currentFrame uint64
	lastDecided  uint64

	// Event processing
	eventQueue chan *Event
	blockQueue chan *Block

	// Network
	network *NetworkManager

	// Configuration
	config *ConsensusConfig

	// Context for graceful shutdown
	ctx    context.Context
	cancel context.CancelFunc

	// Consensus Components (exactly like Fantom)
	roundAssignment *RoundAssignment
	fameVoting      *FameVoting
	clothoSelector  *ClothoSelector
	finalityEngine  *FinalityEngine
}

type ConsensusConfig struct {
	BlockTime         time.Duration
	MaxEventsPerBlock int
	MinValidators     int
	MaxValidators     int
	StakeRequired     uint64
	ConsensusTimeout  time.Duration
}

// NewConsensusEngine ایجاد Consensus Engine جدید
func NewConsensusEngine(config *ConsensusConfig) *ConsensusEngine {
	posetConfig := &PosetConfig{
		MaxEventsPerBlock: config.MaxEventsPerBlock,
		BlockTime:         config.BlockTime,
		MinValidators:     config.MinValidators,
		MaxValidators:     config.MaxValidators,
		StakeRequired:     config.StakeRequired,
	}

	poset := NewPoset(posetConfig)

	// ایجاد DAG برای consensus components
	dag := &DAG{
		Events: poset.Events,
		Rounds: poset.Rounds,
		Votes:  make(VoteRecord),
	}

	// ایجاد consensus components (exactly like Fantom)
	roundAssignment := NewRoundAssignment(dag)
	fameVoting := NewFameVoting(dag)
	clothoSelector := NewClothoSelector(dag)
	finalityEngine := NewFinalityEngine(dag)

	ctx, cancel := context.WithCancel(context.Background())
	return &ConsensusEngine{
		poset:           poset,
		validators:      make(map[common.Address]*Validator),
		validatorSet:    NewValidatorSet(),
		blockTime:       config.BlockTime,
		eventQueue:      make(chan *Event, 1000),
		blockQueue:      make(chan *Block, 100),
		config:          config,
		ctx:             ctx,
		cancel:          cancel,
		roundAssignment: roundAssignment,
		fameVoting:      fameVoting,
		clothoSelector:  clothoSelector,
		finalityEngine:  finalityEngine,
	}
}

// Start شروع Consensus Engine
func (ce *ConsensusEngine) Start() error {
	// شروع event processing
	go ce.processEvents()

	// شروع block creation
	go ce.createBlocks()

	// شروع consensus loop
	go ce.consensusLoop()

	return nil
}

// Stop توقف Consensus Engine
func (ce *ConsensusEngine) Stop() {
	// Graceful shutdown
	if ce.cancel != nil {
		ce.cancel()
	}
	close(ce.eventQueue)
	close(ce.blockQueue)
}

// AddEvent اضافه کردن event
func (ce *ConsensusEngine) AddEvent(event *Event) error {
	select {
	case ce.eventQueue <- event:
		return nil
	default:
		return fmt.Errorf("event queue full")
	}
}

// processEvents پردازش events
func (ce *ConsensusEngine) processEvents() {
	for event := range ce.eventQueue {
		// بررسی اینکه آیا event قبلاً وجود دارد
		if _, exists := ce.poset.Events[event.Hash()]; exists {
			// Event قبلاً وجود دارد، نادیده گرفتن
			continue
		}

		if err := ce.poset.AddEvent(event); err != nil {
			fmt.Printf("Failed to add event: %v\n", err)
			continue
		}

		// Gossip event
		if ce.network != nil {
			ce.network.GossipEvent(event)
		}
	}
}

// createBlocks ایجاد بلاک‌ها
func (ce *ConsensusEngine) createBlocks() {
	ticker := time.NewTicker(ce.blockTime)
	defer ticker.Stop()

	for range ticker.C {
		ce.createBlock()
	}
}

// createBlock ایجاد یک بلاک
func (ce *ConsensusEngine) createBlock() {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	// بررسی زمان مناسب
	if time.Since(ce.lastBlock) < ce.blockTime {
		return
	}

	// دریافت events نهایی شده
	finalizedEvents := ce.finalityEngine.GetFinalizedEvents()
	if len(finalizedEvents) == 0 {
		return
	}

	// گروه‌بندی events بر اساس زمان
	eventGroups := ce.groupEventsByTime(finalizedEvents)

	for _, events := range eventGroups {
		// ایجاد بلاک از این گروه events
		block := ce.createBlockFromEvents(events)
		if block == nil {
			continue
		}

		// اضافه کردن به queue
		select {
		case ce.blockQueue <- block:
			ce.lastBlock = time.Now()
			fmt.Printf("📦 Block %d created with %d events\n", block.Header.Number, len(events))
		default:
			fmt.Println("Block queue full")
		}
	}
}

// groupEventsByTime گروه‌بندی events بر اساس زمان
func (ce *ConsensusEngine) groupEventsByTime(events []*Event) [][]*Event {
	if len(events) == 0 {
		return nil
	}

	// گروه‌بندی بر اساس AtroposTime
	groups := make(map[uint64][]*Event)
	timeWindow := uint64(2000) // 2 seconds in milliseconds

	for _, event := range events {
		if event.Atropos == (EventID{}) {
			continue // فقط events نهایی شده
		}
		timeGroup := event.AtroposTime / timeWindow
		groups[timeGroup] = append(groups[timeGroup], event)
	}

	result := make([][]*Event, 0, len(groups))
	for _, group := range groups {
		if len(group) > 0 {
			result = append(result, group)
		}
	}

	return result
}

// createBlockFromEvents ایجاد بلاک از events
func (ce *ConsensusEngine) createBlockFromEvents(events []*Event) *Block {
	if len(events) == 0 {
		return nil
	}

	// تبدیل events به transactions
	var transactions types.Transactions
	eventIDs := make([]EventID, len(events))

	for i, event := range events {
		transactions = append(transactions, event.Transactions...)
		eventIDs[i] = event.Hash()
	}

	// محاسبه median time
	times := make([]uint64, 0, len(events))
	for _, event := range events {
		if event.AtroposTime > 0 {
			times = append(times, event.AtroposTime)
		}
	}

	var medianTime uint64
	if len(times) > 0 {
		medianTime = ce.calculateMedian(times)
	}

	// ایجاد block header
	latestBlock := ce.poset.GetLatestBlock()
	var parentHash common.Hash
	var blockNumber uint64

	if latestBlock != nil {
		parentHash = latestBlock.Hash()
		blockNumber = latestBlock.Header.Number + 1
	} else {
		parentHash = common.Hash{}
		blockNumber = 1
	}

	header := BlockHeader{
		Number:      blockNumber,
		ParentHash:  parentHash,
		AtroposTime: medianTime,
		CreatedAt:   time.Now(),
		Creator:     "SinarChain", // در نسخه کامل از validator استفاده می‌شود
	}

	// محاسبه state root
	header.Root = ce.calculateStateRoot(transactions)

	// ایجاد بلاک
	block := &Block{
		Header:       header,
		Transactions: transactions,
		Events:       eventIDs,
	}

	return block
}

// calculateMedian محاسبه median
func (ce *ConsensusEngine) calculateMedian(values []uint64) uint64 {
	if len(values) == 0 {
		return 0
	}

	// کپی کردن slice برای مرتب‌سازی
	sorted := make([]uint64, len(values))
	copy(sorted, values)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	// محاسبه median
	n := len(sorted)
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2
	}
	return sorted[n/2]
}

// calculateStateRoot محاسبه state root
func (ce *ConsensusEngine) calculateStateRoot(txs types.Transactions) common.Hash {
	// در نسخه کامل، این با EVM ادغام می‌شود
	hasher := sha3.NewLegacyKeccak256()
	for _, tx := range txs {
		hasher.Write(tx.Hash().Bytes())
	}
	var hash common.Hash
	copy(hash[:], hasher.Sum(nil))
	return hash
}

// getFinalizedEvents دریافت events نهایی شده
func (ce *ConsensusEngine) getFinalizedEvents() []*Event {
	return ce.finalityEngine.GetFinalizedEvents()
}

// consensusLoop حلقه consensus (exactly like Fantom)
func (ce *ConsensusEngine) consensusLoop() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		ce.runConsensus()
	}
}

// runConsensus اجرای consensus (exactly like Fantom)
func (ce *ConsensusEngine) runConsensus() {
	ticker := time.NewTicker(100 * time.Millisecond) // هر 100ms مثل Fantom
	defer ticker.Stop()

	for {
		select {
		case <-ce.ctx.Done():
			return
		case <-ticker.C:
			// اجرای مراحل consensus مثل Fantom
			ce.runRoundAssignment()
			ce.runFameVoting()
			ce.runClothoSelection()
			ce.runFinality()
			ce.createBlocks()
		}
	}
}

// runRoundAssignment اجرای round assignment (exactly like Fantom)
func (ce *ConsensusEngine) runRoundAssignment() {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	// برای تمام events جدید
	for _, event := range ce.poset.Events {
		if event.Round == 0 && !event.IsRoot { // هنوز round محاسبه نشده
			ce.roundAssignment.AssignRound(event)
		}
	}
}

// runFameVoting اجرای fame voting (exactly like Fantom)
func (ce *ConsensusEngine) runFameVoting() {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	// اجرای fame voting برای تمام rounds
	ce.fameVoting.StartFameVoting()
}

// runClothoSelection اجرای clotho selection (exactly like Fantom)
func (ce *ConsensusEngine) runClothoSelection() {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	// انتخاب Clothos برای تمام rounds
	ce.clothoSelector.SelectClothos()
}

// runFinality اجرای finality (exactly like Fantom)
func (ce *ConsensusEngine) runFinality() {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	// نهایی‌سازی events و تبدیل Clothos به Atropos
	ce.finalityEngine.FinalizeEvents()
}

// AddValidator اضافه کردن validator
func (ce *ConsensusEngine) AddValidator(validator *Validator) error {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	// تبدیل string address به common.Address
	address := common.HexToAddress(validator.Address)
	ce.validators[address] = validator
	ce.validatorSet.AddValidator(validator)

	return nil
}

// RemoveValidator حذف validator
func (ce *ConsensusEngine) RemoveValidator(address common.Address) error {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	delete(ce.validators, address)
	// تبدیل common.Address به string برای RemoveValidator
	ce.validatorSet.RemoveValidator(address.Hex())

	return nil
}

// GetValidators دریافت validators
func (ce *ConsensusEngine) GetValidators() map[common.Address]*Validator {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	result := make(map[common.Address]*Validator)
	for addr, validator := range ce.validators {
		result[addr] = validator
	}

	return result
}

// GetLatestBlock دریافت آخرین بلاک
func (ce *ConsensusEngine) GetLatestBlock() *Block {
	return ce.poset.GetLatestBlock()
}

// GetPoset دریافت Poset
func (ce *ConsensusEngine) GetPoset() *Poset {
	return ce.poset
}

// GetValidatorSet دریافت ValidatorSet
func (ce *ConsensusEngine) GetValidatorSet() *ValidatorSet {
	return ce.validatorSet
}

// SetNetwork تنظیم network
func (ce *ConsensusEngine) SetNetwork(network *NetworkManager) {
	ce.network = network
	ce.poset.Network = network
}

// GetConsensusStats آمار consensus
func (ce *ConsensusEngine) GetConsensusStats() map[string]interface{} {
	stats := make(map[string]interface{})

	// آمار events
	stats["total_events"] = len(ce.poset.Events)
	stats["total_rounds"] = len(ce.poset.Rounds)

	// آمار fame voting
	trueVotes, falseVotes, totalVotes := ce.fameVoting.GetVoteStats(EventID{}, 0)
	stats["fame_voting_stats"] = map[string]int{
		"true_votes":  trueVotes,
		"false_votes": falseVotes,
		"total_votes": totalVotes,
	}

	// آمار Clothos
	clothoStats := ce.clothoSelector.GetClothosStats()
	stats["clotho_stats"] = clothoStats

	// آمار Finality
	finalityStats := ce.finalityEngine.GetFinalityStats()
	stats["finality_stats"] = finalityStats

	return stats
}
