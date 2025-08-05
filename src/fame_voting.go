package main

import (
	"fmt"
	"sync"
)

// FameVoting الگوریتم Fame Voting مشابه Fantom Opera
type FameVoting struct {
	dag *DAG
	mu  sync.RWMutex

	// Fame voting state
	votes     map[string]*Vote
	voteCount map[string]int

	// Witness tracking
	witnesses map[uint64]map[EventID]*Event
	rounds    map[uint64]*RoundInfo

	// Fame determination
	famousWitnesses map[EventID]bool
	decidedRounds   map[uint64]bool
}

// Vote رأی در Fame Voting
type Vote struct {
	WitnessID EventID
	Round     uint64
	Voter     EventID
	Choice    bool // true = yes, false = no
	Decided   bool
}

// NewFameVoting ایجاد FameVoting جدید
func NewFameVoting(dag *DAG) *FameVoting {
	return &FameVoting{
		dag:             dag,
		votes:           make(map[string]*Vote),
		voteCount:       make(map[string]int),
		witnesses:       make(map[uint64]map[EventID]*Event),
		rounds:          make(map[uint64]*RoundInfo),
		famousWitnesses: make(map[EventID]bool),
		decidedRounds:   make(map[uint64]bool),
	}
}

// StartFameVoting شروع Fame Voting برای یک round
func (fv *FameVoting) StartFameVoting(round uint64) error {
	fv.mu.Lock()
	defer fv.mu.Unlock()

	// بررسی اینکه آیا این round قبلاً تصمیم گرفته شده
	if fv.decidedRounds[round] {
		return nil
	}

	// دریافت witnesses این round
	witnesses := fv.getWitnesses(round)
	if len(witnesses) == 0 {
		return fmt.Errorf("no witnesses found for round %d", round)
	}

	// شروع voting برای هر witness
	for witnessID := range witnesses {
		if fv.famousWitnesses[witnessID] {
			continue // قبلاً famous شده
		}

		// شروع voting process
		fv.startVotingForWitness(witnessID, round)
	}

	return nil
}

// startVotingForWitness شروع voting برای یک witness خاص
func (fv *FameVoting) startVotingForWitness(witnessID EventID, round uint64) {
	// ایجاد vote برای این witness
	voteKey := fmt.Sprintf("%x_%d", witnessID, round)

	if _, exists := fv.votes[voteKey]; exists {
		return // قبلاً voting شروع شده
	}

	// محاسبه رأی بر اساس الگوریتم Lachesis
	vote := fv.calculateVote(witnessID, round)
	fv.votes[voteKey] = vote

	// بررسی اینکه آیا consensus رسیده
	if fv.checkConsensus(witnessID, round) {
		fv.famousWitnesses[witnessID] = true
		fv.decideRound(round)
	}
}

// calculateVote محاسبه رأی بر اساس الگوریتم Lachesis
func (fv *FameVoting) calculateVote(witnessID EventID, round uint64) *Vote {
	// الگوریتم Fame Voting از Fantom Opera:
	// 1. بررسی تعداد events که این witness را می‌بینند
	// 2. محاسبه نسبت events که رأی مثبت می‌دهند
	// 3. تصمیم بر اساس threshold

	// شمارش events که این witness را می‌بینند
	seeCount := 0
	yesCount := 0

	fv.dag.mu.RLock()
	for eventID, event := range fv.dag.Events {
		if event.Round > round {
			continue
		}

		// بررسی اینکه آیا این event این witness را می‌بیند
		if fv.canSee(eventID, witnessID) {
			seeCount++

			// محاسبه رأی بر اساس الگوریتم Lachesis
			if fv.shouldVoteYes(eventID, witnessID, round) {
				yesCount++
			}
		}
	}
	fv.dag.mu.RUnlock()

	// تصمیم بر اساس threshold (2/3 majority)
	threshold := (seeCount * 2) / 3
	isFamous := yesCount > threshold

	return &Vote{
		WitnessID: witnessID,
		Round:     round,
		Choice:    isFamous,
		Decided:   true,
	}
}

// canSee بررسی اینکه آیا event A می‌تواند event B را ببیند
func (fv *FameVoting) canSee(eventA, eventB EventID) bool {
	// الگوریتم canSee از Fantom Opera
	// بررسی اینکه آیا eventA ancestor از eventB است
	return fv.dag.IsAncestor(eventA, eventB)
}

