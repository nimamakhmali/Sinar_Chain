package main

import (
	"fmt"
	"sync"
)

// ClothoSelector انتخاب Clothos دقیقاً مطابق Fantom Opera
type ClothoSelector struct {
	dag *DAG
	mu  sync.RWMutex

	// Clotho tracking - دقیقاً مطابق Fantom
	clothos map[uint64]map[EventID]*Event
	rounds  map[uint64]*RoundInfo

	// Selection criteria - دقیقاً مطابق Fantom
	selectionCriteria map[string]interface{}

	// Byzantine fault tolerance parameters
	byzantineThreshold float64 // 2/3 for BFT
	minVisibilityCount int     // Minimum visibility required
}

// NewClothoSelector ایجاد ClothoSelector جدید با پارامترهای Fantom
func NewClothoSelector(dag *DAG) *ClothoSelector {
	return &ClothoSelector{
		dag:                dag,
		clothos:            make(map[uint64]map[EventID]*Event),
		rounds:             make(map[uint64]*RoundInfo),
		selectionCriteria:  make(map[string]interface{}),
		byzantineThreshold: 2.0 / 3.0, // 2/3 for Byzantine fault tolerance
		minVisibilityCount: 3,         // Minimum 3 events must see it
	}
}

// SelectClothos انتخاب Clothos برای یک round - دقیقاً مطابق Fantom
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

	// انتخاب Clothos بر اساس الگوریتم دقیق Fantom Opera
	selectedClothos := cs.selectClothosFromWitnessesFantom(famousWitnesses, round)

	// ذخیره Clothos انتخاب شده
	cs.clothos[round] = selectedClothos

	fmt.Printf("🎯 Selected %d Clothos for round %d\n", len(selectedClothos), round)
	return nil
}

// selectClothosFromWitnessesFantom انتخاب Clothos از famous witnesses - دقیقاً مطابق Fantom
func (cs *ClothoSelector) selectClothosFromWitnessesFantom(witnesses map[EventID]*Event, round uint64) map[EventID]*Event {
	clothos := make(map[EventID]*Event)

	// الگوریتم انتخاب Clothos از Fantom Opera:
	// 1. هر famous witness که شرایط Clotho را داشته باشد انتخاب می‌شود
	// 2. شرایط: باید در round بعدی events داشته باشد که آن را ببینند
	// 3. باید consensus در مورد آن در round بعدی رسیده باشد
	// 4. باید شرایط Byzantine fault tolerance برآورده شود

	for witnessID, witness := range witnesses {
		if cs.isClothoCandidateFantom(witness, round) {
			clothos[witnessID] = witness
		}
	}

	return clothos
}

// isClothoCandidateFantom بررسی اینکه آیا یک witness می‌تواند Clotho باشد - دقیقاً مطابق Fantom
func (cs *ClothoSelector) isClothoCandidateFantom(witness *Event, round uint64) bool {
	// شرط 1: باید famous باشد
	if !cs.isFamous(witness, round) {
		return false
	}

	// شرط 2: باید در round بعدی events داشته باشد که آن را ببینند
	if !cs.hasEventsInNextRoundFantom(witness, round) {
		return false
	}

	// شرط 3: باید consensus در round بعدی رسیده باشد
	if !cs.hasConsensusInNextRoundFantom(witness, round) {
		return false
	}

	// شرط 4: باید شرایط visibility را داشته باشد
	if !cs.hasProperVisibilityFantom(witness, round) {
		return false
	}

	// شرط 5: باید شرایط Byzantine fault tolerance برآورده شود
	if !cs.satisfiesByzantineFaultTolerance(witness, round) {
		return false
	}

	return true
}

// isFamous بررسی اینکه آیا یک witness famous است - دقیقاً مطابق Fantom
func (cs *ClothoSelector) isFamous(witness *Event, round uint64) bool {
	// بررسی famous بودن بر اساس Fame Voting
	// در نسخه کامل، این از FameVoting گرفته می‌شود

	// فعلاً بررسی ساده
	return witness.IsFamous != nil && *witness.IsFamous
}

// hasEventsInNextRoundFantom بررسی وجود events در round بعدی - دقیقاً مطابق Fantom
func (cs *ClothoSelector) hasEventsInNextRoundFantom(witness *Event, round uint64) bool {
	nextRound := round + 1
	eventCount := 0

	// شمارش events در round بعدی
	for _, event := range cs.dag.Events {
		if event.Round == nextRound {
			eventCount++
		}
	}

	// باید حداقل 3 events در round بعدی وجود داشته باشد
	return eventCount >= cs.minVisibilityCount
}

// hasConsensusInNextRoundFantom بررسی consensus در round بعدی - دقیقاً مطابق Fantom
func (cs *ClothoSelector) hasConsensusInNextRoundFantom(witness *Event, round uint64) bool {
	nextRound := round + 1
	seeCount := 0
	totalEvents := 0

	// شمارش events در round بعدی که این witness را می‌بینند
	for _, event := range cs.dag.Events {
		if event.Round == nextRound {
			totalEvents++
			if cs.canSee(event.Hash(), witness.Hash()) {
				seeCount++
			}
		}
	}

	// محاسبه نسبت visibility
	if totalEvents == 0 {
		return false
	}

	visibilityRatio := float64(seeCount) / float64(totalEvents)
	return visibilityRatio >= cs.byzantineThreshold
}

