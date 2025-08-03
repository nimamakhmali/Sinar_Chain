package main

import (
	"math/big"
	"sync"
)

// ClothoSelector مسئول انتخاب Clothos از famous witnesses
type ClothoSelector struct {
	dag          *DAG
	fameVoting   *FameVoting
	cacheManager *CacheManager
	mu           sync.RWMutex
}

// ClothoInfo اطلاعات Clotho
type ClothoInfo struct {
	EventID       EventID
	Round         uint64
	CreatorID     string
	IsSelected    bool
	SelectionTime uint64
	AtroposTime   uint64
}

// NewClothoSelector ایجاد ClothoSelector جدید
func NewClothoSelector(dag *DAG) *ClothoSelector {
	return &ClothoSelector{
		dag:          dag,
		fameVoting:   NewFameVoting(dag),
		cacheManager: NewCacheManager(1000),
	}
}

// SelectClothos انتخاب Clothos برای تمام rounds
func (cs *ClothoSelector) SelectClothos() {
	// برای هر round که famous witnesses دارد
	for round := range cs.dag.Rounds {
		famousWitnesses := cs.getFamousWitnesses(round)
		if len(famousWitnesses) == 0 {
			continue
		}

		// انتخاب Clothos برای این round
		cs.selectClothosForRound(round, famousWitnesses)
	}
}

// selectClothosForRound انتخاب Clothos برای یک round
func (cs *ClothoSelector) selectClothosForRound(round uint64, famousWitnesses []*Event) {
	// برای هر famous witness، بررسی تبدیل به Clotho
	for _, witness := range famousWitnesses {
		if witness.IsClotho {
			continue // قبلاً Clotho شده
		}

		// بررسی شرایط تبدیل به Clotho
		if cs.canBecomeClotho(witness, round) {
			cs.convertToClotho(witness, round)
		}
	}
}

// canBecomeClotho بررسی شرایط تبدیل witness به Clotho
func (cs *ClothoSelector) canBecomeClotho(witness *Event, round uint64) bool {
	// شرط 1: باید famous باشد
	if witness.IsFamous == nil || !*witness.IsFamous {
		return false
	}

	// شرط 2: باید اکثریت famous witnesses از round+1 آن را ببینند
	nextRound := round + 1
	famousWitnessesNextRound := cs.getFamousWitnesses(nextRound)
	if len(famousWitnessesNextRound) == 0 {
		return false
	}

	// شمارش famous witnesses که این witness را می‌بینند
	seeCount := 0
	totalFamousWitnesses := len(famousWitnessesNextRound)

	for _, nextWitness := range famousWitnessesNextRound {
		if cs.dag.IsAncestor(witness.Hash(), nextWitness.Hash()) {
			seeCount++
		}
	}

	// باید اکثریت (2/3) آن را ببینند (Byzantine fault tolerance)
	requiredCount := (2 * totalFamousWitnesses) / 3
	return seeCount > requiredCount
}

// convertToClotho تبدیل witness به Clotho
func (cs *ClothoSelector) convertToClotho(witness *Event, round uint64) {
	// تبدیل به Clotho
	witness.IsClotho = true
	witness.RoundReceived = round + 2 // Clotho در round+2 نهایی می‌شود

	// اضافه کردن به round info
	cs.ensureRound(round + 2)
	cs.dag.Rounds[round+2].Clothos[witness.Hash()] = witness

	// محاسبه زمان انتخاب
	witness.AtroposTime = witness.Lamport
}

// getFamousWitnesses دریافت famous witnesses یک round
func (cs *ClothoSelector) getFamousWitnesses(round uint64) []*Event {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	// بررسی cache manager
	if cachedData, exists := cs.cacheManager.GetConsensusCache(round); exists {
		if witnessesData, ok := cachedData["famous_witnesses"]; ok {
			if witnesses, ok := witnessesData.([]*Event); ok {
				return witnesses
			}
		}
	}

	roundInfo, exists := cs.dag.Rounds[round]
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
	cs.cacheManager.SetConsensusCache(round, cacheData)

	return famousWitnesses
}

