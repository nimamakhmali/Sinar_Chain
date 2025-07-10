package core

import (
	"errors"
	"sinar_chain/src/event"
	"sync"
)

// DAG نگهدارنده DAG از رویدادهاست
type DAG struct {
	Events map[EventID]*Event
	Rounds RoundTable
	Votes  VoteRecord
	mu     sync.RWMutex
}


// NewDAG ایجاد یک DAG جدید خالی
func NewDAG() *DAG {
	return &DAG{
		Events: make(map[EventID]*Event),
		Rounds: make(RoundTable),
		Votes:  make(VoteRecord), // 👈 اینجا مقداردهی کن
	}
}


// AddEvent اضافه‌کردن یک رویداد به DAG
func (d *DAG) AddEvent(e *event.Event) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	id := e.Hash()
	if _, exists := d.Events[id]; exists {
		return errors.New("event already exists")
	}
	d.Events[id] = e
	return nil
}

// GetEvent دریافت یک رویداد با ID
func (d *DAG) GetEvent(id event.EventID) (*event.Event, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	e, ok := d.Events[id]
	return e, ok
}

// GetParents دریافت والدهای یک رویداد
func (d *DAG) GetParents(e *event.Event) ([]*event.Event, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	parents := make([]*event.Event, 0, len(e.Parents))
	for _, pid := range e.Parents {
		parent, ok := d.Events[pid]
		if !ok {
			return nil, errors.New("parent not found in DAG")
		}
		parents = append(parents, parent)
	}
	return parents, nil
}

// IsAncestor بررسی اینکه a اجداد b هست یا نه
func (d *DAG) IsAncestor(aID, bID event.EventID) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	visited := make(map[event.EventID]bool)
	var dfs func(event.EventID) bool

	dfs = func(current event.EventID) bool {
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
