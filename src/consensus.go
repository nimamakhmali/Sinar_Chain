package main

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"golang.org/x/crypto/sha3"
)

// CacheManager مدیریت cache برای بهبود عملکرد
type CacheManager struct {
	eventCache     map[EventID]*Event
	voteCache      map[string]*Vote
	consensusCache map[uint64]map[string]interface{}
	stateCache     map[string]*big.Int
	mu             sync.RWMutex
	maxCacheSize   int
	cacheHits      int64
	cacheMisses    int64
}

// NewCacheManager ایجاد CacheManager جدید
func NewCacheManager(maxSize int) *CacheManager {
	return &CacheManager{
		eventCache:     make(map[EventID]*Event),
		voteCache:      make(map[string]*Vote),
		consensusCache: make(map[uint64]map[string]interface{}),
		stateCache:     make(map[string]*big.Int),
		maxCacheSize:   maxSize,
	}
}

// GetEvent از cache دریافت event
func (cm *CacheManager) GetEvent(eventID EventID) (*Event, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	event, exists := cm.eventCache[eventID]
	if exists {
		atomic.AddInt64(&cm.cacheHits, 1)
		return event, true
	}

	atomic.AddInt64(&cm.cacheMisses, 1)
	return nil, false
}

// SetEvent ذخیره event در cache
func (cm *CacheManager) SetEvent(eventID EventID, event *Event) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// بررسی اندازه cache
	if len(cm.eventCache) >= cm.maxCacheSize {
		cm.evictOldest()
	}

	cm.eventCache[eventID] = event
}

// GetVote از cache دریافت vote
func (cm *CacheManager) GetVote(key string) (*Vote, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	vote, exists := cm.voteCache[key]
	if exists {
		atomic.AddInt64(&cm.cacheHits, 1)
		return vote, true
	}

	atomic.AddInt64(&cm.cacheMisses, 1)
	return nil, false
}

// SetVote ذخیره vote در cache
func (cm *CacheManager) SetVote(key string, vote *Vote) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// بررسی اندازه cache
	if len(cm.voteCache) >= cm.maxCacheSize {
		cm.evictOldest()
	}

	cm.voteCache[key] = vote
}

// GetConsensusCache از cache دریافت consensus data
func (cm *CacheManager) GetConsensusCache(round uint64) (map[string]interface{}, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	data, exists := cm.consensusCache[round]
	if exists {
		atomic.AddInt64(&cm.cacheHits, 1)
		return data, true
	}

	atomic.AddInt64(&cm.cacheMisses, 1)
	return nil, false
}

// SetConsensusCache ذخیره consensus data در cache
func (cm *CacheManager) SetConsensusCache(round uint64, data map[string]interface{}) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// بررسی اندازه cache
	if len(cm.consensusCache) >= cm.maxCacheSize {
		cm.evictOldest()
	}

	cm.consensusCache[round] = data
}

// GetStateCache از cache دریافت state data
func (cm *CacheManager) GetStateCache(key string) (*big.Int, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	value, exists := cm.stateCache[key]
	if exists {
		atomic.AddInt64(&cm.cacheHits, 1)
		return value, true
	}

	atomic.AddInt64(&cm.cacheMisses, 1)
	return nil, false
}

// SetStateCache ذخیره state data در cache
func (cm *CacheManager) SetStateCache(key string, value *big.Int) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// بررسی اندازه cache
	if len(cm.stateCache) >= cm.maxCacheSize {
		cm.evictOldest()
	}

	cm.stateCache[key] = value
}

