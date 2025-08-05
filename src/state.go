package main

import (
	"crypto/sha256"
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

// SINAR Token Configuration
const (
	SINAR_SYMBOL        = "SINAR"
	SINAR_DECIMALS      = 18
	SINAR_TOTAL_SUPPLY  = "1000000000000000000000000000" // 1 Billion SINAR (with 18 decimals)
	SINAR_INITIAL_PRICE = "0.01"                         // $0.01 USD

	// Initial Distribution Percentages
	SINAR_TEAM_PERCENTAGE       = 15 // 15% for team and founders
	SINAR_ECOSYSTEM_PERCENTAGE  = 25 // 25% for ecosystem development
	SINAR_VALIDATORS_PERCENTAGE = 20 // 20% for initial validators
	SINAR_LIQUIDITY_PERCENTAGE  = 10 // 10% for liquidity pools
	SINAR_COMMUNITY_PERCENTAGE  = 20 // 20% for community rewards
	SINAR_RESERVE_PERCENTAGE    = 10 // 10% for future development
)

// SINARToken اطلاعات ارز بومی سینار
type SINARToken struct {
	Symbol        string
	Decimals      uint8
	TotalSupply   *big.Int
	CurrentSupply *big.Int
	Price         *big.Float
	Creator       common.Address
	CreatedAt     uint64
}

// InitialDistribution آدرس‌های اولیه برای توزیع سینار
type InitialDistribution struct {
	TeamWallets      []common.Address
	EcosystemWallets []common.Address
	ValidatorWallets []common.Address
	LiquidityWallets []common.Address
	CommunityWallets []common.Address
	ReserveWallets   []common.Address
}

// NewInitialDistribution ایجاد توزیع اولیه
func NewInitialDistribution() *InitialDistribution {
	return &InitialDistribution{
		// Team Wallets (15% = 150M SINAR)
		TeamWallets: []common.Address{
			common.HexToAddress("0x1111111111111111111111111111111111111111"), // Founder 1
			common.HexToAddress("0x2222222222222222222222222222222222222222"), // Founder 2
			common.HexToAddress("0x3333333333333333333333333333333333333333"), // founder 3
			common.HexToAddress("0x4444444444444444444444444444444444444444"), // founder 4
			common.HexToAddress("0x5555555555555555555555555555555555555555"), // Marketing Lead
		},

		// Ecosystem Wallets (25% = 250M SINAR)
		EcosystemWallets: []common.Address{
			common.HexToAddress("0x6666666666666666666666666666666666666666"), // Development Fund
			common.HexToAddress("0x7777777777777777777777777777777777777777"), // Partnership Fund
			common.HexToAddress("0x8888888888888888888888888888888888888888"), // Research Fund
			common.HexToAddress("0x9999999999999999999999999999999999999999"), // Education Fund
			common.HexToAddress("0xAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"), // Innovation Fund
		},

		// Validator Wallets (20% = 200M SINAR)
		ValidatorWallets: []common.Address{
			common.HexToAddress("0xBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"), // Validator 1
			common.HexToAddress("0xCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC"), // Validator 2
			common.HexToAddress("0xDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDD"), // Validator 3
			common.HexToAddress("0xEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEE"), // Validator 4
			common.HexToAddress("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"), // Validator 5
		},

		// Liquidity Wallets (10% = 100M SINAR)
		LiquidityWallets: []common.Address{
			common.HexToAddress("0x1010101010101010101010101010101010101010"), // DEX Liquidity
			common.HexToAddress("0x2020202020202020202020202020202020202020"), // Bridge Liquidity
			common.HexToAddress("0x3030303030303030303030303030303030303030"), // Staking Pool
		},

		// Community Wallets (20% = 200M SINAR)
		CommunityWallets: []common.Address{
			common.HexToAddress("0x4040404040404040404040404040404040404040"), // Community Rewards
			common.HexToAddress("0x5050505050505050505050505050505050505050"), // Airdrop Fund
			common.HexToAddress("0x6060606060606060606060606060606060606060"), // Bug Bounty
			common.HexToAddress("0x7070707070707070707070707070707070707070"), // Hackathon Prizes
			common.HexToAddress("0x8080808080808080808080808080808080808080"), // Ambassador Program
		},

		// Reserve Wallets (10% = 100M SINAR)
		ReserveWallets: []common.Address{
			common.HexToAddress("0x9090909090909090909090909090909090909090"), // Emergency Fund
			common.HexToAddress("0xA0A0A0A0A0A0A0A0A0A0A0A0A0A0A0A0A0A0A0A0"), // Future Development
			common.HexToAddress("0xB0B0B0B0B0B0B0B0B0B0B0B0B0B0B0B0B0B0B0B0"), // Strategic Reserve
		},
	}
}

// NewSINARToken ایجاد ارز بومی سینار
func NewSINARToken() *SINARToken {
	totalSupply, _ := new(big.Int).SetString(SINAR_TOTAL_SUPPLY, 10)
	price, _ := new(big.Float).SetString(SINAR_INITIAL_PRICE)

	return &SINARToken{
		Symbol:        SINAR_SYMBOL,
		Decimals:      SINAR_DECIMALS,
		TotalSupply:   totalSupply,
		CurrentSupply: big.NewInt(0),
		Price:         price,
		Creator:       common.HexToAddress("0x0000000000000000000000000000000000000000"), // Zero address for native token
		CreatedAt:     uint64(time.Now().Unix()),
	}
}

// StateDB مدیریت state مشابه Fantom Opera
type StateDB struct {
	mu sync.RWMutex

	// State storage
	stateDB *state.StateDB

	// Account management
	accounts map[common.Address]*Account

	// Contract storage
	contracts map[common.Address]*Contract

	// Validator state
	validators map[common.Address]*ValidatorState

	// Staking state
	stakes map[common.Address]*big.Int

	// Delegation state
	delegations map[common.Address]map[common.Address]*big.Int

	// Governance state
	proposals map[uint64]*Proposal
	votes     map[uint64]map[common.Address]*StateVote

	// Native SINAR Token
	sinarToken *SINARToken

	// State root cache
	stateRootCache common.Hash
	stateRootDirty bool

	// Configuration
	config *StateConfig

	// Memory optimization
	memoryOptimizer *MemoryOptimizer
}

type Account struct {
	Address     common.Address
	Balance     *big.Int
	Nonce       uint64
	Code        []byte
	CodeHash    common.Hash
	Storage     map[common.Hash]common.Hash
	IsContract  bool
	Stake       *big.Int
	IsValidator bool
	LastSeen    uint64
}

type Contract struct {
	Address   common.Address
	Code      []byte
	Storage   map[common.Hash]common.Hash
	Balance   *big.Int
	Nonce     uint64
	Creator   common.Address
	CreatedAt uint64
	UpdatedAt uint64
}

type ValidatorState struct {
	Address    common.Address
	Stake      *big.Int
	IsActive   bool
	Commission uint64 // در هزارم
	Delegators map[common.Address]*big.Int
	TotalStake *big.Int
	Rewards    *big.Int
	LastReward uint64
	LastSeen   uint64
}

type Proposal struct {
	ID           uint64
	Title        string
	Description  string
	Creator      common.Address
	Type         ProposalType
	Data         []byte
	StartTime    uint64
	EndTime      uint64
	Executed     bool
	VotesFor     *big.Int
	VotesAgainst *big.Int
	TotalVotes   *big.Int
}

type StateVote struct {
	Voter    common.Address
	Proposal uint64
	Choice   VoteChoice
	Stake    *big.Int
	Time     uint64
}

type ProposalType uint8

const (
	ProposalTypeText ProposalType = iota
	ProposalTypeParameter
	ProposalTypeUpgrade
	ProposalTypeValidator
)

type VoteChoice uint8

const (
	VoteChoiceFor VoteChoice = iota
	VoteChoiceAgainst
	VoteChoiceAbstain
)

type StateConfig struct {
	MinStake           *big.Int
	ValidatorReward    *big.Int
	DelegatorReward    *big.Int
	MaxValidators      uint64
	MinValidators      uint64
	StateRootCacheSize int
}

// NewStateDB ایجاد StateDB جدید
func NewStateDB() *StateDB {
	config := &StateConfig{
		MinStake:           big.NewInt(1000000), // 1M tokens
		ValidatorReward:    big.NewInt(100),     // 100 tokens
		DelegatorReward:    big.NewInt(10),      // 10 tokens
		MaxValidators:      100,
		MinValidators:      4,
		StateRootCacheSize: 1000,
	}

	// ایجاد memory optimizer
	memoryOptimizer := NewMemoryOptimizer()

	return &StateDB{
		stateDB:         nil, // فعلاً nil می‌گذاریم
		accounts:        make(map[common.Address]*Account),
		contracts:       make(map[common.Address]*Contract),
		validators:      make(map[common.Address]*ValidatorState),
		stakes:          make(map[common.Address]*big.Int),
		delegations:     make(map[common.Address]map[common.Address]*big.Int),
		proposals:       make(map[uint64]*Proposal),
		votes:           make(map[uint64]map[common.Address]*StateVote),
		sinarToken:      NewSINARToken(),
		config:          config,
		stateRootDirty:  true,
		memoryOptimizer: memoryOptimizer,
	}
}

// GetAccount دریافت حساب
func (s *StateDB) GetAccount(address common.Address) *Account {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if account, exists := s.accounts[address]; exists {
		return account
	}

	// ایجاد حساب جدید
	account := &Account{
		Address:     address,
		Balance:     big.NewInt(0),
		Nonce:       0,
		Code:        []byte{},
		CodeHash:    common.Hash{},
		Storage:     make(map[common.Hash]common.Hash),
		IsContract:  false,
		Stake:       big.NewInt(0),
		IsValidator: false,
		LastSeen:    0,
	}

	s.accounts[address] = account
	return account
}

// SetBalance تنظیم موجودی
func (s *StateDB) SetBalance(address common.Address, balance *big.Int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	account := s.GetAccount(address)
	account.Balance = new(big.Int).Set(balance)
	s.markStateRootDirty()
}

// GetBalance دریافت موجودی
func (s *StateDB) GetBalance(address common.Address) *big.Int {
	account := s.GetAccount(address)
	return new(big.Int).Set(account.Balance)
}

// SetNonce تنظیم nonce
func (s *StateDB) SetNonce(address common.Address, nonce uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	account := s.GetAccount(address)
	account.Nonce = nonce
	s.markStateRootDirty()
}

// GetNonce دریافت nonce
func (s *StateDB) GetNonce(address common.Address) uint64 {
	account := s.GetAccount(address)
	return account.Nonce
}

// SetCode تنظیم کد قرارداد
func (s *StateDB) SetCode(address common.Address, code []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()

	account := s.GetAccount(address)
	account.Code = code
	account.CodeHash = crypto.Keccak256Hash(code)
	account.IsContract = len(code) > 0
	s.markStateRootDirty()
}

// GetCode دریافت کد قرارداد
func (s *StateDB) GetCode(address common.Address) []byte {
	account := s.GetAccount(address)
	return account.Code
}

// SetStorage تنظیم storage
func (s *StateDB) SetStorage(address common.Address, key, value common.Hash) {
	s.mu.Lock()
	defer s.mu.Unlock()

	account := s.GetAccount(address)
	account.Storage[key] = value
	s.markStateRootDirty()
}

// GetStorage دریافت storage
func (s *StateDB) GetStorage(address common.Address, key common.Hash) common.Hash {
	account := s.GetAccount(address)
	return account.Storage[key]
}

// CreateContract ایجاد قرارداد جدید
func (s *StateDB) CreateContract(address common.Address, code []byte, creator common.Address, blockNumber uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// تنظیم کد
	s.SetCode(address, code)

	// ایجاد contract record
	contract := &Contract{
		Address:   address,
		Code:      code,
		Storage:   make(map[common.Hash]common.Hash),
		Balance:   big.NewInt(0),
		Nonce:     0,
		Creator:   creator,
		CreatedAt: blockNumber,
		UpdatedAt: blockNumber,
	}

	s.contracts[address] = contract
	s.markStateRootDirty()

	return nil
}

// GetContract دریافت قرارداد
func (s *StateDB) GetContract(address common.Address) *Contract {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.contracts[address]
}

// AddValidator اضافه کردن validator
func (s *StateDB) AddValidator(address common.Address, stake *big.Int, commission uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if stake.Cmp(s.config.MinStake) < 0 {
		return fmt.Errorf("insufficient stake: required %s, got %s", s.config.MinStake, stake)
	}

	account := s.GetAccount(address)
	account.IsValidator = true
	account.Stake = new(big.Int).Set(stake)

	validator := &ValidatorState{
		Address:    address,
		Stake:      new(big.Int).Set(stake),
		IsActive:   true,
		Commission: commission,
		Delegators: make(map[common.Address]*big.Int),
		TotalStake: new(big.Int).Set(stake),
		Rewards:    big.NewInt(0),
		LastReward: 0,
		LastSeen:   0,
	}

	s.validators[address] = validator
	s.stakes[address] = new(big.Int).Set(stake)
	s.markStateRootDirty()

	return nil
}

// RemoveValidator حذف validator
func (s *StateDB) RemoveValidator(address common.Address) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	account := s.GetAccount(address)
	account.IsValidator = false

	delete(s.validators, address)
	delete(s.stakes, address)
	s.markStateRootDirty()

	return nil
}

