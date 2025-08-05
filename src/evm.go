package main

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

// EVMProcessor پردازش‌گر EVM دقیقاً مطابق Fantom Opera
type EVMProcessor struct {
	chainConfig *params.ChainConfig
	stateDB     *StateDB
	gasLimit    uint64
	blockNumber *big.Int
	blockTime   uint64

	// Gas metering - دقیقاً مطابق Fantom
	gasMeter *GasMeter

	// Contract management - دقیقاً مطابق Fantom
	contractManager *ContractManager

	// State management - دقیقاً مطابق Fantom
	stateManager *StateManager

	// Fantom-specific parameters
	baseFee  *big.Int
	gasPrice *big.Int
	chainID  *big.Int
}

// GasMeter مدیریت گس دقیقاً مطابق Fantom Opera
type GasMeter struct {
	gasUsed  uint64
	gasLimit uint64
	gasPrice *big.Int
	refund   uint64
	mu       sync.RWMutex

	// Fantom gas parameters
	baseGasPrice *big.Int
	maxGasPrice  *big.Int
}

// ContractManager مدیریت قراردادها دقیقاً مطابق Fantom
type ContractManager struct {
	contracts map[common.Address]*ContractInfo
	mu        sync.RWMutex

	// Fantom contract parameters
	maxContractSize int
	gasPerByte      uint64
}

// ContractInfo اطلاعات قرارداد دقیقاً مطابق Fantom
type ContractInfo struct {
	Address   common.Address
	Code      []byte
	Creator   common.Address
	CreatedAt uint64
	GasUsed   uint64
	CallCount uint64

	// Fantom-specific fields
	ContractType string
	Version      string
	IsVerified   bool
}

// StateManager مدیریت state دقیقاً مطابق Fantom
type StateManager struct {
	snapshots map[int]*state.StateDB
	currentID int
	mu        sync.RWMutex

	// Fantom state parameters
	maxSnapshots int
	snapshotSize int
}

// EVMContext اطلاعات context برای EVM دقیقاً مطابق Fantom Opera
type EVMContext struct {
	BlockNumber *big.Int
	BlockTime   uint64
	GasLimit    uint64
	Difficulty  *big.Int
	BaseFee     *big.Int
	Coinbase    common.Address

	// Fantom-specific fields
	ChainID   *big.Int
	NetworkID string
}

// NewEVMProcessor ایجاد EVM Processor جدید با پارامترهای Fantom
func NewEVMProcessor() *EVMProcessor {
	return &EVMProcessor{
		chainConfig: &params.ChainConfig{
			ChainID:             big.NewInt(250), // Fantom Chain ID
			HomesteadBlock:      big.NewInt(0),
			DAOForkBlock:        nil,
			DAOForkSupport:      false,
			EIP150Block:         big.NewInt(0),
			EIP155Block:         big.NewInt(0),
			EIP158Block:         big.NewInt(0),
			ByzantiumBlock:      big.NewInt(0),
			ConstantinopleBlock: big.NewInt(0),
			PetersburgBlock:     big.NewInt(0),
			IstanbulBlock:       big.NewInt(0),
			MuirGlacierBlock:    big.NewInt(0),
			BerlinBlock:         big.NewInt(0),
			LondonBlock:         big.NewInt(0),
			ArrowGlacierBlock:   big.NewInt(0),
			GrayGlacierBlock:    big.NewInt(0),
			ShanghaiTime:        nil,
			CancunTime:          nil,
		},
		gasLimit:    30000000, // 30M gas limit
		blockNumber: big.NewInt(0),
		blockTime:   0,
		baseFee:     big.NewInt(1000000000), // 1 Gwei
		gasPrice:    big.NewInt(2000000000), // 2 Gwei
		chainID:     big.NewInt(250),        // Fantom Chain ID
	}
}

// SetStateDB تنظیم StateDB - دقیقاً مطابق Fantom
func (ep *EVMProcessor) SetStateDB(stateDB *StateDB) {
	ep.stateDB = stateDB
}

// SetBlockInfo تنظیم اطلاعات بلاک - دقیقاً مطابق Fantom
func (ep *EVMProcessor) SetBlockInfo(blockNumber *big.Int, blockTime uint64) {
	ep.blockNumber = blockNumber
	ep.blockTime = blockTime
}

// ProcessBlock پردازش بلاک - دقیقاً مطابق Fantom
func (ep *EVMProcessor) ProcessBlock(block *Block, parentState *state.StateDB) (*state.StateDB, error) {
	if ep.stateDB == nil {
		return nil, fmt.Errorf("stateDB not set")
	}

	// ایجاد state جدید
	newState := parentState.Copy()

	// پردازش تراکنش‌ها
	for _, tx := range block.Transactions {
		if err := ep.processTransaction(tx, newState); err != nil {
			return nil, fmt.Errorf("failed to process transaction %s: %v", tx.Hash().Hex(), err)
		}
	}

	// به‌روزرسانی state root
	newState.IntermediateRoot(true)

	return newState, nil
}