// evictOldest حذف قدیمی‌ترین entries از cache
func (cm *CacheManager) evictOldest() {
	// حذف 20% از قدیمی‌ترین entries
	evictCount := cm.maxCacheSize / 5

	// حذف از event cache
	if len(cm.eventCache) > evictCount {
		keys := make([]EventID, 0, len(cm.eventCache))
		for k := range cm.eventCache {
			keys = append(keys, k)
		}
		for i := 0; i < evictCount && i < len(keys); i++ {
			delete(cm.eventCache, keys[i])
		}
	}

	// حذف از vote cache
	if len(cm.voteCache) > evictCount {
		keys := make([]string, 0, len(cm.voteCache))
		for k := range cm.voteCache {
			keys = append(keys, k)
		}
		for i := 0; i < evictCount && i < len(keys); i++ {
			delete(cm.voteCache, keys[i])
		}
	}

	// حذف از consensus cache
	if len(cm.consensusCache) > evictCount {
		keys := make([]uint64, 0, len(cm.consensusCache))
		for k := range cm.consensusCache {
			keys = append(keys, k)
		}
		for i := 0; i < evictCount && i < len(keys); i++ {
			delete(cm.consensusCache, keys[i])
		}
	}

	// حذف از state cache
	if len(cm.stateCache) > evictCount {
		keys := make([]string, 0, len(cm.stateCache))
		for k := range cm.stateCache {
			keys = append(keys, k)
		}
		for i := 0; i < evictCount && i < len(keys); i++ {
			delete(cm.stateCache, keys[i])
		}
	}
}

// GetCacheStats آمار cache
func (cm *CacheManager) GetCacheStats() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	hits := atomic.LoadInt64(&cm.cacheHits)
	misses := atomic.LoadInt64(&cm.cacheMisses)
	total := hits + misses

	var hitRate float64
	if total > 0 {
		hitRate = float64(hits) / float64(total)
	}

	return map[string]interface{}{
		"event_cache_size":     len(cm.eventCache),
		"vote_cache_size":      len(cm.voteCache),
		"consensus_cache_size": len(cm.consensusCache),
		"state_cache_size":     len(cm.stateCache),
		"cache_hits":           hits,
		"cache_misses":         misses,
		"hit_rate":             hitRate,
		"max_cache_size":       cm.maxCacheSize,
	}
}

// ClearCache پاک کردن تمام cache
func (cm *CacheManager) ClearCache() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.eventCache = make(map[EventID]*Event)
	cm.voteCache = make(map[string]*Vote)
	cm.consensusCache = make(map[uint64]map[string]interface{})
	cm.stateCache = make(map[string]*big.Int)

	atomic.StoreInt64(&cm.cacheHits, 0)
	atomic.StoreInt64(&cm.cacheMisses, 0)
}

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

	// Blockchain integration
	blockchain *Blockchain
	stateDB    *StateDB

	// Performance optimization
	cacheManager      *CacheManager
	parallelProcessor *ParallelProcessor
	memoryOptimizer   *MemoryOptimizer
}

type ConsensusConfig struct {
	BlockTime         time.Duration
	MaxEventsPerBlock int
	MinValidators     int
	MaxValidators     int
	StakeRequired     uint64
	ConsensusTimeout  time.Duration
	GasLimit          uint64
	BaseFee           *big.Int
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

	// ایجاد blockchain
	blockchain := NewBlockchain(dag)
	stateDB := NewStateDB()

	// ایجاد cache manager برای بهبود عملکرد
	cacheManager := NewCacheManager(10000) // 10K entries

	// ایجاد parallel processor برای پردازش موازی
	parallelProcessor := NewParallelProcessor(4, 2) // 4 event workers, 2 consensus workers

	// ایجاد memory optimizer برای بهینه‌سازی حافظه
	memoryOptimizer := NewMemoryOptimizer()

	ctx, cancel := context.WithCancel(context.Background())
	return &ConsensusEngine{
		poset:             poset,
		validators:        make(map[common.Address]*Validator),
		validatorSet:      NewValidatorSet(),
		blockTime:         config.BlockTime,
		eventQueue:        make(chan *Event, 1000),
		blockQueue:        make(chan *Block, 100),
		config:            config,
		ctx:               ctx,
		cancel:            cancel,
		roundAssignment:   roundAssignment,
		fameVoting:        fameVoting,
		clothoSelector:    clothoSelector,
		finalityEngine:    finalityEngine,
		blockchain:        blockchain,
		stateDB:           stateDB,
		cacheManager:      cacheManager,
		parallelProcessor: parallelProcessor,
		memoryOptimizer:   memoryOptimizer,
	}
}

// Start شروع Consensus Engine
func (ce *ConsensusEngine) Start() error {
	// شروع parallel processor
	ce.parallelProcessor.Start()

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

	// توقف parallel processor
	if ce.parallelProcessor != nil {
		ce.parallelProcessor.Stop()
	}

	close(ce.eventQueue)
	close(ce.blockQueue)
}

