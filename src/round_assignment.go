package main

// RoundAssignment مسئول محاسبه round، witness بودن و root بودن برای events
type RoundAssignment struct {
	dag *DAG
}

// NewRoundAssignment ایجاد RoundAssignment جدید
func NewRoundAssignment(dag *DAG) *RoundAssignment {
	return &RoundAssignment{
		dag: dag,
	}
}

// AssignRound محاسبه‌ی round، witness بودن و root بودن برای یک event
func (ra *RoundAssignment) AssignRound(e *Event) {
	parents, err := ra.dag.GetParents(e)
	if err != nil {
		return
	}

	// اگر هیچ پدر نداره، یعنی genesis
	if len(parents) == 0 {
		e.Round = 0
		e.IsRoot = true
		ra.ensureRound(e.Round)
		ra.dag.Rounds[e.Round].Witnesses[e.Hash()] = e
		ra.dag.Rounds[e.Round].Roots[e.Hash()] = e
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
	isWitness := ra.isWitness(e)
	if isWitness {
		e.IsRoot = true
		ra.ensureRound(e.Round)
		ra.dag.Rounds[e.Round].Witnesses[e.Hash()] = e
		ra.dag.Rounds[e.Round].Roots[e.Hash()] = e
	}
}

// isWitness بررسی اینکه آیا event یک witness است
func (ra *RoundAssignment) isWitness(e *Event) bool {
	// اولین event از این creator در این round
	first := true
	for _, other := range ra.dag.Events {
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
func (ra *RoundAssignment) ensureRound(r uint64) {
	if ra.dag.Rounds == nil {
		ra.dag.Rounds = make(RoundTable)
	}
	if _, exists := ra.dag.Rounds[r]; !exists {
		ra.dag.Rounds[r] = &RoundInfo{
			Witnesses: make(map[EventID]*Event),
			Roots:     make(map[EventID]*Event),
			Clothos:   make(map[EventID]*Event),
			Atropos:   make(map[EventID]*Event),
		}
	}
}

// GetWitnesses دریافت witnesses یک round
func (ra *RoundAssignment) GetWitnesses(round uint64) []*Event {
	roundInfo, exists := ra.dag.Rounds[round]
	if !exists {
		return nil
	}

	witnesses := make([]*Event, 0, len(roundInfo.Witnesses))
	for _, witness := range roundInfo.Witnesses {
		witnesses = append(witnesses, witness)
	}
	return witnesses
}

// GetRoots دریافت roots یک round
func (ra *RoundAssignment) GetRoots(round uint64) []*Event {
	roundInfo, exists := ra.dag.Rounds[round]
	if !exists {
		return nil
	}

	roots := make([]*Event, 0, len(roundInfo.Roots))
	for _, root := range roundInfo.Roots {
		roots = append(roots, root)
	}
	return roots
}

// GetRoundInfo دریافت اطلاعات یک round
func (ra *RoundAssignment) GetRoundInfo(round uint64) (*RoundInfo, bool) {
	roundInfo, exists := ra.dag.Rounds[round]
	return roundInfo, exists
}

// GetLatestRound دریافت آخرین round
func (ra *RoundAssignment) GetLatestRound() uint64 {
	maxRound := uint64(0)
	for round := range ra.dag.Rounds {
		if round > maxRound {
			maxRound = round
		}
	}
	return maxRound
}

// AssignRoundsForEvents محاسبه round برای تمام events
func (ra *RoundAssignment) AssignRoundsForEvents(events []*Event) {
	for _, event := range events {
		ra.AssignRound(event)
	}
}