// GetValidator دریافت validator
func (s *StateDB) GetValidator(address common.Address) *ValidatorState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.validators[address]
}

// GetAllValidators دریافت تمام validators
func (s *StateDB) GetAllValidators() map[common.Address]*ValidatorState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[common.Address]*ValidatorState)
	for addr, validator := range s.validators {
		result[addr] = validator
	}

	return result
}

// Delegate delegation
func (s *StateDB) Delegate(delegator, validator common.Address, amount *big.Int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// بررسی وجود validator
	valState := s.validators[validator]
	if valState == nil || !valState.IsActive {
		return fmt.Errorf("validator not found or inactive")
	}

	// بررسی موجودی delegator
	delegatorAccount := s.GetAccount(delegator)
	if delegatorAccount.Balance.Cmp(amount) < 0 {
		return fmt.Errorf("insufficient balance")
	}

	// کاهش موجودی delegator
	delegatorAccount.Balance.Sub(delegatorAccount.Balance, amount)

	// افزایش delegation
	if valState.Delegators[delegator] == nil {
		valState.Delegators[delegator] = big.NewInt(0)
	}
	valState.Delegators[delegator].Add(valState.Delegators[delegator], amount)

	// به‌روزرسانی total stake
	valState.TotalStake.Add(valState.TotalStake, amount)

	// ثبت delegation
	if s.delegations[delegator] == nil {
		s.delegations[delegator] = make(map[common.Address]*big.Int)
	}
	s.delegations[delegator][validator] = new(big.Int).Set(amount)
	s.markStateRootDirty()

	return nil
}

