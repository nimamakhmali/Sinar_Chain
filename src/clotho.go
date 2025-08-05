package main

import (
	"fmt"
	"sync"
)

// ClothoSelector انتخاب Clothos مشابه Fantom Opera
type ClothoSelector struct {
	dag *DAG
	mu  sync.RWMutex

	// Clotho tracking
	clothos map[uint64]map[EventID]*Event
	rounds  map[uint64]*RoundInfo

	// Selection criteria
	selectionCriteria map[string]interface{}
}

// NewClothoSelector ایجاد ClothoSelector جدید
func NewClothoSelector(dag *DAG) *ClothoSelector {
	return &ClothoSelector{
		dag:               dag,
		clothos:           make(map[uint64]map[EventID]*Event),
		rounds:            make(map[uint64]*RoundInfo),
		selectionCriteria: make(map[string]interface{}),
	}
}

// SelectClothos انتخاب Clothos برای یک round
func (cs *ClothoSelector) SelectClothos(round uint64) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// بررسی اینکه آیا قبلاً انتخاب شده
	if _, exists := cs.clothos[round]; exists {
		return nil
	}

	// دریافت famous witnesses این round
	famousWitnesses := cs.getFamousWitnesses(round)
	if len(famousWitnesses) == 0 {
		return fmt.Errorf("no famous witnesses found for round %d", round)
	}

	// انتخاب Clothos بر اساس الگوریتم Fantom Opera
	selectedClothos := cs.selectClothosFromWitnesses(famousWitnesses, round)

	// ذخیره Clothos انتخاب شده
	cs.clothos[round] = selectedClothos

	fmt.Printf("🎯 Selected %d Clothos for round %d\n", len(selectedClothos), round)
	return nil
}

// selectClothosFromWitnesses انتخاب Clothos از famous witnesses
func (cs *ClothoSelector) selectClothosFromWitnesses(witnesses map[EventID]*Event, round uint64) map[EventID]*Event {
	clothos := make(map[EventID]*Event)

	// الگوریتم انتخاب Clothos از Fantom Opera:
	// 1. هر famous witness که شرایط Clotho را داشته باشد انتخاب می‌شود
	// 2. شرایط: باید در round بعدی events داشته باشد که آن را ببینند
	// 3. باید consensus در مورد آن در round بعدی رسیده باشد

	for witnessID, witness := range witnesses {
		if cs.isClothoCandidate(witness, round) {
			clothos[witnessID] = witness
		}
	}

	return clothos
}

// isClothoCandidate بررسی اینکه آیا یک witness می‌تواند Clotho باشد
func (cs *ClothoSelector) isClothoCandidate(witness *Event, round uint64) bool {
	// شرط 1: باید famous باشد
	if !cs.isFamous(witness, round) {
		return false
	}

	// شرط 2: باید در round بعدی events داشته باشد که آن را ببینند
	if !cs.hasEventsInNextRound(witness, round) {
		return false
	}

	// شرط 3: باید consensus در round بعدی رسیده باشد
	if !cs.hasConsensusInNextRound(witness, round) {
		return false
	}

	// شرط 4: باید شرایط visibility را داشته باشد
	if !cs.hasProperVisibility(witness, round) {
		return false
	}

	return true
}

// isFamous بررسی اینکه آیا یک witness famous است
func (cs *ClothoSelector) isFamous(witness *Event, round uint64) bool {
	// بررسی از fame voting
	fameVoting := cs.dag.GetFameVoting()
	if fameVoting == nil {
		return false
	}

	// بررسی اینکه آیا در famous witnesses قرار دارد
	famousWitnesses := fameVoting.getFamousWitnesses(round)
	_, isFamous := famousWitnesses[witness.Hash()]
	return isFamous
}

// hasEventsInNextRound بررسی اینکه آیا در round بعدی events وجود دارد که این witness را ببینند
func (cs *ClothoSelector) hasEventsInNextRound(witness *Event, round uint64) bool {
	nextRound := round + 1

	cs.dag.mu.RLock()
	defer cs.dag.mu.RUnlock()

	// شمارش events در round بعدی که این witness را می‌بینند
	seeCount := 0
	for eventID, event := range cs.dag.Events {
		if event.Round == nextRound {
			if cs.canSee(eventID, witness.Hash()) {
				seeCount++
			}
		}
	}

	// باید حداقل 2/3 events در round بعدی این witness را ببینند
	totalEventsInNextRound := 0
	for _, event := range cs.dag.Events {
		if event.Round == nextRound {
			totalEventsInNextRound++
		}
	}

	if totalEventsInNextRound == 0 {
		return false
	}

	threshold := (totalEventsInNextRound * 2) / 3
	return seeCount > threshold
}