// GetClothos دریافت Clothos یک round
func (cs *ClothoSelector) GetClothos(round uint64) []*Event {
	roundInfo, exists := cs.dag.Rounds[round]
	if !exists {
		return nil
	}

	clothos := make([]*Event, 0, len(roundInfo.Clothos))
	for _, clotho := range roundInfo.Clothos {
		clothos = append(clothos, clotho)
	}

	return clothos
}

// ensureRound اطمینان از وجود round
func (cs *ClothoSelector) ensureRound(r uint64) {
	if cs.dag.Rounds == nil {
		cs.dag.Rounds = make(RoundTable)
	}
	if _, exists := cs.dag.Rounds[r]; !exists {
		cs.dag.Rounds[r] = &RoundInfo{
			Witnesses: make(map[EventID]*Event),
			Roots:     make(map[EventID]*Event),
			Clothos:   make(map[EventID]*Event),
			Atropos:   make(map[EventID]*Event),
		}
	}
}

// GetClothosForRound دریافت Clothos برای یک round خاص
func (cs *ClothoSelector) GetClothosForRound(round uint64) []*Event {
	roundInfo, exists := cs.dag.Rounds[round]
	if !exists {
		return nil
	}

	var clothos []*Event
	for _, clotho := range roundInfo.Clothos {
		clothos = append(clothos, clotho)
	}

	return clothos
}

// IsClotho بررسی اینکه آیا یک event Clotho است
func (cs *ClothoSelector) IsClotho(eventID EventID) bool {
	event, exists := cs.dag.GetEvent(eventID)
	if !exists {
		return false
	}
	return event.IsClotho
}

// GetClothosByCreator دریافت Clothos یک creator
func (cs *ClothoSelector) GetClothosByCreator(creatorID string) []*Event {
	var clothos []*Event

	for _, event := range cs.dag.Events {
		if event.CreatorID == creatorID && event.IsClotho {
			clothos = append(clothos, event)
		}
	}

	return clothos
}

// GetClothosInRange دریافت Clothos در یک بازه round
func (cs *ClothoSelector) GetClothosInRange(fromRound, toRound uint64) []*Event {
	var clothos []*Event

	for round := fromRound; round <= toRound; round++ {
		roundClothos := cs.GetClothos(round)
		clothos = append(clothos, roundClothos...)
	}

	return clothos
}

// GetClothosByStake دریافت Clothos بر اساس stake
func (cs *ClothoSelector) GetClothosByStake(minStake *big.Int) []*Event {
	var clothos []*Event

	for _, event := range cs.dag.Events {
		if event.IsClotho {
			// در نسخه کامل، stake از validator set گرفته می‌شود
			stake := big.NewInt(1000000) // 1M tokens default
			if stake.Cmp(minStake) >= 0 {
				clothos = append(clothos, event)
			}
		}
	}

	return clothos
}

// GetClothosByTime دریافت Clothos در یک بازه زمانی
func (cs *ClothoSelector) GetClothosByTime(fromTime, toTime uint64) []*Event {
	var clothos []*Event

	for _, event := range cs.dag.Events {
		if event.IsClotho &&
			event.AtroposTime >= fromTime &&
			event.AtroposTime <= toTime {
			clothos = append(clothos, event)
		}
	}

	return clothos
}

