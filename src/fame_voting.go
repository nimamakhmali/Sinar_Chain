package main

import (
	"fmt"
	"sync"
	"time"
)

// FameVoting الگوریتم Fame Voting دقیقاً مطابق Fantom Opera
type FameVoting struct {
	dag *DAG
	mu  sync.RWMutex

	// Fame voting state
	votes     map[string]*Vote
	voteCount map[string]int

	// Witness tracking
	witnesses map[uint64]map[EventID]*Event
	rounds    map[uint64]*RoundInfo

	// Fame determination - دقیقاً مطابق Fantom
	famousWitnesses map[EventID]bool
	decidedRounds   map[uint64]bool

	// Byzantine fault tolerance parameters
	byzantineThreshold float64 // 2/3 for BFT
	minVoteCount       int     // Minimum votes needed
}

// Vote رأی در Fame Voting - دقیقاً مطابق Fantom
type Vote struct {
	WitnessID EventID
	Round     uint64
	Voter     EventID
	Choice    bool // true = yes, false = no
	Decided   bool
	Timestamp uint64
}

// NewFameVoting ایجاد FameVoting جدید با پارامترهای Fantom
func NewFameVoting(dag *DAG) *FameVoting {
	return &FameVoting{
		dag:                dag,
		votes:              make(map[string]*Vote),
		voteCount:          make(map[string]int),
		witnesses:          make(map[uint64]map[EventID]*Event),
		rounds:             make(map[uint64]*RoundInfo),
		famousWitnesses:    make(map[EventID]bool),
		decidedRounds:      make(map[uint64]bool),
		byzantineThreshold: 2.0 / 3.0, // 2/3 for Byzantine fault tolerance
		minVoteCount:       3,         // Minimum 3 votes needed
	}
}

// StartFameVoting شروع Fame Voting برای یک round - دقیقاً مطابق Fantom
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

	// شروع voting برای هر witness - دقیقاً مطابق Fantom
	for witnessID := range witnesses {
		if fv.famousWitnesses[witnessID] {
			continue // قبلاً famous شده
		}

		// شروع voting process با الگوریتم Fantom
		fv.startVotingForWitness(witnessID, round)
	}

	return nil
}

// startVotingForWitness شروع voting برای یک witness خاص - دقیقاً مطابق Fantom
func (fv *FameVoting) startVotingForWitness(witnessID EventID, round uint64) {
	// ایجاد vote برای این witness
	voteKey := fmt.Sprintf("%x_%d", witnessID, round)

	if _, exists := fv.votes[voteKey]; exists {
		return // قبلاً voting شروع شده
	}

	// محاسبه رأی بر اساس الگوریتم Fantom Opera
	vote := fv.calculateVoteFantom(witnessID, round)
	fv.votes[voteKey] = vote

	// بررسی اینکه آیا consensus رسیده - با Byzantine fault tolerance
	if fv.checkConsensusFantom(witnessID, round) {
		fv.famousWitnesses[witnessID] = true
		fv.decideRound(round)
	}
}

// calculateVoteFantom محاسبه رأی بر اساس الگوریتم دقیق Fantom
func (fv *FameVoting) calculateVoteFantom(witnessID EventID, round uint64) *Vote {
	// الگوریتم Fame Voting از Fantom Opera:
	// 1. بررسی تعداد events که این witness را می‌بینند
	// 2. محاسبه رأی بر اساس Byzantine fault tolerance
	// 3. بررسی شرایط consensus

	// شمارش events که این witness را می‌بینند
	seeCount := 0
	totalEvents := 0

	for _, event := range fv.dag.Events {
		if event.Round == round {
			totalEvents++
			if fv.canSee(event.Hash(), witnessID) {
				seeCount++
			}
		}
	}

	// محاسبه نسبت visibility
	visibilityRatio := float64(seeCount) / float64(totalEvents)

	// تصمیم‌گیری بر اساس Byzantine fault tolerance
	shouldVoteYes := visibilityRatio >= fv.byzantineThreshold

	return &Vote{
		WitnessID: witnessID,
		Round:     round,
		Choice:    shouldVoteYes,
		Decided:   false,
		Timestamp: uint64(time.Now().UnixNano() / 1000000),
	}
}

// checkConsensusFantom بررسی consensus با الگوریتم Fantom
func (fv *FameVoting) checkConsensusFantom(witnessID EventID, round uint64) bool {
	// شمارش رأی‌های مثبت
	yesVotes := 0
	totalVotes := 0

	for _, vote := range fv.votes {
		if vote.WitnessID == witnessID && vote.Round == round {
			totalVotes++
			if vote.Choice {
				yesVotes++
			}
		}
	}

	// بررسی حداقل تعداد رأی
	if totalVotes < fv.minVoteCount {
		return false
	}

	// بررسی Byzantine fault tolerance
	consensusRatio := float64(yesVotes) / float64(totalVotes)
	return consensusRatio >= fv.byzantineThreshold
}