// Undelegate لغو delegation
func (s *StateDB) Undelegate(delegator, validator common.Address, amount *big.Int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	valState := s.validators[validator]
	if valState == nil {
		return fmt.Errorf("validator not found")
	}

	delegation := valState.Delegators[delegator]
	if delegation == nil || delegation.Cmp(amount) < 0 {
		return fmt.Errorf("insufficient delegation")
	}

	// کاهش delegation
	delegation.Sub(delegation, amount)
	if delegation.Sign() == 0 {
		delete(valState.Delegators, delegator)
	}

	// کاهش total stake
	valState.TotalStake.Sub(valState.TotalStake, amount)

	// بازگرداندن موجودی
	delegatorAccount := s.GetAccount(delegator)
	delegatorAccount.Balance.Add(delegatorAccount.Balance, amount)
	s.markStateRootDirty()

	return nil
}

// CreateProposal ایجاد proposal جدید
func (s *StateDB) CreateProposal(id uint64, title, description string, creator common.Address, proposalType ProposalType, data []byte, startTime, endTime uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	proposal := &Proposal{
		ID:           id,
		Title:        title,
		Description:  description,
		Creator:      creator,
		Type:         proposalType,
		Data:         data,
		StartTime:    startTime,
		EndTime:      endTime,
		Executed:     false,
		VotesFor:     big.NewInt(0),
		VotesAgainst: big.NewInt(0),
		TotalVotes:   big.NewInt(0),
	}

	s.proposals[id] = proposal
	s.votes[id] = make(map[common.Address]*StateVote)
	s.markStateRootDirty()

	return nil
}

