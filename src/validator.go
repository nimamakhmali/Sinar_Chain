package main

import (
	"crypto/ecdsa"
	"fmt"
	"sync"
	"time"
)

// Validator نماینده شبکه
type Validator struct {
	ID         string
	Address    string
	PublicKey  *ecdsa.PublicKey
	PrivateKey *ecdsa.PrivateKey
	Stake      uint64
	IsActive   bool
	LastSeen   time.Time
}

// ValidatorSet مجموعه تمام validatorها
type ValidatorSet struct {
	validators map[string]*Validator
	mu         sync.RWMutex
}

func NewValidatorSet() *ValidatorSet {
	return &ValidatorSet{
		validators: make(map[string]*Validator),
	}
}

func NewValidator(id string, privKey *ecdsa.PrivateKey, stake uint64) *Validator {
	return &Validator{
		ID:         id,
		PublicKey:  &privKey.PublicKey,
		PrivateKey: privKey,
		Stake:      stake,
		IsActive:   true,
		LastSeen:   time.Now(),
	}
}

func (vs *ValidatorSet) AddValidator(validator *Validator) error {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	if _, exists := vs.validators[validator.ID]; exists {
		return fmt.Errorf("validator %s already exists", validator.ID)
	}

	vs.validators[validator.ID] = validator
	return nil
}

func (vs *ValidatorSet) RemoveValidator(id string) error {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	if _, exists := vs.validators[id]; !exists {
		return fmt.Errorf("validator %s not found", id)
	}

	delete(vs.validators, id)
	return nil
}

func (vs *ValidatorSet) GetValidator(id string) (*Validator, bool) {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	validator, exists := vs.validators[id]
	return validator, exists
}

func (vs *ValidatorSet) GetAllValidators() []*Validator {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	validators := make([]*Validator, 0, len(vs.validators))
	for _, v := range vs.validators {
		validators = append(validators, v)
	}
	return validators
}

func (vs *ValidatorSet) GetActiveValidators() []*Validator {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	active := make([]*Validator, 0)
	for _, v := range vs.validators {
		if v.IsActive {
			active = append(active, v)
		}
	}
	return active
}

func (vs *ValidatorSet) UpdateValidatorStake(id string, newStake uint64) error {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	validator, exists := vs.validators[id]
	if !exists {
		return fmt.Errorf("validator %s not found", id)
	}

	validator.Stake = newStake
	return nil
}

func (vs *ValidatorSet) SetValidatorActive(id string, active bool) error {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	validator, exists := vs.validators[id]
	if !exists {
		return fmt.Errorf("validator %s not found", id)
	}

	validator.IsActive = active
	if active {
		validator.LastSeen = time.Now()
	}
	return nil
}

func (vs *ValidatorSet) GetTotalStake() uint64 {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	total := uint64(0)
	for _, v := range vs.validators {
		if v.IsActive {
			total += v.Stake
		}
	}
	return total
}

func (vs *ValidatorSet) GetValidatorByAddress(address string) (*Validator, bool) {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	for _, v := range vs.validators {
		if v.Address == address {
			return v, true
		}
	}
	return nil, false
}

// CleanupInactiveValidators حذف validatorهای غیرفعال
func (vs *ValidatorSet) CleanupInactiveValidators(timeout time.Duration) {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	now := time.Now()
	for id, validator := range vs.validators {
		if !validator.IsActive && now.Sub(validator.LastSeen) > timeout {
			delete(vs.validators, id)
		}
	}
}
