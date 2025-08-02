package main

import (
	"math/big"
)

// Vote رای یک voter برای یک witness
type Vote struct {
	Vote    bool
	Decided bool
	Round   uint64
	Weight  *big.Int // وزن رای (بر اساس stake)
}

// VoteRecord نگهداری آراء: [voter][witness]
type VoteRecord map[EventID]map[EventID]*Vote

// FameVotingState نگهداری وضعیت fame voting
type FameVotingState struct {
	CurrentRound uint64
	Votes        VoteRecord
	Decided      map[EventID]bool
	Weights      map[EventID]*big.Int // وزن هر voter
}

// FameVoting مسئول اجرای الگوریتم fame voting
type FameVoting struct {
	dag   *DAG
	state *FameVotingState
}

// NewFameVoting ایجاد FameVoting جدید
func NewFameVoting(dag *DAG) *FameVoting {
	return &FameVoting{
		dag: dag,
		state: &FameVotingState{
			Votes:   make(VoteRecord),
			Decided: make(map[EventID]bool),
			Weights: make(map[EventID]*big.Int),
		},
	}
}

// StartFameVoting شروع fame voting برای تمام rounds
func (fv *FameVoting) StartFameVoting() {
	// پیدا کردن آخرین round
	maxRound := fv.getLatestRound()

	// اجرای fame voting برای هر round
	for r := uint64(0); r <= maxRound; r++ {
		fv.runFameVotingForRound(r)
	}
}

// runFameVotingForRound اجرای fame voting برای یک round
func (fv *FameVoting) runFameVotingForRound(round uint64) {
	roundInfo, exists := fv.dag.Rounds[round]
	if !exists {
		return
	}

	// برای هر witness در این round
	for witnessID := range roundInfo.Witnesses {
		witness, _ := fv.dag.GetEvent(witnessID)
		if witness == nil || witness.IsFamous != nil {
			continue
		}

		// اجرای fame voting برای این witness
		fv.runFameVotingForWitness(witnessID, round)
	}
}

// runFameVotingForWitness اجرای fame voting برای یک witness
func (fv *FameVoting) runFameVotingForWitness(witnessID EventID, round uint64) {
	witness, _ := fv.dag.GetEvent(witnessID)
	if witness == nil {
		return
	}

	// شروع از round بعدی
	for votingRound := round + 1; votingRound <= round+10; votingRound++ { // حداکثر 10 round
		voters := fv.getVotersForRound(votingRound)
		if len(voters) == 0 {
			continue
		}

		trueVotes := big.NewInt(0)
		falseVotes := big.NewInt(0)
		totalWeight := big.NewInt(0)

		// جمع‌آوری آراء با وزن
		for voterID := range voters {
			vote := fv.getVoteForWitness(voterID, witnessID, votingRound)
			if vote == nil {
				continue
			}

			weight := fv.getVoterWeight(voterID)
			totalWeight.Add(totalWeight, weight)

			if vote.Vote {
				trueVotes.Add(trueVotes, weight)
			} else {
				falseVotes.Add(falseVotes, weight)
			}
		}

		if totalWeight.Sign() == 0 {
			continue
		}

		// محاسبه دو سوم وزن کل
		twoThirds := new(big.Int).Mul(totalWeight, big.NewInt(2))
		twoThirds.Div(twoThirds, big.NewInt(3))

		// بررسی تصمیم‌گیری
		if trueVotes.Cmp(twoThirds) > 0 {
			// Witness famous شد
			t := true
			witness.IsFamous = &t
			fv.state.Decided[witnessID] = true
			return
		} else if falseVotes.Cmp(twoThirds) > 0 {
			// Witness not famous شد
			f := false
			witness.IsFamous = &f
			fv.state.Decided[witnessID] = true
			return
		}
	}
}

// getVotersForRound دریافت voters برای یک round
func (fv *FameVoting) getVotersForRound(round uint64) map[EventID]*Event {
	roundInfo, exists := fv.dag.Rounds[round]
	if !exists {
		return nil
	}

	return roundInfo.Witnesses
}

// getVoteForWitness دریافت رای یک voter برای یک witness
func (fv *FameVoting) getVoteForWitness(voterID, witnessID EventID, round uint64) *Vote {
	// بررسی cache
	if fv.state.Votes[voterID] == nil {
		fv.state.Votes[voterID] = make(map[EventID]*Vote)
	}

	if vote, exists := fv.state.Votes[voterID][witnessID]; exists {
		return vote
	}

	// محاسبه رای جدید
	vote := fv.calculateVote(voterID, witnessID, round)
	fv.state.Votes[voterID][witnessID] = vote
	return vote
}

// calculateVote محاسبه رای یک voter برای یک witness
func (fv *FameVoting) calculateVote(voterID, witnessID EventID, round uint64) *Vote {
	voter, _ := fv.dag.GetEvent(voterID)
	witness, _ := fv.dag.GetEvent(witnessID)

	if voter == nil || witness == nil {
		return nil
	}

	// بررسی اینکه آیا voter می‌تواند witness را ببیند
	if !fv.dag.IsAncestor(witnessID, voterID) {
		return nil
	}

	// محاسبه رای بر اساس الگوریتم Lachesis کامل
	vote := fv.computeVoteAdvanced(voter, witness, round)

	return &Vote{
		Vote:    vote,
		Decided: false, // در این مرحله تصمیم‌گیری نشده
		Round:   round,
		Weight:  fv.getVoterWeight(voterID),
	}
}