// shouldVoteYes بررسی اینکه آیا باید رأی مثبت داد
func (fv *FameVoting) shouldVoteYes(voterID, witnessID EventID, round uint64) bool {
	// الگوریتم رأی‌گیری از Fantom Opera
	// بر اساس Lamport timestamp و round assignment

	voter, exists := fv.dag.GetEvent(voterID)
	if !exists {
		return false
	}

	witness, exists := fv.dag.GetEvent(witnessID)
	if !exists {
		return false
	}

	// رأی مثبت اگر voter در round بالاتر از witness باشد
	return voter.Round > witness.Round
}

// checkConsensus بررسی رسیدن به consensus
func (fv *FameVoting) checkConsensus(witnessID EventID, round uint64) bool {
	voteKey := fmt.Sprintf("%x_%d", witnessID, round)
	_, exists := fv.votes[voteKey]
	if !exists {
		return false
	}

	// بررسی تعداد رأی‌های مثبت
	yesVotes := 0
	totalVotes := 0

	for _, v := range fv.votes {
		if v.Round == round && v.WitnessID == witnessID {
			totalVotes++
			if v.Choice {
				yesVotes++
			}
		}
	}

	// Consensus اگر 2/3 رأی مثبت باشد
	threshold := (totalVotes * 2) / 3
	return yesVotes > threshold
}

// decideRound تصمیم‌گیری برای یک round
func (fv *FameVoting) decideRound(round uint64) {
	fv.decidedRounds[round] = true

	// به‌روزرسانی round info
	if roundInfo, exists := fv.rounds[round]; exists {
		roundInfo.Decided = true

		// اضافه کردن famous witnesses به round
		for witnessID := range fv.famousWitnesses {
			if witness, exists := fv.dag.GetEvent(witnessID); exists {
				if witness.Round == round {
					roundInfo.Witnesses[witnessID] = witness
				}
			}
		}
	}

	fmt.Printf("🎯 Round %d decided with %d famous witnesses\n",
		round, len(fv.getFamousWitnesses(round)))
}

// getWitnesses دریافت witnesses یک round
func (fv *FameVoting) getWitnesses(round uint64) map[EventID]*Event {
	witnesses := make(map[EventID]*Event)

	fv.dag.mu.RLock()
	for eventID, event := range fv.dag.Events {
		if event.Round == round && fv.isWitness(event, round) {
			witnesses[eventID] = event
		}
	}
	fv.dag.mu.RUnlock()

	return witnesses
}

// isWitness بررسی اینکه آیا یک event witness است
func (fv *FameVoting) isWitness(event *Event, round uint64) bool {
	// الگوریتم تشخیص witness از Fantom Opera
	// یک event witness است اگر:
	// 1. در این round باشد
	// 2. اولین event از creator خود در این round باشد

	if event.Round != round {
		return false
	}

	// بررسی اینکه آیا اولین event از این creator در این round است
	for _, otherEvent := range fv.dag.Events {
		if otherEvent.CreatorID == event.CreatorID &&
			otherEvent.Round == round &&
			otherEvent.Lamport < event.Lamport {
			return false
		}
	}

	return true
}

// getFamousWitnesses دریافت famous witnesses یک round
func (fv *FameVoting) getFamousWitnesses(round uint64) map[EventID]*Event {
	famous := make(map[EventID]*Event)

	for witnessID, isFamous := range fv.famousWitnesses {
		if isFamous {
			if witness, exists := fv.dag.GetEvent(witnessID); exists {
				if witness.Round == round {
					famous[witnessID] = witness
				}
			}
		}
	}

	return famous
}

// GetVoteStats آمار voting
func (fv *FameVoting) GetVoteStats() map[string]interface{} {
	fv.mu.RLock()
	defer fv.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_votes"] = len(fv.votes)
	stats["famous_witnesses"] = len(fv.famousWitnesses)
	stats["decided_rounds"] = len(fv.decidedRounds)

	// آمار per round
	roundStats := make(map[uint64]map[string]interface{})
	for round := range fv.rounds {
		witnesses := fv.getWitnesses(round)
		famous := fv.getFamousWitnesses(round)

		roundStats[round] = map[string]interface{}{
			"total_witnesses":  len(witnesses),
			"famous_witnesses": len(famous),
			"decided":          fv.decidedRounds[round],
		}
	}
	stats["round_stats"] = roundStats

	return stats
}

// Reset بازنشانی برای تست
func (fv *FameVoting) Reset() {
	fv.mu.Lock()
	defer fv.mu.Unlock()

	fv.votes = make(map[string]*Vote)
	fv.voteCount = make(map[string]int)
	fv.famousWitnesses = make(map[EventID]bool)
	fv.decidedRounds = make(map[uint64]bool)
}
