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

// EVMProcessor پردازش‌گر EVM برای Sinar Chain مشابه Fantom Opera
type EVMProcessor struct {
	chainConfig *params.ChainConfig
	stateDB     *StateDB
	gasLimit    uint64
	blockNumber *big.Int
	blockTime   uint64

	// Gas metering
	gasMeter *GasMeter

	// Contract management
	contractManager *ContractManager

	// State management
	stateManager *StateManager
}

// GasMeter مدیریت گس مشابه Fantom Opera
type GasMeter struct {
	gasUsed  uint64
	gasLimit uint64
	gasPrice *big.Int
	refund   uint64
	mu       sync.RWMutex
}

// ContractManager مدیریت قراردادها
type ContractManager struct {
	contracts map[common.Address]*ContractInfo
	mu        sync.RWMutex
}

// ContractInfo اطلاعات قرارداد
type ContractInfo struct {
	Address   common.Address
	Code      []byte
	Creator   common.Address
	CreatedAt uint64
	GasUsed   uint64
	CallCount uint64
}

// StateManager مدیریت state
type StateManager struct {
	snapshots map[int]*state.StateDB
	currentID int
	mu        sync.RWMutex
}

// EVMContext اطلاعات context برای EVM مشابه Fantom Opera
type EVMContext struct {
	BlockNumber *big.Int
	BlockTime   uint64
	GasLimit    uint64
	Difficulty  *big.Int
	BaseFee     *big.Int
	Coinbase    common.Address
}

// NewEVMProcessor ایجاد EVM Processor جدید
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
		gasMeter: &GasMeter{
			gasUsed:  0,
			gasLimit: 30000000,
			gasPrice: big.NewInt(1),
			refund:   0,
		},
		contractManager: &ContractManager{
			contracts: make(map[common.Address]*ContractInfo),
		},
		stateManager: &StateManager{
			snapshots: make(map[int]*state.StateDB),
			currentID: 0,
		},
	}
}

// SetStateDB تنظیم StateDB
func (ep *EVMProcessor) SetStateDB(stateDB *StateDB) {
	ep.stateDB = stateDB
}

// SetBlockInfo تنظیم اطلاعات بلاک
func (ep *EVMProcessor) SetBlockInfo(blockNumber *big.Int, blockTime uint64) {
	ep.blockNumber = blockNumber
	ep.blockTime = blockTime
}

// ProcessBlock پردازش بلاک با EVM مشابه Fantom Opera
func (ep *EVMProcessor) ProcessBlock(block *Block, parentState *state.StateDB) (*state.StateDB, error) {
	// ایجاد state جدید
	var newState *state.StateDB
	if parentState == nil {
		// Genesis state - ایجاد state جدید
		newState, _ = state.New(common.Hash{}, nil, nil)
	} else {
		// کپی کردن state از parent
		newState = parentState.Copy()
	}

	// به‌روزرسانی block info
	ep.SetBlockInfo(big.NewInt(int64(block.Header.Number)), block.Header.AtroposTime)

	// پردازش تراکنش‌ها
	totalGasUsed := uint64(0)
	for i, tx := range block.Transactions {
		if err := ep.processTransaction(tx, newState); err != nil {
			return nil, fmt.Errorf("failed to process transaction %d: %v", i, err)
		}

		// جمع‌آوری gas used
		totalGasUsed += ep.gasMeter.gasUsed
	}

	// به‌روزرسانی block header
	block.Header.GasUsed = totalGasUsed

	return newState, nil
}

