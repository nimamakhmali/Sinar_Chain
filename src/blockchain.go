package main

import (
	"crypto/ecdsa"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"golang.org/x/crypto/sha3"
)

// Block بلاک نهایی شده
type Block struct {
	Header       BlockHeader
	Transactions types.Transactions
	Events       []EventID
	hash         common.Hash
}

type BlockHeader struct {
	Number      uint64
	ParentHash  common.Hash
	Root        common.Hash
	AtroposTime uint64
	CreatedAt   time.Time
	Creator     string
	Signature   []byte
}

type Blockchain struct {
	blocks       map[common.Hash]*Block
	latest       common.Hash
	mu           sync.RWMutex
	dag          *DAG
	evmProcessor *EVMProcessor
	currentState *state.StateDB
}

func NewBlockchain(dag *DAG) *Blockchain {
	return &Blockchain{
		blocks:       make(map[common.Hash]*Block),
		dag:          dag,
		evmProcessor: NewEVMProcessor(),
		currentState: nil, // Will be initialized when first block is processed
	}
}

// CreateBlock ایجاد بلاک جدید از Atropos events
func (bc *Blockchain) CreateBlock(atroposEvents []*Event, validator *Validator) (*Block, error) {
	if len(atroposEvents) == 0 {
		return nil, fmt.Errorf("no atropos events to create block")
	}

	// جمع‌آوری تمام تراکنش‌ها
	var allTxs types.Transactions
	eventIDs := make([]EventID, 0, len(atroposEvents))

	for _, event := range atroposEvents {
		allTxs = append(allTxs, event.Transactions...)
		eventIDs = append(eventIDs, event.Hash())
	}

	// محاسبه median time
	times := make([]uint64, 0, len(atroposEvents))
	for _, event := range atroposEvents {
		times = append(times, event.AtroposTime)
	}
	medianTime := calculateMedian(times)

	// ایجاد header
	header := BlockHeader{
		Number:      bc.GetLatestBlockNumber() + 1,
		ParentHash:  bc.latest,
		AtroposTime: medianTime,
		CreatedAt:   time.Now(),
		Creator:     validator.ID,
	}

	// محاسبه state root (در نسخه کامل باید با EVM ادغام شود)
	header.Root = bc.calculateStateRoot(allTxs)

	block := &Block{
		Header:       header,
		Transactions: allTxs,
		Events:       eventIDs,
	}

	// امضای بلاک
	if err := block.Sign(validator.PrivateKey); err != nil {
		return nil, fmt.Errorf("failed to sign block: %v", err)
	}

	return block, nil
}

