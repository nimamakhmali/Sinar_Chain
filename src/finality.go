package main

import (
	"sort"
)

// FinalityEngine مسئول نهایی‌سازی events و تبدیل Clothos به Atropos
type FinalityEngine struct {
	dag *DAG
}

// NewFinalityEngine ایجاد FinalityEngine جدید
func NewFinalityEngine(dag *DAG) *FinalityEngine {
	return &FinalityEngine{
		dag: dag,
	}
}

// FinalizeEvents نهایی‌سازی events و تبدیل Clothos به Atropos
func (fe *FinalityEngine) FinalizeEvents() {
	// برای هر round که Clothos دارد
	for round := range fe.dag.Rounds {
		clothos := fe.getClothos(round)
		if len(clothos) == 0 {
			continue
		}

		// نهایی‌سازی Clothos این round
		fe.finalizeClothosForRound(round, clothos)
	}
}

// finalizeClothosForRound نهایی‌سازی Clothos برای یک round
func (fe *FinalityEngine) finalizeClothosForRound(round uint64, clothos []*Event) {
	// برای هر Clotho، بررسی تبدیل به Atropos
	for _, clotho := range clothos {
		if clotho.Atropos != (EventID{}) {
			continue // قبلاً Atropos شده
		}

		// بررسی شرایط تبدیل به Atropos
		if fe.canBecomeAtropos(clotho, round) {
			fe.convertToAtropos(clotho, round)
		}
	}
}

// canBecomeAtropos بررسی شرایط تبدیل Clotho به Atropos
func (fe *FinalityEngine) canBecomeAtropos(clotho *Event, round uint64) bool {
	// شرط 1: باید Clotho باشد
	if !clotho.IsClotho {
		return false
	}

	// شرط 2: باید اکثریت famous witnesses از round+2 آن را ببینند
	nextRound := round + 2
	famousWitnessesNextRound := fe.getFamousWitnesses(nextRound)
	if len(famousWitnessesNextRound) == 0 {
		return false
	}

	// شمارش famous witnesses که این Clotho را می‌بینند
	seeCount := 0
	for _, witness := range famousWitnessesNextRound {
		if fe.dag.IsAncestor(clotho.Hash(), witness.Hash()) {
			seeCount++
		}
	}

	// باید اکثریت (2/3) آن را ببینند
	requiredCount := (2 * len(famousWitnessesNextRound)) / 3
	return seeCount > requiredCount
}

// convertToAtropos تبدیل Clotho به Atropos
func (fe *FinalityEngine) convertToAtropos(clotho *Event, round uint64) {
	// تبدیل به Atropos
	clotho.Atropos = clotho.Hash()
	clotho.RoundReceived = round + 2

	// محاسبه AtroposTime (median time از تمام witnesses)
	times := fe.calculateAtroposTime(clotho, round+2)
	clotho.AtroposTime = fe.median(times)

	// محاسبه MedianTime
	clotho.MedianTime = clotho.AtroposTime

	// اضافه کردن به round info
	fe.ensureRound(round + 2)
	fe.dag.Rounds[round+2].Atropos[clotho.Hash()] = clotho
}

// calculateAtroposTime محاسبه زمان‌های Atropos
func (fe *FinalityEngine) calculateAtroposTime(clotho *Event, round uint64) []uint64 {
	var times []uint64

	// جمع‌آوری timestamps از تمام witnesses که این Clotho را می‌بینند
	famousWitnesses := fe.getFamousWitnesses(round)
	for _, witness := range famousWitnesses {
		if fe.dag.IsAncestor(clotho.Hash(), witness.Hash()) {
			// استفاده از Lamport timestamp به عنوان زمان
			times = append(times, witness.Lamport)
		}
	}

	return times
}

// median محاسبه median از یک slice
func (fe *FinalityEngine) median(values []uint64) uint64 {
	if len(values) == 0 {
		return 0
	}

	// کپی کردن slice برای مرتب‌سازی
	sorted := make([]uint64, len(values))
	copy(sorted, values)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	// محاسبه median
	n := len(sorted)
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2
	}
	return sorted[n/2]
}

// getClothos دریافت Clothos یک round
func (fe *FinalityEngine) getClothos(round uint64) []*Event {
	roundInfo, exists := fe.dag.Rounds[round]
	if !exists {
		return nil
	}

	var clothos []*Event
	for _, clotho := range roundInfo.Clothos {
		clothos = append(clothos, clotho)
	}

	return clothos
}

// getFamousWitnesses دریافت famous witnesses یک round
func (fe *FinalityEngine) getFamousWitnesses(round uint64) []*Event {
	roundInfo, exists := fe.dag.Rounds[round]
	if !exists {
		return nil
	}

	var famousWitnesses []*Event
	for _, witness := range roundInfo.Witnesses {
		if witness.IsFamous != nil && *witness.IsFamous {
			famousWitnesses = append(famousWitnesses, witness)
		}
	}

	return famousWitnesses
}

// ensureRound اطمینان از وجود round
func (fe *FinalityEngine) ensureRound(r uint64) {
	if fe.dag.Rounds == nil {
		fe.dag.Rounds = make(RoundTable)
	}
	if _, exists := fe.dag.Rounds[r]; !exists {
		fe.dag.Rounds[r] = &RoundInfo{
			Witnesses: make(map[EventID]*Event),
			Roots:     make(map[EventID]*Event),
			Clothos:   make(map[EventID]*Event),
			Atropos:   make(map[EventID]*Event),
		}
	}
}

// GetAtropos دریافت Atropos یک round
func (fe *FinalityEngine) GetAtropos(round uint64) []*Event {
	roundInfo, exists := fe.dag.Rounds[round]
	if !exists {
		return nil
	}

	var atropos []*Event
	for _, atro := range roundInfo.Atropos {
		atropos = append(atropos, atro)
	}

	return atropos
}

// GetFinalizedEvents دریافت تمام events نهایی شده
func (fe *FinalityEngine) GetFinalizedEvents() []*Event {
	var finalizedEvents []*Event

	for _, event := range fe.dag.Events {
		if event.Atropos != (EventID{}) {
			finalizedEvents = append(finalizedEvents, event)
		}
	}

	return finalizedEvents
}

// IsFinalized بررسی اینکه آیا یک event نهایی شده است
func (fe *FinalityEngine) IsFinalized(eventID EventID) bool {
	event, exists := fe.dag.GetEvent(eventID)
	if !exists {
		return false
	}
	return event.Atropos != (EventID{})
}

// GetFinalizedEventsInRange دریافت events نهایی شده در یک بازه زمانی
func (fe *FinalityEngine) GetFinalizedEventsInRange(fromTime, toTime uint64) []*Event {
	var events []*Event

	for _, event := range fe.dag.Events {
		if event.Atropos != (EventID{}) &&
			event.AtroposTime >= fromTime &&
			event.AtroposTime <= toTime {
			events = append(events, event)
		}
	}

	return events
}

// GetFinalityStats آمار نهایی‌سازی
func (fe *FinalityEngine) GetFinalityStats() map[string]interface{} {
	stats := make(map[string]interface{})

	totalFinalized := 0
	finalizedByRound := make(map[uint64]int)
	finalizedByCreator := make(map[string]int)

	for _, event := range fe.dag.Events {
		if event.Atropos != (EventID{}) {
			totalFinalized++
			finalizedByRound[event.RoundReceived]++
			finalizedByCreator[event.CreatorID]++
		}
	}

	stats["total_finalized"] = totalFinalized
	stats["finalized_by_round"] = finalizedByRound
	stats["finalized_by_creator"] = finalizedByCreator

	return stats
}
