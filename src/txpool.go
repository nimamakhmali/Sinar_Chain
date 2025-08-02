package main

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// TransactionPool استخر تراکنش‌های pending
type TransactionPool struct {
	pending map[common.Hash]*types.Transaction
	queued  map[common.Hash]*types.Transaction
	mu      sync.RWMutex
}

// NewTransactionPool ایجاد استخر تراکنش جدید
func NewTransactionPool() *TransactionPool {
	return &TransactionPool{
		pending: make(map[common.Hash]*types.Transaction),
		queued:  make(map[common.Hash]*types.Transaction),
	}
}

// AddTransaction اضافه کردن تراکنش به استخر
func (tp *TransactionPool) AddTransaction(tx *types.Transaction) error {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	txHash := tx.Hash()

	// بررسی تکراری نبودن
	if _, exists := tp.pending[txHash]; exists {
		return fmt.Errorf("transaction already exists in pending pool")
	}
	if _, exists := tp.queued[txHash]; exists {
		return fmt.Errorf("transaction already exists in queued pool")
	}

	// اضافه کردن به pending
	tp.pending[txHash] = tx

	fmt.Printf("Transaction added to pool: %s\n", txHash.Hex())
	return nil
}

// GetPendingTransactions دریافت تراکنش‌های pending
func (tp *TransactionPool) GetPendingTransactions() []*types.Transaction {
	tp.mu.RLock()
	defer tp.mu.RUnlock()

	txs := make([]*types.Transaction, 0, len(tp.pending))
	for _, tx := range tp.pending {
		txs = append(txs, tx)
	}
	return txs
}

// GetTransactionsForBlock دریافت تراکنش‌ها برای یک بلاک
func (tp *TransactionPool) GetTransactionsForBlock(blockGasLimit uint64) []*types.Transaction {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	var selectedTxs []*types.Transaction
	currentGas := uint64(0)

	// مرتب‌سازی بر اساس gas price (بالاترین اول)
	sortedTxs := tp.sortTransactionsByGasPrice()

	for _, tx := range sortedTxs {
		if currentGas+tx.Gas() <= blockGasLimit {
			selectedTxs = append(selectedTxs, tx)
			currentGas += tx.Gas()

			// حذف از pending
			delete(tp.pending, tx.Hash())
		} else {
			break
		}
	}

	return selectedTxs
}

// sortTransactionsByGasPrice مرتب‌سازی تراکنش‌ها بر اساس gas price
func (tp *TransactionPool) sortTransactionsByGasPrice() []*types.Transaction {
	txs := make([]*types.Transaction, 0, len(tp.pending))
	for _, tx := range tp.pending {
		txs = append(txs, tx)
	}

	// مرتب‌سازی ساده بر اساس gas price
	for i := 0; i < len(txs)-1; i++ {
		for j := i + 1; j < len(txs); j++ {
			if txs[i].GasPrice().Cmp(txs[j].GasPrice()) < 0 {
				txs[i], txs[j] = txs[j], txs[i]
			}
		}
	}

	return txs
}

// RemoveTransaction حذف تراکنش از استخر
func (tp *TransactionPool) RemoveTransaction(txHash common.Hash) {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	delete(tp.pending, txHash)
	delete(tp.queued, txHash)
}

// GetPoolStats آمار استخر
func (tp *TransactionPool) GetPoolStats() (pending, queued int) {
	tp.mu.RLock()
	defer tp.mu.RUnlock()

	return len(tp.pending), len(tp.queued)
}

// ClearPool پاک کردن استخر
func (tp *TransactionPool) ClearPool() {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	tp.pending = make(map[common.Hash]*types.Transaction)
	tp.queued = make(map[common.Hash]*types.Transaction)
}

// CreateSampleTransaction ایجاد تراکنش نمونه
func CreateSampleTransaction(from, to common.Address, value *big.Int, gasPrice *big.Int) *types.Transaction {
	// ایجاد تراکنش ساده
	tx := types.NewTransaction(
		0,        // nonce
		to,       // to
		value,    // value
		21000,    // gas limit
		gasPrice, // gas price
		nil,      // data
	)

	return tx
}