// Vote رأی دادن
func (s *StateDB) Vote(proposalID uint64, voter common.Address, choice VoteChoice, stake *big.Int, time uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	proposal := s.proposals[proposalID]
	if proposal == nil {
		return fmt.Errorf("proposal not found")
	}

	if time < proposal.StartTime || time > proposal.EndTime {
		return fmt.Errorf("voting period not active")
	}

	vote := &StateVote{
		Voter:    voter,
		Proposal: proposalID,
		Choice:   choice,
		Stake:    new(big.Int).Set(stake),
		Time:     time,
	}

	s.votes[proposalID][voter] = vote

	// به‌روزرسانی آمار
	switch choice {
	case VoteChoiceFor:
		proposal.VotesFor.Add(proposal.VotesFor, stake)
	case VoteChoiceAgainst:
		proposal.VotesAgainst.Add(proposal.VotesAgainst, stake)
	}
	proposal.TotalVotes.Add(proposal.TotalVotes, stake)
	s.markStateRootDirty()

	return nil
}

// GetProposal دریافت proposal
func (s *StateDB) GetProposal(id uint64) *Proposal {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.proposals[id]
}

// GetAllProposals دریافت تمام proposals
func (s *StateDB) GetAllProposals() map[uint64]*Proposal {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[uint64]*Proposal)
	for id, proposal := range s.proposals {
		result[id] = proposal
	}

	return result
}

// IntermediateRoot محاسبه state root
func (s *StateDB) IntermediateRoot(commit bool) common.Hash {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.stateRootDirty && s.stateRootCache != (common.Hash{}) {
		return s.stateRootCache
	}

	// محاسبه state root جدید
	stateRoot := s.calculateStateRoot()

	if commit {
		s.stateRootCache = stateRoot
		s.stateRootDirty = false
	}

	return stateRoot
}