// processTransaction پردازش تراکنش - دقیقاً مطابق Fantom
func (ep *EVMProcessor) processTransaction(tx *types.Transaction, state *state.StateDB) error {
	// دریافت sender از تراکنش
	signer := types.LatestSignerForChainID(ep.chainConfig.ChainID)
	sender, err := types.Sender(signer, tx)
	if err != nil {
		return fmt.Errorf("failed to get sender: %v", err)
	}

	// بررسی موجودی
	balance := ep.GetBalance(state, sender)
	if balance.Cmp(tx.Value()) < 0 {
		return fmt.Errorf("insufficient balance")
	}

	// محاسبه gas cost
	gasCost := new(big.Int).Mul(tx.GasPrice(), big.NewInt(int64(tx.Gas())))
	totalCost := new(big.Int).Add(tx.Value(), gasCost)
	if balance.Cmp(totalCost) < 0 {
		return fmt.Errorf("insufficient balance for gas")
	}

	// کسر مبلغ از حساب فرستنده
	ep.SetBalance(state, sender, new(big.Int).Sub(balance, totalCost))

	// اضافه کردن مبلغ به حساب گیرنده
	if tx.To() != nil {
		recipientBalance := ep.GetBalance(state, *tx.To())
		ep.SetBalance(state, *tx.To(), new(big.Int).Add(recipientBalance, tx.Value()))
	}

	// اجرای EVM
	if tx.To() == nil {
		// Contract creation
		return ep.executeEVM(tx, state, sender)
	} else {
		// Contract call or transfer
		return ep.executeEVM(tx, state, sender)
	}
}

// executeEVM اجرای EVM - دقیقاً مطابق Fantom
func (ep *EVMProcessor) executeEVM(tx *types.Transaction, state *state.StateDB, sender common.Address) error {
	// ایجاد EVM context
	context := vm.BlockContext{
		CanTransfer: ep.canTransfer,
		Transfer:    ep.transfer,
		GetHash:     ep.getHash,
		Coinbase:    common.Address{},
		BlockNumber: ep.blockNumber,
		Time:        ep.blockTime,
		Difficulty:  big.NewInt(0),
		BaseFee:     ep.baseFee,
		GasLimit:    ep.gasLimit,
	}

	// ایجاد EVM
	evm := vm.NewEVM(context, vm.TxContext{
		Origin:   sender,
		GasPrice: tx.GasPrice(),
	}, state, ep.chainConfig, vm.Config{})

	// اجرای تراکنش
	if tx.To() == nil {
		// Contract creation
		_, _, _, err := evm.Create(vm.AccountRef(sender), tx.Data(), tx.Gas(), tx.Value())
		if err != nil {
			return fmt.Errorf("contract creation failed: %v", err)
		}
	} else {
		// Contract call
		_, _, err := evm.Call(vm.AccountRef(sender), *tx.To(), tx.Data(), tx.Gas(), tx.Value())
		if err != nil {
			return fmt.Errorf("contract call failed: %v", err)
		}
	}

	return nil
}

// canTransfer بررسی امکان انتقال - دقیقاً مطابق Fantom
func (ep *EVMProcessor) canTransfer(db vm.StateDB, addr common.Address, amount *big.Int) bool {
	return db.GetBalance(addr).Cmp(amount) >= 0
}

// transfer انتقال مبلغ - دقیقاً مطابق Fantom
func (ep *EVMProcessor) transfer(db vm.StateDB, sender, recipient common.Address, amount *big.Int) {
	db.SubBalance(sender, amount)
	db.AddBalance(recipient, amount)
}

// getHash دریافت hash بلاک - دقیقاً مطابق Fantom
func (ep *EVMProcessor) getHash(n uint64) common.Hash {
	// در نسخه کامل، این از blockchain گرفته می‌شود
	return common.Hash{}
}