// AddEvent اضافه کردن event
func (ce *ConsensusEngine) AddEvent(event *Event) error {
	// ارسال event به parallel processor
	ce.parallelProcessor.AddEvent(event)

	// همچنین ارسال به queue اصلی
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
		// بررسی cache برای event
		eventID := event.Hash()
		if cachedEvent, exists := ce.cacheManager.GetEvent(eventID); exists {
			// اگر event در cache موجود است، از آن استفاده کن
			event = cachedEvent
		} else {
			// ذخیره event در cache
			ce.cacheManager.SetEvent(eventID, event)
		}

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
		GasLimit:    ce.config.GasLimit,
		GasUsed:     0,
		BaseFee:     ce.config.BaseFee,
	}

	// محاسبه state root
	header.Root = ce.calculateStateRoot(transactions)

	// ایجاد بلاک با استفاده از memory optimizer
	block := ce.memoryOptimizer.GetBlock()
	block.Header = header
	block.Transactions = transactions
	block.Events = eventIDs

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

	// ایجاد consensus task برای fame voting
	task := &ConsensusTask{
		Round:    ce.currentRound,
		Events:   ce.getEventsForRound(ce.currentRound),
		Type:     "fame_voting",
		Priority: 1,
	}

	// ارسال task به parallel processor
	ce.parallelProcessor.AddConsensusTask(task)

	// اجرای fame voting برای تمام rounds (همچنان برای backward compatibility)
	ce.fameVoting.StartFameVoting()
}

// runClothoSelection اجرای clotho selection (exactly like Fantom)
func (ce *ConsensusEngine) runClothoSelection() {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	// ایجاد consensus task برای clotho selection
	task := &ConsensusTask{
		Round:    ce.currentRound,
		Events:   ce.getEventsForRound(ce.currentRound),
		Type:     "clotho_selection",
		Priority: 2,
	}

	// ارسال task به parallel processor
	ce.parallelProcessor.AddConsensusTask(task)

	// انتخاب Clothos برای تمام rounds (همچنان برای backward compatibility)
	ce.clothoSelector.SelectClothos()
}

// runFinality اجرای finality (exactly like Fantom)
func (ce *ConsensusEngine) runFinality() {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	// ایجاد consensus task برای finality
	task := &ConsensusTask{
		Round:    ce.currentRound,
		Events:   ce.getEventsForRound(ce.currentRound),
		Type:     "finality",
		Priority: 3,
	}

	// ارسال task به parallel processor
	ce.parallelProcessor.AddConsensusTask(task)

	// نهایی‌سازی events و تبدیل Clothos به Atropos (همچنان برای backward compatibility)
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

	// آمار Blockchain
	blockStats := ce.blockchain.GetBlockStats()
	stats["blockchain_stats"] = blockStats

	// آمار State
	stateStats := ce.stateDB.GetStateStats()
	stats["state_stats"] = stateStats

	// آمار Cache
	if ce.cacheManager != nil {
		cacheStats := ce.cacheManager.GetCacheStats()
		stats["cache_stats"] = cacheStats
	}

	// آمار Memory Optimization
	if ce.memoryOptimizer != nil {
		memoryStats := ce.memoryOptimizer.GetMemoryStats()
		stats["memory_stats"] = memoryStats
	}

	// آمار Parallel Processing
	if ce.parallelProcessor != nil {
		parallelStats := ce.parallelProcessor.GetStats()
		stats["parallel_stats"] = parallelStats
	}

	return stats
}

// GetBlockchain دریافت Blockchain
func (ce *ConsensusEngine) GetBlockchain() *Blockchain {
	return ce.blockchain
}

// GetStateDB دریافت StateDB
func (ce *ConsensusEngine) GetStateDB() *StateDB {
	return ce.stateDB
}

// getEventsForRound دریافت events برای یک round خاص
func (ce *ConsensusEngine) getEventsForRound(round uint64) []*Event {
	var events []*Event
	for _, event := range ce.poset.Events {
		if event.Round == round {
			events = append(events, event)
		}
	}
	return events
}