// processTransaction پردازش یک تراکنش مشابه Fantom Opera
func (ep *EVMProcessor) processTransaction(tx *types.Transaction, state *state.StateDB) error {
	// Reset gas meter
	ep.gasMeter.Reset()

	// دریافت sender از تراکنش
	signer := types.LatestSignerForChainID(ep.chainConfig.ChainID)
	from, err := types.Sender(signer, tx)
	if err != nil {
		return fmt.Errorf("failed to get sender: %v", err)
	}

	// بررسی موجودی
	required := new(big.Int).Add(tx.Value(), new(big.Int).Mul(tx.GasPrice(), new(big.Int).SetUint64(tx.Gas())))
	if state.GetBalance(from).Cmp(required) < 0 {
		return fmt.Errorf("insufficient balance")
	}

	// کسر موجودی
	state.SubBalance(from, required)

	// اگر تراکنش به آدرس خاصی است، موجودی اضافه کن
	if tx.To() != nil {
		state.AddBalance(*tx.To(), tx.Value())
	}

	// پرداخت gas fee به miner
	gasCost := new(big.Int).Mul(tx.GasPrice(), new(big.Int).SetUint64(tx.Gas()))
	state.AddBalance(common.HexToAddress("0x0000000000000000000000000000000000000000"), gasCost)

	// پردازش EVM اگر تراکنش شامل data است
	if len(tx.Data()) > 0 {
		if err := ep.executeEVM(tx, state); err != nil {
			return fmt.Errorf("EVM execution failed: %v", err)
		}
	}

	fmt.Printf("✅ Transaction processed: %s (From: %s, To: %s, Value: %s, Gas: %d)\n",
		tx.Hash().Hex(), from.Hex(), tx.To().Hex(), tx.Value().String(), ep.gasMeter.gasUsed)

	return nil
}

// executeEVM اجرای EVM مشابه Fantom Opera
func (ep *EVMProcessor) executeEVM(tx *types.Transaction, state *state.StateDB) error {
	// ایجاد EVM context
	context := vm.BlockContext{
		CanTransfer: ep.canTransfer,
		Transfer:    ep.transfer,
		GetHash:     ep.getHash,
		Coinbase:    common.Address{},
		BlockNumber: ep.blockNumber,
		Time:        ep.blockTime,
		Difficulty:  big.NewInt(0),
		BaseFee:     big.NewInt(0),
		GasLimit:    ep.gasLimit,
	}

	// دریافت sender از تراکنش
	signer := types.LatestSignerForChainID(ep.chainConfig.ChainID)
	from, err := types.Sender(signer, tx)
	if err != nil {
		return fmt.Errorf("failed to get sender: %v", err)
	}

	// ایجاد EVM
	evm := vm.NewEVM(context, vm.TxContext{
		Origin:   from,
		GasPrice: tx.GasPrice(),
	}, state, ep.chainConfig, vm.Config{})

	// اجرای تراکنش
	_, gasUsed, err := evm.Call(vm.AccountRef(from), *tx.To(), tx.Data(), tx.Gas(), tx.Value())
	if err != nil {
		return err
	}

	// به‌روزرسانی gas meter
	ep.gasMeter.gasUsed = gasUsed

	// ثبت قرارداد اگر تراکنش deployment است
	if tx.To() == nil {
		contractAddr := crypto.CreateAddress(from, state.GetNonce(from))
		ep.contractManager.RegisterContract(contractAddr, tx.Data(), from, ep.blockNumber.Uint64())
	}

	return nil
}

// canTransfer بررسی امکان انتقال
func (ep *EVMProcessor) canTransfer(db vm.StateDB, addr common.Address, amount *big.Int) bool {
	return db.GetBalance(addr).Cmp(amount) >= 0
}

// transfer انتقال موجودی
func (ep *EVMProcessor) transfer(db vm.StateDB, sender, recipient common.Address, amount *big.Int) {
	db.SubBalance(sender, amount)
	db.AddBalance(recipient, amount)
}

// getHash دریافت hash بلاک
func (ep *EVMProcessor) getHash(n uint64) common.Hash {
	// در نسخه کامل، این از blockchain گرفته می‌شود
	return common.Hash{}
}

