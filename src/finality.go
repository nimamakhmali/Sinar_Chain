package main

import (
	"math/big"
	"sort"
	"sync"
	"time"
)

// FinalityEngine مسئول نهایی‌سازی events و تبدیل Clothos به Atropos
type FinalityEngine struct {
	dag            *DAG
	clothoSelector *ClothoSelector
	fameVoting     *FameVoting
	cacheManager   *CacheManager
	mu             sync.RWMutex
}

// FinalityInfo اطلاعات نهائی‌سازی
type FinalityInfo struct {
	EventID          EventID
	Round            uint64
	AtroposTime      uint64
	MedianTime       uint64
	IsFinalized      bool
	FinalizationTime time.Time
}

// TimeConsensus اطلاعات اجماع زمانی
type TimeConsensus struct {
	EventID          EventID
	WitnessTimes     []uint64
	MedianTime       uint64
	ConsensusReached bool
}

// NewFinalityEngine ایجاد FinalityEngine جدید
func NewFinalityEngine(dag *DAG) *FinalityEngine {
	return &FinalityEngine{
		dag:            dag,
		clothoSelector: NewClothoSelector(dag),
		fameVoting:     NewFameVoting(dag),
		cacheManager:   NewCacheManager(1000),
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

	// شرط 2: باید famous باشد
	if clotho.IsFamous == nil || !*clotho.IsFamous {
		return false
	}

	// شرط 3: باید اکثریت famous witnesses از round+2 آن را ببینند
	nextRound := round + 2
	famousWitnessesNextRound := fe.getFamousWitnesses(nextRound)
	if len(famousWitnessesNextRound) == 0 {
		return false
	}

	// شمارش famous witnesses که این Clotho را می‌بینند
	seeCount := 0
	totalFamousWitnesses := len(famousWitnessesNextRound)

	for _, witness := range famousWitnessesNextRound {
		if fe.dag.IsAncestor(clotho.Hash(), witness.Hash()) {
			seeCount++
		}
	}

	// باید اکثریت (2/3) آن را ببینند (Byzantine fault tolerance)
	requiredCount := (2 * totalFamousWitnesses) / 3
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

	// ثبت زمان نهائی‌سازی
	clotho.AtroposTime = uint64(time.Now().UnixNano() / 1000000) // milliseconds
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
	fe.mu.RLock()
	defer fe.mu.RUnlock()

	// بررسی cache manager
	if cachedData, exists := fe.cacheManager.GetConsensusCache(round); exists {
		if clothosData, ok := cachedData["clothos"]; ok {
			if clothos, ok := clothosData.([]*Event); ok {
				return clothos
			}
		}
	}

	roundInfo, exists := fe.dag.Rounds[round]
	if !exists {
		return nil
	}

	var clothos []*Event
	for _, clotho := range roundInfo.Clothos {
		clothos = append(clothos, clotho)
	}

	// ذخیره در cache manager
	cacheData := map[string]interface{}{
		"clothos": clothos,
	}
	fe.cacheManager.SetConsensusCache(round, cacheData)

	return clothos
}

// getFamousWitnesses دریافت famous witnesses یک round
func (fe *FinalityEngine) getFamousWitnesses(round uint64) []*Event {
	fe.mu.RLock()
	defer fe.mu.RUnlock()

	// بررسی cache manager
	if cachedData, exists := fe.cacheManager.GetConsensusCache(round); exists {
		if witnessesData, ok := cachedData["famous_witnesses"]; ok {
			if witnesses, ok := witnessesData.([]*Event); ok {
				return witnesses
			}
		}
	}

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

	// ذخیره در cache manager
	cacheData := map[string]interface{}{
		"famous_witnesses": famousWitnesses,
	}
	fe.cacheManager.SetConsensusCache(round, cacheData)

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

// GetFinalizedEventsByStake دریافت events نهایی شده بر اساس stake
func (fe *FinalityEngine) GetFinalizedEventsByStake(minStake *big.Int) []*Event {
	var events []*Event

	for _, event := range fe.dag.Events {
		if event.Atropos != (EventID{}) {
			// در نسخه کامل، stake از validator set گرفته می‌شود
			stake := big.NewInt(1000000) // 1M tokens default
			if stake.Cmp(minStake) >= 0 {
				events = append(events, event)
			}
		}
	}

	return events
}

// GetFinalizedEventsByCreator دریافت events نهایی شده یک creator
func (fe *FinalityEngine) GetFinalizedEventsByCreator(creatorID string) []*Event {
	var events []*Event

	for _, event := range fe.dag.Events {
		if event.Atropos != (EventID{}) && event.CreatorID == creatorID {
			events = append(events, event)
		}
	}

	return events
}

// ValidateFinality اعتبارسنجی نهائی‌سازی
func (fe *FinalityEngine) ValidateFinality(event *Event, round uint64) bool {
	// بررسی شرایط اعتبارسنجی
	if event.Atropos == (EventID{}) {
		return false
	}

	// بررسی Clotho بودن
	if !event.IsClotho {
		return false
	}

	// بررسی famous بودن
	if event.IsFamous == nil || !*event.IsFamous {
		return false
	}

	// بررسی round assignment
	if event.RoundReceived != round+2 {
		return false
	}

	// بررسی visibility conditions
	nextRound := round + 2
	famousWitnessesNextRound := fe.getFamousWitnesses(nextRound)
	seeCount := 0

	for _, witness := range famousWitnessesNextRound {
		if fe.dag.IsAncestor(event.Hash(), witness.Hash()) {
			seeCount++
		}
	}

	requiredCount := (2 * len(famousWitnessesNextRound)) / 3
	return seeCount > requiredCount
}

// GetTimeConsensus محاسبه اجماع زمانی
func (fe *FinalityEngine) GetTimeConsensus(event *Event) *TimeConsensus {
	var witnessTimes []uint64

	// جمع‌آوری زمان‌های شاهدان
	for _, otherEvent := range fe.dag.Events {
		if otherEvent.IsFamous != nil && *otherEvent.IsFamous {
			if fe.dag.IsAncestor(event.Hash(), otherEvent.Hash()) {
				witnessTimes = append(witnessTimes, otherEvent.Lamport)
			}
		}
	}

	medianTime := fe.median(witnessTimes)
	consensusReached := len(witnessTimes) > 0

	return &TimeConsensus{
		EventID:          event.Hash(),
		WitnessTimes:     witnessTimes,
		MedianTime:       medianTime,
		ConsensusReached: consensusReached,
	}
}

// GetFinalityStats آمار نهائی‌سازی
func (fe *FinalityEngine) GetFinalityStats() map[string]interface{} {
	stats := make(map[string]interface{})

	totalFinalized := 0
	finalizedByRound := make(map[uint64]int)
	finalizedByCreator := make(map[string]int)
	finalizedByTime := make(map[uint64]int)

	for _, event := range fe.dag.Events {
		if event.Atropos != (EventID{}) {
			totalFinalized++
			finalizedByRound[event.RoundReceived]++
			finalizedByCreator[event.CreatorID]++
			finalizedByTime[event.AtroposTime]++
		}
	}

	stats["total_finalized"] = totalFinalized
	stats["finalized_by_round"] = finalizedByRound
	stats["finalized_by_creator"] = finalizedByCreator
	stats["finalized_by_time"] = finalizedByTime

	// محاسبه آمار اضافی
	if totalFinalized > 0 {
		stats["avg_finalized_per_round"] = float64(totalFinalized) / float64(len(finalizedByRound))
		stats["avg_finalized_per_creator"] = float64(totalFinalized) / float64(len(finalizedByCreator))
	} else {
		stats["avg_finalized_per_round"] = 0.0
		stats["avg_finalized_per_creator"] = 0.0
	}

	// آمار cache
	if fe.cacheManager != nil {
		cacheStats := fe.cacheManager.GetCacheStats()
		for key, value := range cacheStats {
			stats["cache_"+key] = value
		}
	}

	return stats
}

// GetFinalityByVisibility دریافت نهائی‌سازی بر اساس visibility
func (fe *FinalityEngine) GetFinalityByVisibility(minVisibility int) []*Event {
	var events []*Event

	for _, event := range fe.dag.Events {
		if event.Atropos != (EventID{}) {
			// محاسبه visibility
			visibility := fe.calculateVisibility(event)
			if visibility >= minVisibility {
				events = append(events, event)
			}
		}
	}

	return events
}

// calculateVisibility محاسبه visibility یک event
func (fe *FinalityEngine) calculateVisibility(event *Event) int {
	visibility := 0

	for _, otherEvent := range fe.dag.Events {
		if fe.dag.IsAncestor(event.Hash(), otherEvent.Hash()) {
			visibility++
		}
	}

	return visibility
}

// GetFinalityByConsensus دریافت نهائی‌سازی بر اساس شرایط اجماع
func (fe *FinalityEngine) GetFinalityByConsensus(consensusThreshold float64) []*Event {
	var events []*Event

	for _, event := range fe.dag.Events {
		if event.Atropos != (EventID{}) {
			// محاسبه consensus ratio
			consensusRatio := fe.calculateConsensusRatio(event)
			if consensusRatio >= consensusThreshold {
				events = append(events, event)
			}
		}
	}

	return events
}

// calculateConsensusRatio محاسبه نسبت اجماع
func (fe *FinalityEngine) calculateConsensusRatio(event *Event) float64 {
	totalWitnesses := 0
	agreeingWitnesses := 0

	// شمارش شاهدان موافق
	for _, otherEvent := range fe.dag.Events {
		if otherEvent.IsFamous != nil && *otherEvent.IsFamous {
			totalWitnesses++
			if fe.dag.IsAncestor(event.Hash(), otherEvent.Hash()) {
				agreeingWitnesses++
			}
		}
	}

	if totalWitnesses == 0 {
		return 0.0
	}

	return float64(agreeingWitnesses) / float64(totalWitnesses)
}