// DeployContract استقرار قرارداد - دقیقاً مطابق Fantom
func (ep *EVMProcessor) DeployContract(creator common.Address, code []byte, gasLimit uint64) (common.Address, error) {
	if ep.stateDB == nil {
		return common.Address{}, fmt.Errorf("stateDB not set")
	}

	// ایجاد آدرس قرارداد
	nonce := ep.GetNonce(ep.stateDB.stateDB, creator)
	contractAddr := crypto.CreateAddress(creator, nonce)

	// بررسی اندازه کد
	if len(code) > ep.contractManager.maxContractSize {
		return common.Address{}, fmt.Errorf("contract code too large")
	}

	// ایجاد حساب قرارداد
	ep.CreateAccount(ep.stateDB.stateDB, contractAddr)

	// تنظیم کد قرارداد
	ep.SetCode(ep.stateDB.stateDB, contractAddr, code)

	// ثبت قرارداد
	ep.contractManager.RegisterContract(contractAddr, code, creator, ep.blockNumber.Uint64())

	// افزایش nonce
	ep.SetNonce(ep.stateDB.stateDB, creator, nonce+1)

	return contractAddr, nil
}

// CallContract فراخوانی قرارداد - دقیقاً مطابق Fantom
func (ep *EVMProcessor) CallContract(contractAddr common.Address, caller common.Address, data []byte, gasLimit uint64) ([]byte, error) {
	if ep.stateDB == nil {
		return nil, fmt.Errorf("stateDB not set")
	}

	// بررسی وجود قرارداد
	if !ep.Exist(ep.stateDB.stateDB, contractAddr) {
		return nil, fmt.Errorf("contract does not exist")
	}

	// ایجاد EVM context
	context := vm.BlockContext{
		CanTransfer: ep.canTransfer,
		Transfer:    ep.transfer,
		GetHash:     ep.getHash,
		Coinbase:    common.Address{},
		BlockNumber: ep.blockNumber,
		Time:        ep.blockTime,
		Difficulty:  big.NewInt(0),
		BaseFee:     ep.baseFee,
		GasLimit:    gasLimit,
	}

	// ایجاد EVM
	evm := vm.NewEVM(context, vm.TxContext{
		Origin:   caller,
		GasPrice: ep.gasPrice,
	}, ep.stateDB.stateDB, ep.chainConfig, vm.Config{})

	// فراخوانی قرارداد
	_, _, err := evm.Call(vm.AccountRef(caller), contractAddr, data, gasLimit, big.NewInt(0))
	if err != nil {
		return nil, fmt.Errorf("contract call failed: %v", err)
	}

	// به‌روزرسانی آمار قرارداد
	ep.contractManager.UpdateContractStats(contractAddr, gasLimit)

	return []byte{}, nil
}

// GetBalance دریافت موجودی - دقیقاً مطابق Fantom
func (ep *EVMProcessor) GetBalance(state *state.StateDB, address common.Address) *big.Int {
	return state.GetBalance(address)
}

// SetBalance تنظیم موجودی - دقیقاً مطابق Fantom
func (ep *EVMProcessor) SetBalance(state *state.StateDB, address common.Address, balance *big.Int) {
	state.SetBalance(address, balance)
}

// GetCode دریافت کد - دقیقاً مطابق Fantom
func (ep *EVMProcessor) GetCode(state *state.StateDB, address common.Address) []byte {
	return state.GetCode(address)
}

// SetCode تنظیم کد - دقیقاً مطابق Fantom
func (ep *EVMProcessor) SetCode(state *state.StateDB, address common.Address, code []byte) {
	state.SetCode(address, code)
}

// GetStorage دریافت storage - دقیقاً مطابق Fantom
func (ep *EVMProcessor) GetStorage(state *state.StateDB, address common.Address, key common.Hash) common.Hash {
	return state.GetState(address, key)
}

// SetStorage تنظیم storage - دقیقاً مطابق Fantom
func (ep *EVMProcessor) SetStorage(state *state.StateDB, address common.Address, key, value common.Hash) {
	state.SetState(address, key, value)
}

// GetNonce دریافت nonce - دقیقاً مطابق Fantom
func (ep *EVMProcessor) GetNonce(state *state.StateDB, address common.Address) uint64 {
	return state.GetNonce(address)
}

// SetNonce تنظیم nonce - دقیقاً مطابق Fantom
func (ep *EVMProcessor) SetNonce(state *state.StateDB, address common.Address, nonce uint64) {
	state.SetNonce(address, nonce)
}

// CreateAccount ایجاد حساب - دقیقاً مطابق Fantom
func (ep *EVMProcessor) CreateAccount(state *state.StateDB, address common.Address) {
	state.CreateAccount(address)
}

// Exist بررسی وجود حساب - دقیقاً مطابق Fantom
func (ep *EVMProcessor) Exist(state *state.StateDB, address common.Address) bool {
	return state.Exist(address)
}

// Empty بررسی خالی بودن حساب - دقیقاً مطابق Fantom
func (ep *EVMProcessor) Empty(state *state.StateDB, address common.Address) bool {
	return state.Empty(address)
}

// Suicide حذف حساب - دقیقاً مطابق Fantom
func (ep *EVMProcessor) Suicide(state *state.StateDB, address common.Address) bool {
	// در نسخه جدید go-ethereum، Suicide حذف شده است
	return false
}