// GetClothosStats آمار Clothos
func (cs *ClothoSelector) GetClothosStats() map[string]interface{} {
	stats := make(map[string]interface{})

	totalClothos := 0
	clothosByRound := make(map[uint64]int)
	clothosByCreator := make(map[string]int)
	clothosByTime := make(map[uint64]int)

	for _, event := range cs.dag.Events {
		if event.IsClotho {
			totalClothos++
			clothosByRound[event.RoundReceived]++
			clothosByCreator[event.CreatorID]++
			clothosByTime[event.AtroposTime]++
		}
	}

	stats["total_clothos"] = totalClothos
	stats["clothos_by_round"] = clothosByRound
	stats["clothos_by_creator"] = clothosByCreator
	stats["clothos_by_time"] = clothosByTime

	// محاسبه آمار اضافی
	if totalClothos > 0 {
		stats["avg_clothos_per_round"] = float64(totalClothos) / float64(len(clothosByRound))
		stats["avg_clothos_per_creator"] = float64(totalClothos) / float64(len(clothosByCreator))
	} else {
		stats["avg_clothos_per_round"] = 0.0
		stats["avg_clothos_per_creator"] = 0.0
	}

	// آمار cache
	if cs.cacheManager != nil {
		cacheStats := cs.cacheManager.GetCacheStats()
		for key, value := range cacheStats {
			stats["cache_"+key] = value
		}
	}

	return stats
}

// ValidateClothoSelection اعتبارسنجی انتخاب Clotho
func (cs *ClothoSelector) ValidateClothoSelection(clotho *Event, round uint64) bool {
	// بررسی شرایط اعتبارسنجی
	if !clotho.IsClotho {
		return false
	}

	// بررسی famous بودن
	if clotho.IsFamous == nil || !*clotho.IsFamous {
		return false
	}

	// بررسی round assignment
	if clotho.RoundReceived != round+2 {
		return false
	}

	// بررسی visibility conditions
	nextRound := round + 1
	famousWitnessesNextRound := cs.getFamousWitnesses(nextRound)
	seeCount := 0

	for _, witness := range famousWitnessesNextRound {
		if cs.dag.IsAncestor(clotho.Hash(), witness.Hash()) {
			seeCount++
		}
	}

	requiredCount := (2 * len(famousWitnessesNextRound)) / 3
	return seeCount > requiredCount
}

// GetClothosByVisibility دریافت Clothos بر اساس visibility
func (cs *ClothoSelector) GetClothosByVisibility(minVisibility int) []*Event {
	var clothos []*Event

	for _, event := range cs.dag.Events {
		if event.IsClotho {
			// محاسبه visibility
			visibility := cs.calculateVisibility(event)
			if visibility >= minVisibility {
				clothos = append(clothos, event)
			}
		}
	}

	return clothos
}

// calculateVisibility محاسبه visibility یک event
func (cs *ClothoSelector) calculateVisibility(event *Event) int {
	visibility := 0

	for _, otherEvent := range cs.dag.Events {
		if cs.dag.IsAncestor(event.Hash(), otherEvent.Hash()) {
			visibility++
		}
	}

	return visibility
}

// GetClothosByConsensus دریافت Clothos بر اساس شرایط اجماع
func (cs *ClothoSelector) GetClothosByConsensus(consensusThreshold float64) []*Event {
	var clothos []*Event

	for _, event := range cs.dag.Events {
		if event.IsClotho {
			// محاسبه consensus ratio
			consensusRatio := cs.calculateConsensusRatio(event)
			if consensusRatio >= consensusThreshold {
				clothos = append(clothos, event)
			}
		}
	}

	return clothos
}

// calculateConsensusRatio محاسبه نسبت اجماع
func (cs *ClothoSelector) calculateConsensusRatio(event *Event) float64 {
	totalWitnesses := 0
	agreeingWitnesses := 0

	// شمارش شاهدان موافق
	for _, otherEvent := range cs.dag.Events {
		if otherEvent.IsFamous != nil && *otherEvent.IsFamous {
			totalWitnesses++
			if cs.dag.IsAncestor(event.Hash(), otherEvent.Hash()) {
				agreeingWitnesses++
			}
		}
	}

	if totalWitnesses == 0 {
		return 0.0
	}

	return float64(agreeingWitnesses) / float64(totalWitnesses)
}