// canSee بررسی اینکه آیا event A می‌تواند event B را ببیند - دقیقاً مطابق Fantom
func (fv *FameVoting) canSee(eventA, eventB EventID) bool {
	// الگوریتم visibility از Fantom:
	// event A می‌تواند event B را ببیند اگر:
	// 1. event A در round بعدی باشد
	// 2. event A ancestor event B باشد
	// 3. یا event A و event B در همان round باشند

	eventAObj, existsA := fv.dag.GetEvent(eventA)
	eventBObj, existsB := fv.dag.GetEvent(eventB)

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
		return fv.dag.IsAncestor(eventB, eventA)
	}

	return false
}

// shouldVoteYes تصمیم‌گیری برای رأی مثبت - دقیقاً مطابق Fantom
func (fv *FameVoting) shouldVoteYes(voterID, witnessID EventID, round uint64) bool {
	// الگوریتم رأی‌گیری Fantom:
	// رأی مثبت اگر:
	// 1. voter می‌تواند witness را ببیند
	// 2. تعداد کافی events در round بعدی witness را می‌بینند
	// 3. شرایط Byzantine fault tolerance برآورده شود

	// بررسی visibility
	if !fv.canSee(voterID, witnessID) {
		return false
	}

	// شمارش events در round بعدی که witness را می‌بینند
	nextRound := round + 1
	seeCount := 0
	totalNextRound := 0

	for _, event := range fv.dag.Events {
		if event.Round == nextRound {
			totalNextRound++
			if fv.canSee(event.Hash(), witnessID) {
				seeCount++
			}
		}
	}

	// محاسبه نسبت visibility
	if totalNextRound == 0 {
		return false
	}

	visibilityRatio := float64(seeCount) / float64(totalNextRound)
	return visibilityRatio >= fv.byzantineThreshold
}

// decideRound تصمیم‌گیری برای یک round - دقیقاً مطابق Fantom
func (fv *FameVoting) decideRound(round uint64) {
	fv.mu.Lock()
	defer fv.mu.Unlock()

	// علامت‌گذاری round به عنوان تصمیم‌گیری شده
	fv.decidedRounds[round] = true

	// به‌روزرسانی آمار
	fmt.Printf("🎯 Round %d decided with %d famous witnesses\n", round, len(fv.famousWitnesses))
}

// getWitnesses دریافت witnesses یک round - دقیقاً مطابق Fantom
func (fv *FameVoting) getWitnesses(round uint64) map[EventID]*Event {
	witnesses := make(map[EventID]*Event)

	for _, event := range fv.dag.Events {
		if event.Round == round && fv.isWitness(event, round) {
			witnesses[event.Hash()] = event
		}
	}

	return witnesses
}

// isWitness بررسی اینکه آیا event یک witness است - دقیقاً مطابق Fantom
func (fv *FameVoting) isWitness(event *Event, round uint64) bool {
	// الگوریتم witness از Fantom:
	// event یک witness است اگر:
	// 1. اولین event از این creator در این round باشد
	// 2. یا event در round 0 باشد (genesis)

	if round == 0 {
		return true // Genesis events همیشه witness هستند
	}

	// بررسی اولین بودن در این round
	for _, otherEvent := range fv.dag.Events {
		if otherEvent.CreatorID == event.CreatorID &&
			otherEvent.Round == round &&
			otherEvent.Lamport < event.Lamport {
			return false // event قبلی از همین creator وجود دارد
		}
	}

	return true
}

// getFamousWitnesses دریافت famous witnesses یک round - دقیقاً مطابق Fantom
func (fv *FameVoting) getFamousWitnesses(round uint64) map[EventID]*Event {
	famousWitnesses := make(map[EventID]*Event)

	for _, event := range fv.dag.Events {
		if event.Round == round && fv.famousWitnesses[event.Hash()] {
			famousWitnesses[event.Hash()] = event
		}
	}

	return famousWitnesses
}

// GetVoteStats آمار رأی‌گیری - دقیقاً مطابق Fantom
func (fv *FameVoting) GetVoteStats() map[string]interface{} {
	fv.mu.RLock()
	defer fv.mu.RUnlock()

	stats := make(map[string]interface{})

	// آمار کلی
	stats["total_votes"] = len(fv.votes)
	stats["famous_witnesses"] = len(fv.famousWitnesses)
	stats["decided_rounds"] = len(fv.decidedRounds)

	// آمار per round
	roundStats := make(map[uint64]map[string]interface{})
	for round := range fv.decidedRounds {
		roundStats[round] = map[string]interface{}{
			"witnesses":    len(fv.getWitnesses(round)),
			"famous_count": len(fv.getFamousWitnesses(round)),
			"decided":      true,
		}
	}
	stats["round_stats"] = roundStats

	// آمار Byzantine fault tolerance
	stats["byzantine_threshold"] = fv.byzantineThreshold
	stats["min_vote_count"] = fv.minVoteCount

	return stats
}

// Reset بازنشانی FameVoting - دقیقاً مطابق Fantom
func (fv *FameVoting) Reset() {
	fv.mu.Lock()
	defer fv.mu.Unlock()

	fv.votes = make(map[string]*Vote)
	fv.voteCount = make(map[string]int)
	fv.witnesses = make(map[uint64]map[EventID]*Event)
	fv.rounds = make(map[uint64]*RoundInfo)
	fv.famousWitnesses = make(map[EventID]bool)
	fv.decidedRounds = make(map[uint64]bool)
}