// hasConsensusInNextRound بررسی اینکه آیا در round بعدی consensus رسیده
func (cs *ClothoSelector) hasConsensusInNextRound(witness *Event, round uint64) bool {
	nextRound := round + 1

	// بررسی اینکه آیا در round بعدی consensus رسیده
	// این بر اساس fame voting در round بعدی است
	fameVoting := cs.dag.GetFameVoting()
	if fameVoting == nil {
		return false
	}

	// بررسی اینکه آیا round بعدی decided شده
	decidedRounds := fameVoting.decidedRounds
	return decidedRounds[nextRound]
}

// hasProperVisibility بررسی visibility مناسب
func (cs *ClothoSelector) hasProperVisibility(witness *Event, round uint64) bool {
	// بررسی اینکه آیا witness visibility مناسب دارد
	// باید توسط اکثر events در round‌های بعدی دیده شود

	cs.dag.mu.RLock()
	defer cs.dag.mu.RUnlock()

	// شمارش events در round‌های بعدی که این witness را می‌بینند
	seeCount := 0
	totalCount := 0

	for _, event := range cs.dag.Events {
		if event.Round > round {
			totalCount++
			if cs.canSee(event.Hash(), witness.Hash()) {
				seeCount++
			}
		}
	}

	if totalCount == 0 {
		return false
	}

	// باید حداقل 2/3 events در round‌های بعدی این witness را ببینند
	threshold := (totalCount * 2) / 3
	return seeCount > threshold
}

// canSee بررسی اینکه آیا event A می‌تواند event B را ببیند
func (cs *ClothoSelector) canSee(eventA, eventB EventID) bool {
	// استفاده از الگوریتم canSee از DAG
	return cs.dag.IsAncestor(eventA, eventB)
}

// getFamousWitnesses دریافت famous witnesses یک round
func (cs *ClothoSelector) getFamousWitnesses(round uint64) map[EventID]*Event {
	fameVoting := cs.dag.GetFameVoting()
	if fameVoting == nil {
		return make(map[EventID]*Event)
	}

	return fameVoting.getFamousWitnesses(round)
}

// GetClothos دریافت Clothos یک round
func (cs *ClothoSelector) GetClothos(round uint64) map[EventID]*Event {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	if clothos, exists := cs.clothos[round]; exists {
		return clothos
	}

	return make(map[EventID]*Event)
}

// GetAllClothos دریافت تمام Clothos
func (cs *ClothoSelector) GetAllClothos() map[uint64]map[EventID]*Event {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	result := make(map[uint64]map[EventID]*Event)
	for round, clothos := range cs.clothos {
		result[round] = clothos
	}

	return result
}

// GetClothoStats آمار Clothos
func (cs *ClothoSelector) GetClothoStats() map[string]interface{} {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_rounds_with_clothos"] = len(cs.clothos)

	// آمار per round
	roundStats := make(map[uint64]map[string]interface{})
	for round, clothos := range cs.clothos {
		roundStats[round] = map[string]interface{}{
			"clotho_count": len(clothos),
			"clotho_ids":   cs.getClothoIDs(clothos),
		}
	}
	stats["round_stats"] = roundStats

	return stats
}

// getClothoIDs دریافت ID های Clothos
func (cs *ClothoSelector) getClothoIDs(clothos map[EventID]*Event) []string {
	ids := make([]string, 0, len(clothos))
	for clothoID := range clothos {
		ids = append(ids, fmt.Sprintf("%x", clothoID[:8])) // نمایش 8 بایت اول
	}
	return ids
}

// Reset بازنشانی برای تست
func (cs *ClothoSelector) Reset() {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.clothos = make(map[uint64]map[EventID]*Event)
	cs.rounds = make(map[uint64]*RoundInfo)
	cs.selectionCriteria = make(map[string]interface{})
}