// ParallelProcessor پردازش موازی events و consensus
type ParallelProcessor struct {
	eventWorkers     int
	consensusWorkers int
	eventQueue       chan *Event
	consensusQueue   chan *ConsensusTask
	results          chan *ProcessingResult
	mu               sync.RWMutex
	running          bool
}

// ConsensusTask وظیفه consensus
type ConsensusTask struct {
	Round    uint64
	Events   []*Event
	Type     string // "fame_voting", "clotho_selection", "finality"
	Priority int
}

// ProcessingResult نتیجه پردازش
type ProcessingResult struct {
	TaskID   string
	Success  bool
	Data     interface{}
	Error    error
	Duration time.Duration
}

// NewParallelProcessor ایجاد ParallelProcessor جدید
func NewParallelProcessor(eventWorkers, consensusWorkers int) *ParallelProcessor {
	return &ParallelProcessor{
		eventWorkers:     eventWorkers,
		consensusWorkers: consensusWorkers,
		eventQueue:       make(chan *Event, 1000),
		consensusQueue:   make(chan *ConsensusTask, 100),
		results:          make(chan *ProcessingResult, 100),
	}
}

// Start شروع پردازش موازی
func (pp *ParallelProcessor) Start() {
	pp.mu.Lock()
	defer pp.mu.Unlock()

	if pp.running {
		return
	}

	pp.running = true

	// شروع event workers
	for i := 0; i < pp.eventWorkers; i++ {
		go pp.eventWorker(i)
	}

	// شروع consensus workers
	for i := 0; i < pp.consensusWorkers; i++ {
		go pp.consensusWorker(i)
	}

	// شروع result processor
	go pp.resultProcessor()
}

// Stop توقف پردازش موازی
func (pp *ParallelProcessor) Stop() {
	pp.mu.Lock()
	defer pp.mu.Unlock()

	if !pp.running {
		return
	}

	pp.running = false
	close(pp.eventQueue)
	close(pp.consensusQueue)
	close(pp.results)
}

// AddEvent اضافه کردن event برای پردازش موازی
func (pp *ParallelProcessor) AddEvent(event *Event) {
	select {
	case pp.eventQueue <- event:
	default:
		// Queue full, drop event
	}
}

// AddConsensusTask اضافه کردن consensus task برای پردازش موازی
func (pp *ParallelProcessor) AddConsensusTask(task *ConsensusTask) {
	select {
	case pp.consensusQueue <- task:
	default:
		// Queue full, drop task
	}
}

// eventWorker worker برای پردازش events
func (pp *ParallelProcessor) eventWorker(id int) {
	for event := range pp.eventQueue {
		if !pp.running {
			break
		}

		start := time.Now()

		// پردازش event
		result := pp.processEvent(event)

		// ارسال نتیجه
		pp.results <- &ProcessingResult{
			TaskID:   fmt.Sprintf("event_%d", id),
			Success:  result.Success,
			Data:     result.Data,
			Error:    result.Error,
			Duration: time.Since(start),
		}
	}
}

// consensusWorker worker برای پردازش consensus
func (pp *ParallelProcessor) consensusWorker(id int) {
	for task := range pp.consensusQueue {
		if !pp.running {
			break
		}

		start := time.Now()

		// پردازش consensus task
		result := pp.processConsensusTask(task)

		// ارسال نتیجه
		pp.results <- &ProcessingResult{
			TaskID:   fmt.Sprintf("consensus_%d_%s", id, task.Type),
			Success:  result.Success,
			Data:     result.Data,
			Error:    result.Error,
			Duration: time.Since(start),
		}
	}
}

// resultProcessor پردازش نتایج
func (pp *ParallelProcessor) resultProcessor() {
	for result := range pp.results {
		if !pp.running {
			break
		}

		// پردازش نتیجه
		pp.handleResult(result)
	}
}