// DeployContract استقرار قرارداد هوشمند مشابه Fantom Opera
func (ep *EVMProcessor) DeployContract(creator common.Address, code []byte, gasLimit uint64) (common.Address, error) {
	// ایجاد state جدید
	state, _ := state.New(common.Hash{}, nil, nil)

	// محاسبه آدرس قرارداد
	contractAddr := crypto.CreateAddress(creator, 0) // nonce = 0 for deployment

	// ذخیره کد قرارداد
	state.SetCode(contractAddr, code)

	// ایجاد EVM context
	context := vm.BlockContext{
		CanTransfer: ep.canTransfer,
		Transfer:    ep.transfer,
		GetHash:     ep.getHash,
		Coinbase:    common.Address{},
		BlockNumber: ep.blockNumber,
		Time:        ep.blockTime,
		Difficulty:  big.NewInt(0),
		BaseFee:     big.NewInt(0),
		GasLimit:    gasLimit,
	}

	// ایجاد EVM
	evm := vm.NewEVM(context, vm.TxContext{
		Origin:   creator,
		GasPrice: big.NewInt(0),
	}, state, ep.chainConfig, vm.Config{})

	// اجرای deployment
	_, _, _, err := evm.Create(vm.AccountRef(creator), code, gasLimit, big.NewInt(0))
	if err != nil {
		return common.Address{}, fmt.Errorf("contract deployment failed: %v", err)
	}

	// ثبت قرارداد
	ep.contractManager.RegisterContract(contractAddr, code, creator, ep.blockNumber.Uint64())

	fmt.Printf("🏗️ Contract deployed at: %s\n", contractAddr.Hex())

	return contractAddr, nil
}

// CallContract فراخوانی قرارداد هوشمند مشابه Fantom Opera
func (ep *EVMProcessor) CallContract(contractAddr common.Address, caller common.Address, data []byte, gasLimit uint64) ([]byte, error) {
	// ایجاد state جدید
	state, _ := state.New(common.Hash{}, nil, nil)

	// بررسی وجود قرارداد
	code := state.GetCode(contractAddr)
	if len(code) == 0 {
		return nil, fmt.Errorf("contract not found at address: %s", contractAddr.Hex())
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
		BaseFee:     big.NewInt(0),
		GasLimit:    gasLimit,
	}

	// ایجاد EVM
	evm := vm.NewEVM(context, vm.TxContext{
		Origin:   caller,
		GasPrice: big.NewInt(0),
	}, state, ep.chainConfig, vm.Config{})

	// اجرای فراخوانی
	result, gasUsed, err := evm.Call(vm.AccountRef(caller), contractAddr, data, gasLimit, big.NewInt(0))
	if err != nil {
		return nil, fmt.Errorf("contract call failed: %v", err)
	}

	// به‌روزرسانی آمار قرارداد
	ep.contractManager.UpdateContractStats(contractAddr, gasUsed)

	fmt.Printf("📞 Contract called: %s (Data: %x, Result: %x, Gas: %d)\n",
		contractAddr.Hex(), data, result, gasUsed)

	return result, nil
}

// GetBalance دریافت موجودی یک آدرس
func (ep *EVMProcessor) GetBalance(state *state.StateDB, address common.Address) *big.Int {
	return state.GetBalance(address)
}

// SetBalance تنظیم موجودی یک آدرس
func (ep *EVMProcessor) SetBalance(state *state.StateDB, address common.Address, balance *big.Int) {
	state.SetBalance(address, balance)
}

// GetCode دریافت کد قرارداد
func (ep *EVMProcessor) GetCode(state *state.StateDB, address common.Address) []byte {
	return state.GetCode(address)
}

// SetCode تنظیم کد قرارداد
func (ep *EVMProcessor) SetCode(state *state.StateDB, address common.Address, code []byte) {
	state.SetCode(address, code)
}

// GetStorage دریافت storage
func (ep *EVMProcessor) GetStorage(state *state.StateDB, address common.Address, key common.Hash) common.Hash {
	return state.GetState(address, key)
}

// SetStorage تنظیم storage
func (ep *EVMProcessor) SetStorage(state *state.StateDB, address common.Address, key, value common.Hash) {
	state.SetState(address, key, value)
}

// GetNonce دریافت nonce
func (ep *EVMProcessor) GetNonce(state *state.StateDB, address common.Address) uint64 {
	return state.GetNonce(address)
}

// SetNonce تنظیم nonce
func (ep *EVMProcessor) SetNonce(state *state.StateDB, address common.Address, nonce uint64) {
	state.SetNonce(address, nonce)
}

// CreateAccount ایجاد حساب جدید
func (ep *EVMProcessor) CreateAccount(state *state.StateDB, address common.Address) {
	state.CreateAccount(address)
}

// Exist بررسی وجود حساب
func (ep *EVMProcessor) Exist(state *state.StateDB, address common.Address) bool {
	return state.Exist(address)
}