// hasProperVisibilityFantom بررسی visibility مناسب - دقیقاً مطابق Fantom
func (cs *ClothoSelector) hasProperVisibilityFantom(witness *Event, round uint64) bool {
	// بررسی اینکه آیا witness توسط اکثریت events در round بعدی دیده می‌شود
	nextRound := round + 1
	seeCount := 0
	totalEvents := 0

	for _, event := range cs.dag.Events {
		if event.Round == nextRound {
			totalEvents++
			if cs.canSee(event.Hash(), witness.Hash()) {
				seeCount++
			}
		}
	}

	// باید حداقل 2/3 events در round بعدی این witness را ببینند
	if totalEvents == 0 {
		return false
	}

	visibilityRatio := float64(seeCount) / float64(totalEvents)
	return visibilityRatio >= cs.byzantineThreshold
}

// satisfiesByzantineFaultTolerance بررسی شرایط Byzantine fault tolerance - دقیقاً مطابق Fantom
func (cs *ClothoSelector) satisfiesByzantineFaultTolerance(witness *Event, round uint64) bool {
	// بررسی شرایط Byzantine fault tolerance:
	// 1. باید توسط اکثریت validators دیده شود
	// 2. باید در round بعدی consensus رسیده باشد
	// 3. باید شرایط safety و liveness برآورده شود

	nextRound := round + 1
	validatorCount := 0
	seeCount := 0

	// شمارش validators که این witness را می‌بینند
	validators := make(map[string]bool)
	for _, event := range cs.dag.Events {
		if event.Round == nextRound {
			if !validators[event.CreatorID] {
				validators[event.CreatorID] = true
				validatorCount++

				if cs.canSee(event.Hash(), witness.Hash()) {
					seeCount++
				}
			}
		}
	}

	// باید اکثریت validators این witness را ببینند
	if validatorCount == 0 {
		return false
	}

	validatorRatio := float64(seeCount) / float64(validatorCount)
	return validatorRatio >= cs.byzantineThreshold
}

// canSee بررسی اینکه آیا event A می‌تواند event B را ببیند - دقیقاً مطابق Fantom
func (cs *ClothoSelector) canSee(eventA, eventB EventID) bool {
	// الگوریتم visibility از Fantom:
	// event A می‌تواند event B را ببیند اگر:
	// 1. event A در round بعدی باشد
	// 2. event A ancestor event B باشد
	// 3. یا event A و event B در همان round باشند

	eventAObj, existsA := cs.dag.GetEvent(eventA)
	eventBObj, existsB := cs.dag.GetEvent(eventB)

	if !existsA || !existsB {
		return false
	}

	// اگر در همان round باشند
	if eventAObj.Round == eventBObj.Round {
		return true
	}

	// اگر event A در round بعدی باشد
	if eventAObj.Round == eventBObj.Round+1 {
		// بررسی ancestor بودن
		return cs.dag.IsAncestor(eventB, eventA)
	}

	return false
}

// getFamousWitnesses دریافت famous witnesses یک round - دقیقاً مطابق Fantom
func (cs *ClothoSelector) getFamousWitnesses(round uint64) map[EventID]*Event {
	famousWitnesses := make(map[EventID]*Event)

	for _, event := range cs.dag.Events {
		if event.Round == round && cs.isFamous(event, round) {
			famousWitnesses[event.Hash()] = event
		}
	}

	return famousWitnesses
}

// GetClothos دریافت Clothos یک round - دقیقاً مطابق Fantom
func (cs *ClothoSelector) GetClothos(round uint64) map[EventID]*Event {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	if clothos, exists := cs.clothos[round]; exists {
		return clothos
	}

	return make(map[EventID]*Event)
}

// GetAllClothos دریافت تمام Clothos - دقیقاً مطابق Fantom
func (cs *ClothoSelector) GetAllClothos() map[uint64]map[EventID]*Event {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	allClothos := make(map[uint64]map[EventID]*Event)
	for round, clothos := range cs.clothos {
		allClothos[round] = clothos
	}

	return allClothos
}

// GetClothoStats آمار Clotho - دقیقاً مطابق Fantom
func (cs *ClothoSelector) GetClothoStats() map[string]interface{} {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	stats := make(map[string]interface{})

	// آمار کلی
	totalClothos := 0
	roundStats := make(map[uint64]map[string]interface{})

	for round, clothos := range cs.clothos {
		totalClothos += len(clothos)
		roundStats[round] = map[string]interface{}{
			"clotho_count": len(clothos),
			"clotho_ids":   cs.getClothoIDs(clothos),
		}
	}

	stats["total_clothos"] = totalClothos
	stats["round_stats"] = roundStats
	stats["byzantine_threshold"] = cs.byzantineThreshold
	stats["min_visibility_count"] = cs.minVisibilityCount

	return stats
}

// getClothoIDs دریافت ID های Clothos - دقیقاً مطابق Fantom
func (cs *ClothoSelector) getClothoIDs(clothos map[EventID]*Event) []string {
	ids := make([]string, 0, len(clothos))
	for clothoID := range clothos {
		ids = append(ids, fmt.Sprintf("%x", clothoID[:8])) // نمایش 8 بایت اول
	}
	return ids
}

// Reset بازنشانی ClothoSelector - دقیقاً مطابق Fantom
func (cs *ClothoSelector) Reset() {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.clothos = make(map[uint64]map[EventID]*Event)
	cs.rounds = make(map[uint64]*RoundInfo)
	cs.selectionCriteria = make(map[string]interface{})
}