// calculateMedian محاسبه median از یک slice
func calculateMedian(values []uint64) uint64 {
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

func (bc *Blockchain) calculateStateRoot(txs types.Transactions) common.Hash {
	if bc.currentState == nil {
		// اگر state وجود ندارد، از hash تراکنش‌ها استفاده کن
		hasher := sha3.NewLegacyKeccak256()
		for _, tx := range txs {
			hasher.Write(tx.Hash().Bytes())
		}
		var hash common.Hash
		copy(hash[:], hasher.Sum(nil))
		return hash
	}

	// استفاده از state root واقعی
	return bc.currentState.IntermediateRoot(true)
}

func (b *Block) Hash() common.Hash {
	if b.hash != (common.Hash{}) {
		return b.hash
	}

	headerBytes, _ := rlp.EncodeToBytes(b.Header)
	txsBytes, _ := rlp.EncodeToBytes(b.Transactions)
	eventsBytes, _ := rlp.EncodeToBytes(b.Events)

	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(headerBytes)
	hasher.Write(txsBytes)
	hasher.Write(eventsBytes)

	copy(b.hash[:], hasher.Sum(nil))
	return b.hash
}

func (b *Block) Sign(priv *ecdsa.PrivateKey) error {
	data := crypto.Keccak256(b.Hash().Bytes())
	sig, err := crypto.Sign(data, priv)
	if err != nil {
		return err
	}
	b.Header.Signature = sig
	return nil
}

func (b *Block) VerifySignature(pub *ecdsa.PublicKey) bool {
	data := crypto.Keccak256(b.Hash().Bytes())
	recovered, err := crypto.SigToPub(data, b.Header.Signature)
	if err != nil {
		return false
	}
	return recovered.X.Cmp(pub.X) == 0 && recovered.Y.Cmp(pub.Y) == 0
}

func (bc *Blockchain) AddBlock(block *Block) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	// بررسی توالی
	if block.Header.Number != bc.GetLatestBlockNumber()+1 {
		return fmt.Errorf("invalid block number")
	}

	// بررسی parent hash
	if block.Header.ParentHash != bc.latest {
		return fmt.Errorf("invalid parent hash")
	}

	// پردازش بلاک با EVM
	newState, err := bc.evmProcessor.ProcessBlock(block, bc.currentState)
	if err != nil {
		return fmt.Errorf("failed to process block with EVM: %v", err)
	}

	// به‌روزرسانی state
	bc.currentState = newState

	// اضافه کردن بلاک
	blockHash := block.Hash()
	bc.blocks[blockHash] = block
	bc.latest = blockHash

	fmt.Printf("Block %d added: %s (State Root: %s)\n",
		block.Header.Number, blockHash.Hex(), bc.currentState.IntermediateRoot(true).Hex())
	return nil
}

func (bc *Blockchain) GetLatestBlockNumber() uint64 {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	if bc.latest == (common.Hash{}) {
		return 0
	}

	if block, exists := bc.blocks[bc.latest]; exists {
		return block.Header.Number
	}
	return 0
}

func (bc *Blockchain) GetBlock(hash common.Hash) (*Block, bool) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	block, exists := bc.blocks[hash]
	return block, exists
}

func (bc *Blockchain) GetLatestBlock() *Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	if bc.latest == (common.Hash{}) {
		return nil
	}

	return bc.blocks[bc.latest]
}

// ProcessAtroposEvents پردازش Atropos events و ایجاد بلاک
func (bc *Blockchain) ProcessAtroposEvents(validator *Validator) error {
	// پیدا کردن تمام Atropos events که هنوز بلاک نشده‌اند
	atroposEvents := make([]*Event, 0)

	bc.dag.mu.RLock()
	for _, event := range bc.dag.Events {
		if event.Atropos != (EventID{}) && !bc.isEventInBlock(event.Hash()) {
			atroposEvents = append(atroposEvents, event)
		}
	}
	bc.dag.mu.RUnlock()

	if len(atroposEvents) == 0 {
		return nil
	}

	// گروه‌بندی بر اساس AtroposTime
	blocks := bc.groupAtroposEventsByTime(atroposEvents)

	for _, events := range blocks {
		block, err := bc.CreateBlock(events, validator)
		if err != nil {
			return fmt.Errorf("failed to create block: %v", err)
		}

		if err := bc.AddBlock(block); err != nil {
			return fmt.Errorf("failed to add block: %v", err)
		}
	}

	return nil
}

func (bc *Blockchain) groupAtroposEventsByTime(events []*Event) [][]*Event {
	if len(events) == 0 {
		return nil
	}

	// گروه‌بندی بر اساس AtroposTime (در بازه زمانی مشخص)
	groups := make(map[uint64][]*Event)
	timeWindow := uint64(1000) // 1 second in milliseconds

	for _, event := range events {
		timeGroup := event.AtroposTime / timeWindow
		groups[timeGroup] = append(groups[timeGroup], event)
	}

	result := make([][]*Event, 0, len(groups))
	for _, group := range groups {
		result = append(result, group)
	}

	return result
}

func (bc *Blockchain) isEventInBlock(eventID EventID) bool {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	for _, block := range bc.blocks {
		for _, eid := range block.Events {
			if eid == eventID {
				return true
			}
		}
	}
	return false
}