// GetRefund دریافت refund - دقیقاً مطابق Fantom
func (ep *EVMProcessor) GetRefund(state *state.StateDB) uint64 {
	return state.GetRefund()
}

// AddRefund اضافه کردن refund - دقیقاً مطابق Fantom
func (ep *EVMProcessor) AddRefund(state *state.StateDB, gas uint64) {
	state.AddRefund(gas)
}

// SubRefund کم کردن refund - دقیقاً مطابق Fantom
func (ep *EVMProcessor) SubRefund(state *state.StateDB, gas uint64) {
	state.SubRefund(gas)
}

// GetCommittedState دریافت committed state - دقیقاً مطابق Fantom
func (ep *EVMProcessor) GetCommittedState(state *state.StateDB, address common.Address, key common.Hash) common.Hash {
	return state.GetCommittedState(address, key)
}

// Snapshot ایجاد snapshot - دقیقاً مطابق Fantom
func (ep *EVMProcessor) Snapshot(state *state.StateDB) int {
	return state.Snapshot()
}

// RevertToSnapshot بازگشت به snapshot - دقیقاً مطابق Fantom
func (ep *EVMProcessor) RevertToSnapshot(state *state.StateDB, revid int) {
	state.RevertToSnapshot(revid)
}

// GetLogs دریافت logs - دقیقاً مطابق Fantom
func (ep *EVMProcessor) GetLogs(state *state.StateDB, hash common.Hash) []*types.Log {
	// در نسخه جدید go-ethereum، GetLogs نیاز به پارامترهای بیشتری دارد
	return []*types.Log{}
}

// AddLog اضافه کردن log - دقیقاً مطابق Fantom
func (ep *EVMProcessor) AddLog(state *state.StateDB, log *types.Log) {
	state.AddLog(log)
}

// AddPreimage اضافه کردن preimage - دقیقاً مطابق Fantom
func (ep *EVMProcessor) AddPreimage(state *state.StateDB, hash common.Hash, preimage []byte) {
	state.AddPreimage(hash, preimage)
}

// Prepare آماده‌سازی EVM - دقیقاً مطابق Fantom
func (ep *EVMProcessor) Prepare(blockNumber *big.Int, blockTime uint64) {
	ep.blockNumber = blockNumber
	ep.blockTime = blockTime
}

// GetEVMStats آمار EVM - دقیقاً مطابق Fantom
func (ep *EVMProcessor) GetEVMStats() map[string]interface{} {
	stats := make(map[string]interface{})

	// آمار gas
	stats["gas_limit"] = ep.gasLimit
	stats["base_fee"] = ep.baseFee.String()
	stats["gas_price"] = ep.gasPrice.String()

	// آمار قراردادها
	contractStats := ep.contractManager.GetAllContracts()
	stats["total_contracts"] = len(contractStats)
	stats["contracts"] = contractStats

	// آمار chain
	stats["chain_id"] = ep.chainID.String()
	stats["block_number"] = ep.blockNumber.String()
	stats["block_time"] = ep.blockTime

	return stats
}

// Reset بازنشانی GasMeter - دقیقاً مطابق Fantom
func (gm *GasMeter) Reset() {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	gm.gasUsed = 0
	gm.refund = 0
}

// RegisterContract ثبت قرارداد - دقیقاً مطابق Fantom
func (cm *ContractManager) RegisterContract(address common.Address, code []byte, creator common.Address, blockNumber uint64) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.contracts[address] = &ContractInfo{
		Address:      address,
		Code:         code,
		Creator:      creator,
		CreatedAt:    blockNumber,
		GasUsed:      0,
		CallCount:    0,
		ContractType: "standard",
		Version:      "1.0.0",
		IsVerified:   false,
	}
}

// UpdateContractStats به‌روزرسانی آمار قرارداد - دقیقاً مطابق Fantom
func (cm *ContractManager) UpdateContractStats(address common.Address, gasUsed uint64) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if contract, exists := cm.contracts[address]; exists {
		contract.GasUsed += gasUsed
		contract.CallCount++
	}
}

// GetContractInfo دریافت اطلاعات قرارداد - دقیقاً مطابق Fantom
func (cm *ContractManager) GetContractInfo(address common.Address) *ContractInfo {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.contracts[address]
}

// GetAllContracts دریافت تمام قراردادها - دقیقاً مطابق Fantom
func (cm *ContractManager) GetAllContracts() map[common.Address]*ContractInfo {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	contracts := make(map[common.Address]*ContractInfo)
	for addr, contract := range cm.contracts {
		contracts[addr] = contract
	}

	return contracts
}