// Empty بررسی خالی بودن حساب
func (ep *EVMProcessor) Empty(state *state.StateDB, address common.Address) bool {
	return state.Empty(address)
}

// Suicide خودکشی حساب
func (ep *EVMProcessor) Suicide(state *state.StateDB, address common.Address) bool {
	// در نسخه جدید go-ethereum، Suicide حذف شده است
	return false
}

// GetRefund دریافت refund
func (ep *EVMProcessor) GetRefund(state *state.StateDB) uint64 {
	return state.GetRefund()
}

// AddRefund اضافه کردن refund
func (ep *EVMProcessor) AddRefund(state *state.StateDB, gas uint64) {
	state.AddRefund(gas)
}

// SubRefund کم کردن refund
func (ep *EVMProcessor) SubRefund(state *state.StateDB, gas uint64) {
	state.SubRefund(gas)
}

// GetCommittedState دریافت committed state
func (ep *EVMProcessor) GetCommittedState(state *state.StateDB, address common.Address, key common.Hash) common.Hash {
	return state.GetCommittedState(address, key)
}

// Snapshot گرفتن snapshot
func (ep *EVMProcessor) Snapshot(state *state.StateDB) int {
	return state.Snapshot()
}

// RevertToSnapshot برگشت به snapshot
func (ep *EVMProcessor) RevertToSnapshot(state *state.StateDB, revid int) {
	state.RevertToSnapshot(revid)
}

// GetLogs دریافت logs
func (ep *EVMProcessor) GetLogs(state *state.StateDB, hash common.Hash) []*types.Log {
	// در نسخه جدید go-ethereum، GetLogs نیاز به پارامترهای بیشتری دارد
	return []*types.Log{}
}

// AddLog اضافه کردن log
func (ep *EVMProcessor) AddLog(state *state.StateDB, log *types.Log) {
	state.AddLog(log)
}

// AddPreimage اضافه کردن preimage
func (ep *EVMProcessor) AddPreimage(state *state.StateDB, hash common.Hash, preimage []byte) {
	state.AddPreimage(hash, preimage)
}

// Prepare تنظیم EVM برای بلاک جدید
func (ep *EVMProcessor) Prepare(blockNumber *big.Int, blockTime uint64) {
	ep.blockNumber = blockNumber
	ep.blockTime = blockTime
}

// GetEVMStats آمار EVM
func (ep *EVMProcessor) GetEVMStats() map[string]interface{} {
	stats := make(map[string]interface{})

	stats["chain_id"] = ep.chainConfig.ChainID.String()
	stats["gas_limit"] = ep.gasLimit
	stats["block_number"] = ep.blockNumber.String()
	stats["block_time"] = ep.blockTime
	stats["total_contracts"] = len(ep.contractManager.contracts)
	stats["gas_used"] = ep.gasMeter.gasUsed
	stats["gas_price"] = ep.gasMeter.gasPrice.String()

	return stats
}

// Reset بازنشانی gas meter
func (gm *GasMeter) Reset() {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	gm.gasUsed = 0
	gm.refund = 0
}

// RegisterContract ثبت قرارداد جدید
func (cm *ContractManager) RegisterContract(address common.Address, code []byte, creator common.Address, blockNumber uint64) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.contracts[address] = &ContractInfo{
		Address:   address,
		Code:      code,
		Creator:   creator,
		CreatedAt: blockNumber,
		GasUsed:   0,
		CallCount: 0,
	}
}

// UpdateContractStats به‌روزرسانی آمار قرارداد
func (cm *ContractManager) UpdateContractStats(address common.Address, gasUsed uint64) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if contract, exists := cm.contracts[address]; exists {
		contract.GasUsed += gasUsed
		contract.CallCount++
	}
}

// GetContractInfo دریافت اطلاعات قرارداد
func (cm *ContractManager) GetContractInfo(address common.Address) *ContractInfo {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.contracts[address]
}

// GetAllContracts دریافت تمام قراردادها
func (cm *ContractManager) GetAllContracts() map[common.Address]*ContractInfo {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	result := make(map[common.Address]*ContractInfo)
	for addr, contract := range cm.contracts {
		result[addr] = contract
	}

	return result
}