// calculateStateRoot محاسبه state root
func (s *StateDB) calculateStateRoot() common.Hash {
	// جمع‌آوری تمام داده‌های state
	var stateData []interface{}

	// اضافه کردن accounts
	for addr, account := range s.accounts {
		accountData := map[string]interface{}{
			"address":      addr,
			"balance":      account.Balance.String(),
			"nonce":        account.Nonce,
			"code_hash":    account.CodeHash.Hex(),
			"is_contract":  account.IsContract,
			"stake":        account.Stake.String(),
			"is_validator": account.IsValidator,
		}
		stateData = append(stateData, accountData)
	}

	// اضافه کردن validators
	for addr, validator := range s.validators {
		validatorData := map[string]interface{}{
			"address":     addr,
			"stake":       validator.Stake.String(),
			"is_active":   validator.IsActive,
			"commission":  validator.Commission,
			"total_stake": validator.TotalStake.String(),
			"rewards":     validator.Rewards.String(),
		}
		stateData = append(stateData, validatorData)
	}

	// اضافه کردن contracts
	for addr, contract := range s.contracts {
		contractData := map[string]interface{}{
			"address":    addr,
			"code":       contract.Code,
			"balance":    contract.Balance.String(),
			"nonce":      contract.Nonce,
			"creator":    contract.Creator,
			"created_at": contract.CreatedAt,
		}
		stateData = append(stateData, contractData)
	}

	// اضافه کردن proposals
	for id, proposal := range s.proposals {
		proposalData := map[string]interface{}{
			"id":            id,
			"title":         proposal.Title,
			"creator":       proposal.Creator,
			"type":          proposal.Type,
			"votes_for":     proposal.VotesFor.String(),
			"votes_against": proposal.VotesAgainst.String(),
			"total_votes":   proposal.TotalVotes.String(),
			"executed":      proposal.Executed,
		}
		stateData = append(stateData, proposalData)
	}

	// محاسبه hash از تمام داده‌ها
	hasher := sha256.New()
	for _, data := range stateData {
		encoded, _ := rlp.EncodeToBytes(data)
		hasher.Write(encoded)
	}

	var hash common.Hash
	copy(hash[:], hasher.Sum(nil))
	return hash
}

// Commit ذخیره تغییرات
func (s *StateDB) Commit() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// در نسخه کامل، اینجا تغییرات به storage ذخیره می‌شود
	s.stateRootDirty = true
	return nil
}

// GetTotalStake دریافت کل stake
func (s *StateDB) GetTotalStake() *big.Int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := big.NewInt(0)
	for _, v := range s.validators {
		if v.IsActive {
			total.Add(total, v.Stake)
		}
	}
	return total
}

// markStateRootDirty علامت‌گذاری state root به عنوان dirty
func (s *StateDB) markStateRootDirty() {
	s.stateRootDirty = true
}

// GetStateStats آمار state
func (s *StateDB) GetStateStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := make(map[string]interface{})

	// آمار accounts
	stats["total_accounts"] = len(s.accounts)
	stats["total_contracts"] = len(s.contracts)
	stats["total_validators"] = len(s.validators)

	// آمار validators
	activeValidators := 0
	totalStake := big.NewInt(0)
	for _, validator := range s.validators {
		if validator.IsActive {
			activeValidators++
			totalStake.Add(totalStake, validator.Stake)
		}
	}
	stats["active_validators"] = activeValidators
	stats["total_stake"] = totalStake.String()

	// آمار proposals
	stats["total_proposals"] = len(s.proposals)
	executedProposals := 0
	for _, proposal := range s.proposals {
		if proposal.Executed {
			executedProposals++
		}
	}
	stats["executed_proposals"] = executedProposals

	// آمار memory optimization
	if s.memoryOptimizer != nil {
		memoryStats := s.memoryOptimizer.GetMemoryStats()
		for key, value := range memoryStats {
			stats["memory_"+key] = value
		}
	}

	return stats
}

// GetSINARBalance دریافت موجودی سینار
func (s *StateDB) GetSINARBalance(address common.Address) *big.Int {
	account := s.GetAccount(address)
	return new(big.Int).Set(account.Balance)
}

// SetSINARBalance تنظیم موجودی سینار
func (s *StateDB) SetSINARBalance(address common.Address, balance *big.Int) {
	account := s.GetAccount(address)
	account.Balance = new(big.Int).Set(balance)

	// به‌روزرسانی current supply
	s.sinarToken.CurrentSupply = s.calculateTotalSINARSupply()
}

// TransferSINAR انتقال سینار
func (s *StateDB) TransferSINAR(from, to common.Address, amount *big.Int) error {
	// بررسی موجودی
	fromBalance := s.GetSINARBalance(from)
	if fromBalance.Cmp(amount) < 0 {
		return fmt.Errorf("insufficient SINAR balance")
	}

	// کسر از فرستنده
	s.SetSINARBalance(from, new(big.Int).Sub(fromBalance, amount))

	// اضافه کردن به گیرنده
	toBalance := s.GetSINARBalance(to)
	s.SetSINARBalance(to, new(big.Int).Add(toBalance, amount))

	fmt.Printf("💰 SINAR Transfer: %s SINAR from %s to %s\n",
		amount.String(), from.Hex(), to.Hex())

	return nil
}

