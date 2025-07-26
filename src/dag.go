package main

import (
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/core/types"
)

type DAG struct {
	Events map[EventID]*Event
	Rounds RoundTable
	Votes  VoteRecord
	mu     sync.RWMutex
}

func NewDAG() *DAG {
	return &DAG{
		Events: make(map[EventID]*Event),
		Rounds: make(RoundTable),
		Votes:  make(VoteRecord),
	}
}

// AddEvent اضافه‌کردن یک رویداد به DAG
func (d *DAG) AddEvent(e *Event) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	id := e.Hash()
	if _, exists := d.Events[id]; exists {
		return errors.New("event already exists")
	}
	d.Events[id] = e
	d.AssignRound(e)
	return nil
}

func (d *DAG) GetEvent(id EventID) (*Event, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	e, ok := d.Events[id]
	return e, ok
}

func (d *DAG) GetParents(e *Event) ([]*Event, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	parents := make([]*Event, 0, len(e.Parents))
	for _, pid := range e.Parents {
		parent, ok := d.Events[pid]
		if !ok {
			return nil, errors.New("parent not found in DAG")
		}
		parents = append(parents, parent)
	}
	return parents, nil
}

func (d *DAG) IsAncestor(aID, bID EventID) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	visited := make(map[EventID]bool)
	var dfs func(EventID) bool

	dfs = func(current EventID) bool {
		if current == aID {
			return true
		}
		if visited[current] {
			return false
		}
		visited[current] = true
		eventObj, ok := d.Events[current]
		if !ok {
			return false
		}
		for _, parentID := range eventObj.Parents {
			if dfs(parentID) {
				return true
			}
		}
		return false
	}

	return dfs(bID)
}

func (d *DAG) CreateAndAddEvent(creatorID string, parents []EventID, epoch, lamport uint64, txs types.Transactions) (*Event, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	height := uint64(0)
	for _, pid := range parents {
		if parent, ok := d.Events[pid]; ok {
			if parent.Height > height {
				height = parent.Height
			}
		}
	}
	height++

	e := NewEvent(creatorID, parents, epoch, lamport, txs, height)
	id := e.Hash()

	if _, exists := d.Events[id]; exists {
		return nil, errors.New("event already exists")
	}
	d.Events[id] = e
	d.AssignRound(e)
	return e, nil
}
