package main

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
)

// StateDB مدیریت state مشابه Fantom
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

	// Configuration
	config *StateConfig
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
}

type Contract struct {
	Address   common.Address
	Code      []byte
	Storage   map[common.Hash]common.Hash
	Balance   *big.Int
	Nonce     uint64
	Creator   common.Address
	CreatedAt uint64
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
	MinStake        *big.Int
	ValidatorReward *big.Int
	DelegatorReward *big.Int
	MaxValidators   uint64
	MinValidators   uint64
}

// NewStateDB ایجاد StateDB جدید
func NewStateDB() *StateDB {
	config := &StateConfig{
		MinStake:        big.NewInt(1000000), // 1M tokens
		ValidatorReward: big.NewInt(100),     // 100 tokens
		DelegatorReward: big.NewInt(10),      // 10 tokens
		MaxValidators:   100,
		MinValidators:   4,
	}

	return &StateDB{
		stateDB:     nil, // فعلاً nil می‌گذاریم
		accounts:    make(map[common.Address]*Account),
		contracts:   make(map[common.Address]*Contract),
		validators:  make(map[common.Address]*ValidatorState),
		stakes:      make(map[common.Address]*big.Int),
		delegations: make(map[common.Address]map[common.Address]*big.Int),
		proposals:   make(map[uint64]*Proposal),
		votes:       make(map[uint64]map[common.Address]*StateVote),
		config:      config,
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
	account.CodeHash = common.BytesToHash(code)
	account.IsContract = len(code) > 0
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
	}

	s.contracts[address] = contract

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
	}

	s.validators[address] = validator
	s.stakes[address] = new(big.Int).Set(stake)

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
	}

	s.proposals[id] = proposal
	s.votes[id] = make(map[common.Address]*StateVote)

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
	// پیاده‌سازی محاسبه state root
	return common.Hash{}
}

// Commit ذخیره تغییرات
func (s *StateDB) Commit() error {
	// پیاده‌سازی commit
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