// MintSINAR ایجاد سینار جدید (فقط برای rewards)
func (s *StateDB) MintSINAR(to common.Address, amount *big.Int) error {
	// بررسی اینکه آیا از total supply تجاوز می‌کند
	newSupply := new(big.Int).Add(s.sinarToken.CurrentSupply, amount)
	if newSupply.Cmp(s.sinarToken.TotalSupply) > 0 {
		return fmt.Errorf("minting would exceed total supply")
	}

	// اضافه کردن به موجودی
	currentBalance := s.GetSINARBalance(to)
	s.SetSINARBalance(to, new(big.Int).Add(currentBalance, amount))

	// به‌روزرسانی current supply
	s.sinarToken.CurrentSupply = newSupply

	fmt.Printf("🪙 SINAR Minted: %s SINAR to %s\n", amount.String(), to.Hex())
	return nil
}

// BurnSINAR سوزاندن سینار
func (s *StateDB) BurnSINAR(from common.Address, amount *big.Int) error {
	// بررسی موجودی
	currentBalance := s.GetSINARBalance(from)
	if currentBalance.Cmp(amount) < 0 {
		return fmt.Errorf("insufficient SINAR balance for burning")
	}

	// کسر از موجودی
	s.SetSINARBalance(from, new(big.Int).Sub(currentBalance, amount))

	// کاهش current supply
	s.sinarToken.CurrentSupply = new(big.Int).Sub(s.sinarToken.CurrentSupply, amount)

	fmt.Printf("🔥 SINAR Burned: %s SINAR from %s\n", amount.String(), from.Hex())
	return nil
}

// GetSINARInfo دریافت اطلاعات ارز سینار
func (s *StateDB) GetSINARInfo() map[string]interface{} {
	return map[string]interface{}{
		"symbol":         s.sinarToken.Symbol,
		"decimals":       s.sinarToken.Decimals,
		"total_supply":   s.sinarToken.TotalSupply.String(),
		"current_supply": s.sinarToken.CurrentSupply.String(),
		"price_usd":      s.sinarToken.Price.String(),
		"creator":        s.sinarToken.Creator.Hex(),
		"created_at":     s.sinarToken.CreatedAt,
	}
}

// calculateTotalSINARSupply محاسبه کل موجودی سینار
func (s *StateDB) calculateTotalSINARSupply() *big.Int {
	total := big.NewInt(0)

	// جمع‌آوری موجودی تمام accounts
	for _, account := range s.accounts {
		total.Add(total, account.Balance)
	}

	return total
}

// GetSINARPrice دریافت قیمت سینار
func (s *StateDB) GetSINARPrice() *big.Float {
	return new(big.Float).Set(s.sinarToken.Price)
}

// SetSINARPrice تنظیم قیمت سینار
func (s *StateDB) SetSINARPrice(price *big.Float) {
	s.sinarToken.Price = new(big.Float).Set(price)
}

// InitializeSINARDistribution توزیع اولیه سینار
func (s *StateDB) InitializeSINARDistribution() error {
	fmt.Println("🚀 Initializing SINAR token distribution...")

	// محاسبه مقادیر توزیع
	totalSupply, _ := new(big.Int).SetString(SINAR_TOTAL_SUPPLY, 10)

	teamAmount := new(big.Int).Mul(totalSupply, big.NewInt(SINAR_TEAM_PERCENTAGE))
	teamAmount.Div(teamAmount, big.NewInt(100))

	ecosystemAmount := new(big.Int).Mul(totalSupply, big.NewInt(SINAR_ECOSYSTEM_PERCENTAGE))
	ecosystemAmount.Div(ecosystemAmount, big.NewInt(100))

	validatorAmount := new(big.Int).Mul(totalSupply, big.NewInt(SINAR_VALIDATORS_PERCENTAGE))
	validatorAmount.Div(validatorAmount, big.NewInt(100))

	liquidityAmount := new(big.Int).Mul(totalSupply, big.NewInt(SINAR_LIQUIDITY_PERCENTAGE))
	liquidityAmount.Div(liquidityAmount, big.NewInt(100))

	communityAmount := new(big.Int).Mul(totalSupply, big.NewInt(SINAR_COMMUNITY_PERCENTAGE))
	communityAmount.Div(communityAmount, big.NewInt(100))

	reserveAmount := new(big.Int).Mul(totalSupply, big.NewInt(SINAR_RESERVE_PERCENTAGE))
	reserveAmount.Div(reserveAmount, big.NewInt(100))

	// توزیع به Team Wallets
	distribution := NewInitialDistribution()
	s.distributeToWallets(distribution.TeamWallets, teamAmount, "Team")

	// توزیع به Ecosystem Wallets
	s.distributeToWallets(distribution.EcosystemWallets, ecosystemAmount, "Ecosystem")

	// توزیع به Validator Wallets
	s.distributeToWallets(distribution.ValidatorWallets, validatorAmount, "Validators")

	// توزیع به Liquidity Wallets
	s.distributeToWallets(distribution.LiquidityWallets, liquidityAmount, "Liquidity")

	// توزیع به Community Wallets
	s.distributeToWallets(distribution.CommunityWallets, communityAmount, "Community")

	// توزیع به Reserve Wallets
	s.distributeToWallets(distribution.ReserveWallets, reserveAmount, "Reserve")

	// به‌روزرسانی current supply
	s.sinarToken.CurrentSupply = s.calculateTotalSINARSupply()

	fmt.Printf("✅ SINAR distribution completed! Total distributed: %s SINAR\n", s.sinarToken.CurrentSupply.String())
	return nil
}

