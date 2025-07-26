package main

// RoundInfo اطلاعات مربوط به یک Round اجماع
type RoundInfo struct {
	Witnesses map[EventID]bool
	Roots     map[EventID]bool
}

type RoundTable map[uint64]*RoundInfo

// AssignRound محاسبه‌ی round، witness بودن و root بودن برای یک event
func (d *DAG) AssignRound(e *Event) {
	parents, _ := d.GetParents(e)

	// اگر هیچ پدر نداره، یعنی genesis
	if len(parents) == 0 {
		e.Round = 0
		e.IsRoot = true
		d.ensureRound(e.Round)
		d.Rounds[e.Round].Witnesses[e.Hash()] = true
		d.Rounds[e.Round].Roots[e.Hash()] = true
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
	first := true
	for _, other := range d.Events {
		if other.CreatorID == e.CreatorID && other.Round == e.Round && other.Lamport < e.Lamport {
			first = false
			break
		}
	}
	if first {
		e.IsRoot = true
		d.ensureRound(e.Round)
		d.Rounds[e.Round].Witnesses[e.Hash()] = true
		d.Rounds[e.Round].Roots[e.Hash()] = true
	}
}

func (d *DAG) ensureRound(r uint64) {
	if d.Rounds == nil {
		d.Rounds = make(RoundTable)
	}
	if _, exists := d.Rounds[r]; !exists {
		d.Rounds[r] = &RoundInfo{
			Witnesses: make(map[EventID]bool),
			Roots:     make(map[EventID]bool),
		}
	}
}
