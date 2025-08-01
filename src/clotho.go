package main

// ClothoSelector مسئول انتخاب Clothos از famous witnesses
type ClothoSelector struct {
	dag *DAG
}

// NewClothoSelector ایجاد ClothoSelector جدید
func NewClothoSelector(dag *DAG) *ClothoSelector {
	return &ClothoSelector{
		dag: dag,
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
			witness.IsClotho = true
			witness.RoundReceived = round + 2 // Clotho در round+2 نهایی می‌شود

			// اضافه کردن به round info
			cs.ensureRound(round + 2)
			cs.dag.Rounds[round+2].Clothos[witness.Hash()] = witness
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
	for _, nextWitness := range famousWitnessesNextRound {
		if cs.dag.IsAncestor(witness.Hash(), nextWitness.Hash()) {
			seeCount++
		}
	}

	// باید اکثریت (2/3) آن را ببینند
	requiredCount := (2 * len(famousWitnessesNextRound)) / 3
	return seeCount > requiredCount
}

// getFamousWitnesses دریافت famous witnesses یک round
func (cs *ClothoSelector) getFamousWitnesses(round uint64) []*Event {
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

// GetClothosStats آمار Clothos
func (cs *ClothoSelector) GetClothosStats() map[string]interface{} {
	stats := make(map[string]interface{})

	totalClothos := 0
	clothosByRound := make(map[uint64]int)
	clothosByCreator := make(map[string]int)

	for _, event := range cs.dag.Events {
		if event.IsClotho {
			totalClothos++
			clothosByRound[event.RoundReceived]++
			clothosByCreator[event.CreatorID]++
		}
	}

	stats["total_clothos"] = totalClothos
	stats["clothos_by_round"] = clothosByRound
	stats["clothos_by_creator"] = clothosByCreator

	return stats
}