// distributeToWallets توزیع سینار به گروهی از آدرس‌ها
func (s *StateDB) distributeToWallets(wallets []common.Address, totalAmount *big.Int, category string) {
	if len(wallets) == 0 {
		return
	}

	// تقسیم مساوی بین تمام آدرس‌ها
	amountPerWallet := new(big.Int).Div(totalAmount, big.NewInt(int64(len(wallets))))
	remainder := new(big.Int).Mod(totalAmount, big.NewInt(int64(len(wallets))))

	for i, wallet := range wallets {
		amount := new(big.Int).Set(amountPerWallet)

		// اضافه کردن باقیمانده به آخرین آدرس
		if i == len(wallets)-1 {
			amount.Add(amount, remainder)
		}

		// تنظیم موجودی
		s.SetSINARBalance(wallet, amount)

		fmt.Printf("💰 %s Wallet %d: %s SINAR to %s\n",
			category, i+1, amount.String(), wallet.Hex())
	}
}

// GetDistributionInfo دریافت اطلاعات توزیع
func (s *StateDB) GetDistributionInfo() map[string]interface{} {
	distribution := NewInitialDistribution()

	info := map[string]interface{}{
		"team_wallets":      len(distribution.TeamWallets),
		"ecosystem_wallets": len(distribution.EcosystemWallets),
		"validator_wallets": len(distribution.ValidatorWallets),
		"liquidity_wallets": len(distribution.LiquidityWallets),
		"community_wallets": len(distribution.CommunityWallets),
		"reserve_wallets":   len(distribution.ReserveWallets),

		"team_percentage":      SINAR_TEAM_PERCENTAGE,
		"ecosystem_percentage": SINAR_ECOSYSTEM_PERCENTAGE,
		"validator_percentage": SINAR_VALIDATORS_PERCENTAGE,
		"liquidity_percentage": SINAR_LIQUIDITY_PERCENTAGE,
		"community_percentage": SINAR_COMMUNITY_PERCENTAGE,
		"reserve_percentage":   SINAR_RESERVE_PERCENTAGE,
	}

	// محاسبه مقادیر واقعی
	totalSupply, _ := new(big.Int).SetString(SINAR_TOTAL_SUPPLY, 10)

	teamAmount := new(big.Int).Mul(totalSupply, big.NewInt(SINAR_TEAM_PERCENTAGE))
	teamAmount.Div(teamAmount, big.NewInt(100))

	ecosystemAmount := new(big.Int).Mul(totalSupply, big.NewInt(SINAR_ECOSYSTEM_PERCENTAGE))
	ecosystemAmount.Div(ecosystemAmount, big.NewInt(100))

	validatorAmount := new(big.Int).Mul(totalSupply, big.NewInt(SINAR_VALIDATORS_PERCENTAGE))
	validatorAmount.Div(validatorAmount, big.NewInt(100))

	liquidityAmount := new(big.Int).Mul(totalSupply, big.NewInt(SINAR_LIQUIDITY_PERCENTAGE))
	liquidityAmount.Div(liquidityAmount, big.NewInt(100))

	communityAmount := new(big.Int).Mul(totalSupply, big.NewInt(SINAR_COMMUNITY_PERCENTAGE))
	communityAmount.Div(communityAmount, big.NewInt(100))

	reserveAmount := new(big.Int).Mul(totalSupply, big.NewInt(SINAR_RESERVE_PERCENTAGE))
	reserveAmount.Div(reserveAmount, big.NewInt(100))

	info["team_amount"] = teamAmount.String()
	info["ecosystem_amount"] = ecosystemAmount.String()
	info["validator_amount"] = validatorAmount.String()
	info["liquidity_amount"] = liquidityAmount.String()
	info["community_amount"] = communityAmount.String()
	info["reserve_amount"] = reserveAmount.String()

	return info
}

// UpdateFinalizedEvent به‌روزرسانی event نهایی شده
func (s *StateDB) UpdateFinalizedEvent(eventID EventID, round uint64) {
	// ذخیره اطلاعات event نهایی شده
	s.mu.Lock()
	defer s.mu.Unlock()

	// در نسخه کامل، این اطلاعات در database ذخیره می‌شود
	// فعلاً فقط log می‌کنیم
}

