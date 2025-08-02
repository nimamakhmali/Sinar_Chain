package main

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

// EVMProcessor پردازش‌گر EVM برای Sinar Chain
type EVMProcessor struct {
	chainConfig *params.ChainConfig
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
	}
}

// ProcessBlock پردازش بلاک با EVM
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

	// پردازش تراکنش‌ها
	for i, tx := range block.Transactions {
		if err := ep.processTransaction(tx, newState); err != nil {
			return nil, fmt.Errorf("failed to process transaction %d: %v", i, err)
		}
	}

	return newState, nil
}

// processTransaction پردازش یک تراکنش
func (ep *EVMProcessor) processTransaction(tx *types.Transaction, state *state.StateDB) error {
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

	fmt.Printf("✅ Transaction processed: %s (From: %s, To: %s, Value: %s)\n",
		tx.Hash().Hex(), from.Hex(), tx.To().Hex(), tx.Value().String())

	return nil
}

// DeployContract استقرار قرارداد هوشمند
func (ep *EVMProcessor) DeployContract(creator common.Address, code []byte, gasLimit uint64) (common.Address, error) {
	// ایجاد state جدید
	state, _ := state.New(common.Hash{}, nil, nil)

	// محاسبه آدرس قرارداد
	contractAddr := crypto.CreateAddress(creator, 0) // nonce = 0 for deployment

	// ذخیره کد قرارداد
	state.SetCode(contractAddr, code)

	fmt.Printf("🏗️ Contract deployed at: %s\n", contractAddr.Hex())

	return contractAddr, nil
}

// CallContract فراخوانی قرارداد هوشمند
func (ep *EVMProcessor) CallContract(contractAddr common.Address, caller common.Address, data []byte, gasLimit uint64) ([]byte, error) {
	// ایجاد state جدید
	state, _ := state.New(common.Hash{}, nil, nil)

	// بررسی وجود قرارداد
	code := state.GetCode(contractAddr)
	if len(code) == 0 {
		return nil, fmt.Errorf("contract not found at address: %s", contractAddr.Hex())
	}

	fmt.Printf("📞 Contract called: %s (Data: %x)\n", contractAddr.Hex(), data)

	// در نسخه کامل، اینجا EVM اجرا می‌شود
	// فعلاً فقط یک پیام برمی‌گردانیم
	return []byte("contract_call_success"), nil
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