// processEvent پردازش یک event
func (pp *ParallelProcessor) processEvent(event *Event) *ProcessingResult {
	start := time.Now()

	// پردازش event در parallel processor
	// اینجا می‌توانیم event را به consensus engine ارسال کنیم
	result := &ProcessingResult{
		TaskID:   "event_" + fmt.Sprintf("%x", event.Hash()),
		Success:  true,
		Data:     event,
		Error:    nil,
		Duration: time.Since(start),
	}

	// در نسخه کامل، اینجا event را به consensus engine ارسال می‌کنیم
	// برای مثال: ce.AddEvent(event)

	return result
}

// processConsensusTask پردازش یک consensus task
func (pp *ParallelProcessor) processConsensusTask(task *ConsensusTask) *ProcessingResult {
	switch task.Type {
	case "fame_voting":
		return pp.processFameVoting(task)
	case "clotho_selection":
		return pp.processClothoSelection(task)
	case "finality":
		return pp.processFinality(task)
	default:
		return &ProcessingResult{
			Success: false,
			Error:   fmt.Errorf("unknown task type: %s", task.Type),
		}
	}
}

// processFameVoting پردازش fame voting
func (pp *ParallelProcessor) processFameVoting(task *ConsensusTask) *ProcessingResult {
	start := time.Now()

	// پردازش fame voting برای events
	// در نسخه کامل، این با FameVoting component ادغام می‌شود
	result := &ProcessingResult{
		TaskID:   fmt.Sprintf("fame_voting_%d", task.Round),
		Success:  true,
		Data:     map[string]interface{}{"round": task.Round, "type": "fame_voting", "events_count": len(task.Events)},
		Error:    nil,
		Duration: time.Since(start),
	}

	// در نسخه کامل، اینجا FameVoting component را فراخوانی می‌کنیم
	// برای مثال: fameVoting.ProcessFameVoting(task.Round, task.Events)

	return result
}

// processClothoSelection پردازش clotho selection
func (pp *ParallelProcessor) processClothoSelection(task *ConsensusTask) *ProcessingResult {
	start := time.Now()

	// پردازش clotho selection برای events
	// در نسخه کامل، این با ClothoSelector component ادغام می‌شود
	result := &ProcessingResult{
		TaskID:   fmt.Sprintf("clotho_selection_%d", task.Round),
		Success:  true,
		Data:     map[string]interface{}{"round": task.Round, "type": "clotho_selection", "events_count": len(task.Events)},
		Error:    nil,
		Duration: time.Since(start),
	}

	// در نسخه کامل، اینجا ClothoSelector component را فراخوانی می‌کنیم
	// برای مثال: clothoSelector.SelectClothosForRound(task.Round, task.Events)

	return result
}

// processFinality پردازش finality
func (pp *ParallelProcessor) processFinality(task *ConsensusTask) *ProcessingResult {
	start := time.Now()

	// پردازش finality برای events
	// در نسخه کامل، این با FinalityEngine component ادغام می‌شود
	result := &ProcessingResult{
		TaskID:   fmt.Sprintf("finality_%d", task.Round),
		Success:  true,
		Data:     map[string]interface{}{"round": task.Round, "type": "finality", "events_count": len(task.Events)},
		Error:    nil,
		Duration: time.Since(start),
	}

	// در نسخه کامل، اینجا FinalityEngine component را فراخوانی می‌کنیم
	// برای مثال: finalityEngine.FinalizeEventsForRound(task.Round, task.Events)

	return result
}

// handleResult پردازش نتیجه
func (pp *ParallelProcessor) handleResult(result *ProcessingResult) {
	if result.Success {
		// پردازش نتیجه موفق
		fmt.Printf("✅ Parallel processing completed: %s (Duration: %v)\n",
			result.TaskID, result.Duration)
	} else {
		// پردازش خطا
		fmt.Printf("❌ Parallel processing failed: %s (Error: %v)\n",
			result.TaskID, result.Error)
	}
}

// GetStats آمار پردازش موازی
func (pp *ParallelProcessor) GetStats() map[string]interface{} {
	pp.mu.RLock()
	defer pp.mu.RUnlock()

	return map[string]interface{}{
		"event_workers":        pp.eventWorkers,
		"consensus_workers":    pp.consensusWorkers,
		"running":              pp.running,
		"event_queue_size":     len(pp.eventQueue),
		"consensus_queue_size": len(pp.consensusQueue),
		"results_queue_size":   len(pp.results),
	}
}
