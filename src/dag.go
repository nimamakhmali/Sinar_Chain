package main

import (
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/core/types"
)

type DAG struct {
	Events map[EventID]*Event
	Rounds RoundTable
	Votes  map[string]*Vote
	mu     sync.RWMutex
}

func NewDAG() *DAG {
	return &DAG{
		Events: make(map[EventID]*Event),
		Rounds: make(RoundTable),
		Votes:  make(map[string]*Vote),
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

// GetFameVoting دریافت FameVoting instance
func (d *DAG) GetFameVoting() *FameVoting {
	// در نسخه کامل، این از consensus engine گرفته می‌شود
	// فعلاً یک instance جدید برمی‌گردانیم
	return NewFameVoting(d)
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

// AssignRound محاسبه‌ی round، witness بودن و root بودن برای یک event
func (d *DAG) AssignRound(e *Event) {
	parents, err := d.GetParents(e)
	if err != nil {
		return
	}

	// اگر هیچ پدر نداره، یعنی genesis
	if len(parents) == 0 {
		e.Round = 0
		e.IsRoot = true
		d.ensureRound(e.Round)
		d.Rounds[e.Round].Witnesses[e.Hash()] = e
		d.Rounds[e.Round].Roots[e.Hash()] = e
		return
	}

	// حداکثر round والدها رو پیدا کن
	maxParentRound := uint64(0)
	for _, p := range parents {
		if p.Round > maxParentRound {
			maxParentRound = p.Round
		}
	}
	e.Round = maxParentRound

	// بررسی اینکه آیا همه والدها Root بودن → round++
	allParentsAreRoot := true
	for _, p := range parents {
		if !p.IsRoot {
			allParentsAreRoot = false
			break
		}
	}
	if allParentsAreRoot {
		e.Round += 1
	}

	// بررسی Witness بودن
	isWitness := d.isWitness(e)
	if isWitness {
		e.IsRoot = true
		d.ensureRound(e.Round)
		d.Rounds[e.Round].Witnesses[e.Hash()] = e
		d.Rounds[e.Round].Roots[e.Hash()] = e
	}
}

// isWitness بررسی اینکه آیا event یک witness است
func (d *DAG) isWitness(e *Event) bool {
	// اولین event از این creator در این round
	first := true
	for _, other := range d.Events {
		if other.CreatorID == e.CreatorID &&
			other.Round == e.Round &&
			other.Lamport < e.Lamport {
			first = false
			break
		}
	}
	return first
}

// ensureRound اطمینان از وجود round
func (d *DAG) ensureRound(r uint64) {
	if d.Rounds == nil {
		d.Rounds = make(RoundTable)
	}
	if _, exists := d.Rounds[r]; !exists {
		d.Rounds[r] = &RoundInfo{
			Witnesses: make(map[EventID]*Event),
			Roots:     make(map[EventID]*Event),
			Clothos:   make(map[EventID]*Event),
			Atropos:   make(map[EventID]*Event),
		}
	}
}