// GetValidatorStake دریافت stake یک validator
func (s *StateDB) GetValidatorStake(validatorID string) *big.Int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// در نسخه کامل، stake از validator set گرفته می‌شود
	// فعلاً مقدار پیش‌فرض برمی‌گردانیم
	return big.NewInt(1000000) // 1M tokens default
}

// MemoryOptimizer بهینه‌سازی حافظه
type MemoryOptimizer struct {
	eventPool       *sync.Pool
	blockPool       *sync.Pool
	transactionPool *sync.Pool
	gcThreshold     int
	gcInterval      time.Duration
	lastGC          time.Time
	mu              sync.RWMutex
}

// NewMemoryOptimizer ایجاد MemoryOptimizer جدید
func NewMemoryOptimizer() *MemoryOptimizer {
	return &MemoryOptimizer{
		eventPool: &sync.Pool{
			New: func() interface{} {
				return &Event{}
			},
		},
		blockPool: &sync.Pool{
			New: func() interface{} {
				return &Block{}
			},
		},
		transactionPool: &sync.Pool{
			New: func() interface{} {
				return &types.Transaction{}
			},
		},
		gcThreshold: 1000, // تعداد events قبل از GC
		gcInterval:  5 * time.Minute,
		lastGC:      time.Now(),
	}
}

// GetEvent از pool دریافت event
func (mo *MemoryOptimizer) GetEvent() *Event {
	return mo.eventPool.Get().(*Event)
}

// PutEvent بازگرداندن event به pool
func (mo *MemoryOptimizer) PutEvent(event *Event) {
	// پاک کردن event قبل از بازگرداندن
	event.Reset()
	mo.eventPool.Put(event)
}

// GetBlock از pool دریافت block
func (mo *MemoryOptimizer) GetBlock() *Block {
	return mo.blockPool.Get().(*Block)
}

// PutBlock بازگرداندن block به pool
func (mo *MemoryOptimizer) PutBlock(block *Block) {
	// پاک کردن block قبل از بازگرداندن
	block.Reset()
	mo.blockPool.Put(block)
}

// GetTransaction از pool دریافت transaction
func (mo *MemoryOptimizer) GetTransaction() *types.Transaction {
	return mo.transactionPool.Get().(*types.Transaction)
}

// PutTransaction بازگرداندن transaction به pool
func (mo *MemoryOptimizer) PutTransaction(tx *types.Transaction) {
	// پاک کردن transaction قبل از بازگرداندن
	// در نسخه کامل، این با Reset method ادغام می‌شود
	mo.transactionPool.Put(tx)
}

// CheckGC بررسی نیاز به garbage collection
func (mo *MemoryOptimizer) CheckGC(eventCount int) {
	mo.mu.Lock()
	defer mo.mu.Unlock()

	// بررسی بر اساس تعداد events
	if eventCount >= mo.gcThreshold {
		mo.triggerGC()
	}

	// بررسی بر اساس زمان
	if time.Since(mo.lastGC) >= mo.gcInterval {
		mo.triggerGC()
	}
}

// triggerGC اجرای garbage collection
func (mo *MemoryOptimizer) triggerGC() {
	// اجرای GC
	runtime.GC()

	// به‌روزرسانی زمان آخرین GC
	mo.lastGC = time.Now()

	fmt.Printf("🧹 Memory optimization: Garbage collection completed at %v\n", mo.lastGC)
}

// GetMemoryStats آمار حافظه
func (mo *MemoryOptimizer) GetMemoryStats() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]interface{}{
		"alloc":         m.Alloc,
		"total_alloc":   m.TotalAlloc,
		"sys":           m.Sys,
		"num_gc":        m.NumGC,
		"heap_alloc":    m.HeapAlloc,
		"heap_sys":      m.HeapSys,
		"heap_idle":     m.HeapIdle,
		"heap_inuse":    m.HeapInuse,
		"heap_released": m.HeapReleased,
		"heap_objects":  m.HeapObjects,
		"stack_inuse":   m.StackInuse,
		"stack_sys":     m.StackSys,
		"last_gc":       mo.lastGC,
		"gc_threshold":  mo.gcThreshold,
		"gc_interval":   mo.gcInterval,
	}
}

// Reset پاک کردن تمام pools
func (mo *MemoryOptimizer) Reset() {
	mo.mu.Lock()
	defer mo.mu.Unlock()

	// پاک کردن pools
	mo.eventPool = &sync.Pool{
		New: func() interface{} {
			return &Event{}
		},
	}

	mo.blockPool = &sync.Pool{
		New: func() interface{} {
			return &Block{}
		},
	}

	mo.transactionPool = &sync.Pool{
		New: func() interface{} {
			return &types.Transaction{}
		},
	}

	// اجرای GC
	runtime.GC()

	fmt.Println("🧹 Memory optimization: All pools reset")
}