// computeVoteAdvanced محاسبه رای پیشرفته بر اساس الگوریتم Lachesis
func (fv *FameVoting) computeVoteAdvanced(voter, witness *Event, round uint64) bool {
	// الگوریتم پیشرفته Lachesis:
	// 1. بررسی visibility
	// 2. بررسی Lamport time
	// 3. بررسی round assignment
	// 4. بررسی consensus conditions

	// شرط 1: Visibility
	if !fv.dag.IsAncestor(witness.Hash(), voter.Hash()) {
		return false
	}

	// شرط 2: Lamport time consistency
	if voter.Lamport <= witness.Lamport {
		return false
	}

	// شرط 3: Round assignment
	voterRound := voter.Lamport / uint64(2000) // 2 seconds
	witnessRound := witness.Lamport / uint64(2000)

	if voterRound <= witnessRound {
		return false
	}

	// شرط 4: Consensus conditions
	return fv.checkConsensusConditions(voter, witness, round)
}

// checkConsensusConditions بررسی شرایط consensus
func (fv *FameVoting) checkConsensusConditions(voter, witness *Event, round uint64) bool {
	// بررسی اینکه آیا voter در round مناسب است
	voterRound := voter.Lamport / uint64(2000)
	if voterRound < round {
		return false
	}

	// بررسی اینکه آیا witness در round مناسب است
	witnessRound := witness.Lamport / uint64(2000)
	if witnessRound != round {
		return false
	}

	// بررسی consensus conditions اضافی
	return fv.checkAdditionalConditions(voter, witness)
}

// checkAdditionalConditions بررسی شرایط اضافی
func (fv *FameVoting) checkAdditionalConditions(voter, witness *Event) bool {
	// بررسی unique creator
	if voter.CreatorID == witness.CreatorID {
		return false
	}

	// بررسی Lamport time consistency
	if voter.Lamport <= witness.Lamport {
		return false
	}

	return true
}

// getVoterWeight دریافت وزن یک voter
func (fv *FameVoting) getVoterWeight(voterID EventID) *big.Int {
	// بررسی cache
	if weight, exists := fv.state.Weights[voterID]; exists {
		return weight
	}

	// محاسبه وزن بر اساس stake
	voter, _ := fv.dag.GetEvent(voterID)
	if voter == nil {
		return big.NewInt(1) // وزن پیش‌فرض
	}

	// در نسخه کامل، این از validator set گرفته می‌شود
	weight := big.NewInt(1000000) // 1M tokens default
	fv.state.Weights[voterID] = weight
	return weight
}

// GetFamousWitnesses دریافت famous witnesses یک round
func (fv *FameVoting) GetFamousWitnesses(round uint64) []*Event {
	roundInfo, exists := fv.dag.Rounds[round]
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

// getLatestRound دریافت آخرین round
func (fv *FameVoting) getLatestRound() uint64 {
	maxRound := uint64(0)
	for round := range fv.dag.Rounds {
		if round > maxRound {
			maxRound = round
		}
	}
	return maxRound
}

// IsDecided بررسی اینکه آیا fame voting برای یک witness تصمیم‌گیری شده
func (fv *FameVoting) IsDecided(witnessID EventID) bool {
	return fv.state.Decided[witnessID]
}

// GetVoteStats آمار آراء برای یک witness
func (fv *FameVoting) GetVoteStats(witnessID EventID, round uint64) (trueVotes, falseVotes, totalVotes int) {
	voters := fv.getVotersForRound(round)

	for voterID := range voters {
		vote := fv.getVoteForWitness(voterID, witnessID, round)
		if vote == nil {
			continue
		}

		totalVotes++
		if vote.Vote {
			trueVotes++
		} else {
			falseVotes++
		}
	}

	return
}

// GetWeightedVoteStats آمار آراء با وزن
func (fv *FameVoting) GetWeightedVoteStats(witnessID EventID, round uint64) (trueWeight, falseWeight, totalWeight *big.Int) {
	voters := fv.getVotersForRound(round)

	trueWeight = big.NewInt(0)
	falseWeight = big.NewInt(0)
	totalWeight = big.NewInt(0)

	for voterID := range voters {
		vote := fv.getVoteForWitness(voterID, witnessID, round)
		if vote == nil {
			continue
		}

		weight := fv.getVoterWeight(voterID)
		totalWeight.Add(totalWeight, weight)

		if vote.Vote {
			trueWeight.Add(trueWeight, weight)
		} else {
			falseWeight.Add(falseWeight, weight)
		}
	}

	return
}

// GetFameVotingStats آمار کلی fame voting
func (fv *FameVoting) GetFameVotingStats() map[string]interface{} {
	stats := make(map[string]interface{})

	totalDecided := 0
	totalFamous := 0
	totalNotFamous := 0

	for _, event := range fv.dag.Events {
		if fv.state.Decided[event.Hash()] {
			totalDecided++
			if event.IsFamous != nil && *event.IsFamous {
				totalFamous++
			} else {
				totalNotFamous++
			}
		}
	}

	stats["total_decided"] = totalDecided
	stats["total_famous"] = totalFamous
	stats["total_not_famous"] = totalNotFamous
	stats["fame_ratio"] = float64(totalFamous) / float64(totalDecided)

	return stats
}
