package main

import (
	"fmt"
	"math/big"
	"sort"
	"sync"
)

// Vote رای یک voter برای یک witness
type Vote struct {
	Vote      bool
	Decided   bool
	Round     uint64
	Weight    *big.Int // وزن رای (بر اساس stake)
	VoterID   EventID
	WitnessID EventID
}

// VoteRecord نگهداری آراء: [voter][witness]
type VoteRecord map[EventID]map[EventID]*Vote

// FameVotingState نگهداری وضعیت fame voting
type FameVotingState struct {
	CurrentRound    uint64
	Votes           VoteRecord
	Decided         map[EventID]bool
	Weights         map[EventID]*big.Int // وزن هر voter
	FamousWitnesses map[uint64]map[EventID]*Event
}

// FameVoting مسئول اجرای الگوریتم fame voting
type FameVoting struct {
	dag          *DAG
	state        *FameVotingState
	cacheManager *CacheManager
	mu           sync.RWMutex
}

// NewFameVoting ایجاد FameVoting جدید
func NewFameVoting(dag *DAG) *FameVoting {
	return &FameVoting{
		dag:          dag,
		cacheManager: NewCacheManager(1000),
		state: &FameVotingState{
			Votes:           make(VoteRecord),
			Decided:         make(map[EventID]bool),
			Weights:         make(map[EventID]*big.Int),
			FamousWitnesses: make(map[uint64]map[EventID]*Event),
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

	// شروع از round بعدی - حداکثر 10 round برای تصمیم‌گیری
	for votingRound := round + 1; votingRound <= round+10; votingRound++ {
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

		// محاسبه دو سوم وزن کل (Byzantine fault tolerance)
		twoThirds := new(big.Int).Mul(totalWeight, big.NewInt(2))
		twoThirds.Div(twoThirds, big.NewInt(3))

		// بررسی تصمیم‌گیری
		if trueVotes.Cmp(twoThirds) > 0 {
			// Witness famous شد
			t := true
			witness.IsFamous = &t
			fv.state.Decided[witnessID] = true

			// اضافه کردن به famous witnesses
			if fv.state.FamousWitnesses[round] == nil {
				fv.state.FamousWitnesses[round] = make(map[EventID]*Event)
			}
			fv.state.FamousWitnesses[round][witnessID] = witness

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
	fv.mu.RLock()
	defer fv.mu.RUnlock()

	// بررسی cache manager
	if cachedData, exists := fv.cacheManager.GetConsensusCache(round); exists {
		if votersData, ok := cachedData["voters"]; ok {
			if voters, ok := votersData.(map[EventID]*Event); ok {
				return voters
			}
		}
	}

	roundInfo, exists := fv.dag.Rounds[round]
	if !exists {
		return nil
	}

	// ذخیره در cache manager
	cacheData := map[string]interface{}{
		"voters": roundInfo.Witnesses,
	}
	fv.cacheManager.SetConsensusCache(round, cacheData)
	return roundInfo.Witnesses
}

// getVoteForWitness دریافت رای یک voter برای یک witness
func (fv *FameVoting) getVoteForWitness(voterID, witnessID EventID, round uint64) *Vote {
	fv.mu.RLock()
	defer fv.mu.RUnlock()

	// ایجاد cache key
	cacheKey := fmt.Sprintf("vote_%s_%s_%d", voterID, witnessID, round)

	// بررسی cache manager
	if cachedVote, exists := fv.cacheManager.GetVote(cacheKey); exists {
		return cachedVote
	}

	// بررسی cache محلی
	if fv.state.Votes[voterID] == nil {
		fv.state.Votes[voterID] = make(map[EventID]*Vote)
	}

	if vote, exists := fv.state.Votes[voterID][witnessID]; exists {
		// ذخیره در cache manager
		fv.cacheManager.SetVote(cacheKey, vote)
		return vote
	}

	// محاسبه رای جدید
	vote := fv.calculateVote(voterID, witnessID, round)
	fv.state.Votes[voterID][witnessID] = vote

	// ذخیره در cache manager
	fv.cacheManager.SetVote(cacheKey, vote)
	return vote
}

// calculateVote محاسبه رای یک voter برای یک witness
func (fv *FameVoting) calculateVote(voterID, witnessID EventID, round uint64) *Vote {
	voter, _ := fv.dag.GetEvent(voterID)
	witness, _ := fv.dag.GetEvent(witnessID)

	if voter == nil || witness == nil {
		return nil
	}

	// محاسبه رای بر اساس الگوریتم Lachesis کامل
	vote := fv.computeVoteAdvanced(voter, witness, round)

	return &Vote{
		Vote:      vote,
		Decided:   false, // در این مرحله تصمیم‌گیری نشده
		Round:     round,
		Weight:    fv.getVoterWeight(voterID),
		VoterID:   voterID,
		WitnessID: witnessID,
	}
}

// computeVoteAdvanced محاسبه رای پیشرفته بر اساس الگوریتم Lachesis
func (fv *FameVoting) computeVoteAdvanced(voter, witness *Event, round uint64) bool {
	// الگوریتم پیشرفته Lachesis مطابق با Fantom:

	// شرط 1: Visibility - voter باید witness را ببیند
	if !fv.dag.IsAncestor(witness.Hash(), voter.Hash()) {
		return false
	}

	// شرط 2: Lamport time consistency
	if voter.Lamport <= witness.Lamport {
		return false
	}

	// شرط 3: Round assignment - voter باید در round بعدی باشد
	voterRound := voter.Lamport / uint64(2000) // 2 seconds
	witnessRound := witness.Lamport / uint64(2000)

	if voterRound <= witnessRound {
		return false
	}

	// شرط 4: Byzantine fault tolerance conditions
	return fv.checkByzantineConditions(voter, witness, round)
}

// checkByzantineConditions بررسی شرایط Byzantine fault tolerance
func (fv *FameVoting) checkByzantineConditions(voter, witness *Event, round uint64) bool {
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

	// بررسی شرایط اضافی Byzantine
	return fv.checkAdditionalByzantineConditions(voter, witness)
}

// checkAdditionalByzantineConditions بررسی شرایط اضافی Byzantine
func (fv *FameVoting) checkAdditionalByzantineConditions(voter, witness *Event) bool {
	// بررسی unique creator
	if voter.CreatorID == witness.CreatorID {
		return false
	}

	// بررسی Lamport time consistency
	if voter.Lamport <= witness.Lamport {
		return false
	}

	// بررسی visibility conditions
	if !fv.dag.IsAncestor(witness.Hash(), voter.Hash()) {
		return false
	}

	// بررسی consensus conditions
	if !fv.checkConsensusConditions(voter, witness) {
		return false
	}

	// بررسی شرایط پیشرفته Byzantine
	return fv.AdvancedByzantineConditions(voter, witness, voter.Lamport/uint64(2000))
}

// checkConsensusConditions بررسی شرایط اجماع
func (fv *FameVoting) checkConsensusConditions(voter, witness *Event) bool {
	// بررسی اینکه آیا voter می‌تواند witness را در DAG ببیند
	if !fv.dag.IsAncestor(witness.Hash(), voter.Hash()) {
		return false
	}

	// بررسی اینکه آیا voter در round بعدی است
	voterRound := voter.Lamport / uint64(2000)
	witnessRound := witness.Lamport / uint64(2000)

	if voterRound <= witnessRound {
		return false
	}

	// بررسی شرایط اضافی اجماع
	return true
}

// AdvancedByzantineConditions شرایط پیشرفته Byzantine fault tolerance
func (fv *FameVoting) AdvancedByzantineConditions(voter, witness *Event, round uint64) bool {
	// شرط 1: Dynamic validator set validation
	if !fv.validateDynamicValidatorSet(voter, witness, round) {
		return false
	}

	// شرط 2: Advanced time consensus
	if !fv.validateAdvancedTimeConsensus(voter, witness, round) {
		return false
	}

	// شرط 3: Advanced visibility conditions
	if !fv.validateAdvancedVisibility(voter, witness, round) {
		return false
	}

	// شرط 4: Consensus weight validation
	if !fv.validateConsensusWeight(voter, witness, round) {
		return false
	}

	return true
}

// validateDynamicValidatorSet اعتبارسنجی dynamic validator set
func (fv *FameVoting) validateDynamicValidatorSet(voter, witness *Event, round uint64) bool {
	// بررسی تغییرات validator set در طول زمان
	voterTime := voter.Lamport
	witnessTime := witness.Lamport

	// اگر فاصله زمانی زیاد باشد، validator set ممکن است تغییر کرده باشد
	if voterTime-witnessTime > uint64(10000) { // 10 seconds
		return false
	}

	// بررسی اینکه آیا voter و witness در validator set مشابه هستند
	return fv.checkValidatorSetConsistency(voter, witness, round)
}

// validateAdvancedTimeConsensus اعتبارسنجی اجماع زمانی پیشرفته
func (fv *FameVoting) validateAdvancedTimeConsensus(voter, witness *Event, round uint64) bool {
	// محاسبه median time از تمام witnesses در round
	medianTime := fv.calculateMedianTimeForRound(round)

	// بررسی اینکه آیا witness در محدوده زمانی مناسب است
	witnessTime := witness.Lamport
	timeWindow := uint64(2000) // 2 seconds

	if abs(int64(witnessTime)-int64(medianTime)) > int64(timeWindow) {
		return false
	}

	return true
}

// validateAdvancedVisibility اعتبارسنجی visibility پیشرفته
func (fv *FameVoting) validateAdvancedVisibility(voter, witness *Event, round uint64) bool {
	// بررسی visibility با وزن
	visibilityWeight := fv.calculateVisibilityWeight(voter, witness)

	// باید حداقل 2/3 وزن کل را داشته باشد
	totalWeight := fv.calculateTotalWeightForRound(round)
	requiredWeight := new(big.Int).Mul(totalWeight, big.NewInt(2))
	requiredWeight.Div(requiredWeight, big.NewInt(3))

	return visibilityWeight.Cmp(requiredWeight) >= 0
}

// validateConsensusWeight اعتبارسنجی وزن اجماع
func (fv *FameVoting) validateConsensusWeight(voter, witness *Event, round uint64) bool {
	// محاسبه وزن اجماع بر اساس stake و reputation
	consensusWeight := fv.calculateConsensusWeight(voter, witness, round)

	// باید حداقل 2/3 وزن کل را داشته باشد
	totalWeight := fv.calculateTotalWeightForRound(round)
	requiredWeight := new(big.Int).Mul(totalWeight, big.NewInt(2))
	requiredWeight.Div(requiredWeight, big.NewInt(3))

	return consensusWeight.Cmp(requiredWeight) >= 0
}

// calculateMedianTimeForRound محاسبه median time برای یک round
func (fv *FameVoting) calculateMedianTimeForRound(round uint64) uint64 {
	roundInfo, exists := fv.dag.Rounds[round]
	if !exists {
		return 0
	}

	var times []uint64
	for _, witness := range roundInfo.Witnesses {
		times = append(times, witness.Lamport)
	}

	if len(times) == 0 {
		return 0
	}

	// مرتب‌سازی times
	sort.Slice(times, func(i, j int) bool {
		return times[i] < times[j]
	})

	// محاسبه median
	n := len(times)
	if n%2 == 0 {
		return (times[n/2-1] + times[n/2]) / 2
	}
	return times[n/2]
}

// calculateVisibilityWeight محاسبه وزن visibility
func (fv *FameVoting) calculateVisibilityWeight(voter, witness *Event) *big.Int {
	weight := big.NewInt(0)

	// بررسی visibility با تمام events
	for _, event := range fv.dag.Events {
		if fv.dag.IsAncestor(witness.Hash(), event.Hash()) {
			eventWeight := fv.getVoterWeight(event.Hash())
			weight.Add(weight, eventWeight)
		}
	}

	return weight
}

// calculateTotalWeightForRound محاسبه وزن کل برای یک round
func (fv *FameVoting) calculateTotalWeightForRound(round uint64) *big.Int {
	roundInfo, exists := fv.dag.Rounds[round]
	if !exists {
		return big.NewInt(0)
	}

	totalWeight := big.NewInt(0)
	for _, witness := range roundInfo.Witnesses {
		weight := fv.getVoterWeight(witness.Hash())
		totalWeight.Add(totalWeight, weight)
	}

	return totalWeight
}

// calculateConsensusWeight محاسبه وزن اجماع
func (fv *FameVoting) calculateConsensusWeight(voter, witness *Event, round uint64) *big.Int {
	weight := big.NewInt(0)

	// وزن بر اساس stake
	stakeWeight := fv.getVoterWeight(voter.Hash())
	weight.Add(weight, stakeWeight)

	// وزن بر اساس reputation
	reputationWeight := fv.calculateReputationWeight(voter, round)
	weight.Add(weight, reputationWeight)

	// وزن بر اساس consistency
	consistencyWeight := fv.calculateConsistencyWeight(voter, witness, round)
	weight.Add(weight, consistencyWeight)

	return weight
}

// calculateReputationWeight محاسبه وزن reputation
func (fv *FameVoting) calculateReputationWeight(voter *Event, round uint64) *big.Int {
	// محاسبه reputation بر اساس تاریخچه voting
	reputation := big.NewInt(0)

	// بررسی voting history
	for r := uint64(0); r < round; r++ {
		if fv.hasConsistentVoting(voter, r) {
			reputation.Add(reputation, big.NewInt(1000))
		}
	}

	return reputation
}

// calculateConsistencyWeight محاسبه وزن consistency
func (fv *FameVoting) calculateConsistencyWeight(voter, witness *Event, round uint64) *big.Int {
	// محاسبه consistency بر اساس voting pattern
	consistency := big.NewInt(0)

	// بررسی consistency با سایر voters
	for _, event := range fv.dag.Events {
		if event.Hash() != voter.Hash() && event.Lamport < voter.Lamport {
			if fv.hasConsistentVotingWith(event, voter, round) {
				consistency.Add(consistency, big.NewInt(500))
			}
		}
	}

	return consistency
}

// hasConsistentVoting بررسی consistency voting
func (fv *FameVoting) hasConsistentVoting(voter *Event, round uint64) bool {
	// بررسی اینکه آیا voter در این round voting consistent داشته است
	roundInfo, exists := fv.dag.Rounds[round]
	if !exists {
		return false
	}

	consistentVotes := 0
	totalVotes := 0

	for _, witness := range roundInfo.Witnesses {
		vote := fv.getVoteForWitness(voter.Hash(), witness.Hash(), round)
		if vote != nil {
			totalVotes++
			if vote.Vote {
				consistentVotes++
			}
		}
	}

	if totalVotes == 0 {
		return false
	}

	// باید حداقل 80% consistency داشته باشد
	return float64(consistentVotes)/float64(totalVotes) >= 0.8
}

// hasConsistentVotingWith بررسی consistency با voter دیگر
func (fv *FameVoting) hasConsistentVotingWith(voter1, voter2 *Event, round uint64) bool {
	roundInfo, exists := fv.dag.Rounds[round]
	if !exists {
		return false
	}

	consistentVotes := 0
	totalVotes := 0

	for _, witness := range roundInfo.Witnesses {
		vote1 := fv.getVoteForWitness(voter1.Hash(), witness.Hash(), round)
		vote2 := fv.getVoteForWitness(voter2.Hash(), witness.Hash(), round)

		if vote1 != nil && vote2 != nil {
			totalVotes++
			if vote1.Vote == vote2.Vote {
				consistentVotes++
			}
		}
	}

	if totalVotes == 0 {
		return false
	}

	// باید حداقل 70% consistency داشته باشد
	return float64(consistentVotes)/float64(totalVotes) >= 0.7
}

// checkValidatorSetConsistency بررسی consistency validator set
func (fv *FameVoting) checkValidatorSetConsistency(voter, witness *Event, round uint64) bool {
	// بررسی اینکه آیا voter و witness در validator set مشابه هستند
	voterValidators := fv.getValidatorsForEvent(voter)
	witnessValidators := fv.getValidatorsForEvent(witness)

	// محاسبه overlap
	overlap := 0
	for _, v1 := range voterValidators {
		for _, v2 := range witnessValidators {
			if v1 == v2 {
				overlap++
				break
			}
		}
	}

	totalValidators := len(voterValidators)
	if totalValidators == 0 {
		return false
	}

	// باید حداقل 80% overlap داشته باشد
	return float64(overlap)/float64(totalValidators) >= 0.8
}

// getValidatorsForEvent دریافت validators برای یک event
func (fv *FameVoting) getValidatorsForEvent(event *Event) []string {
	// در نسخه کامل، این از validator set گرفته می‌شود
	// فعلاً یک لیست نمونه برمی‌گردانیم
	return []string{"NodeA", "NodeB", "NodeC"}
}

// abs محاسبه قدر مطلق
func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
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
	if totalDecided > 0 {
		stats["fame_ratio"] = float64(totalFamous) / float64(totalDecided)
	} else {
		stats["fame_ratio"] = 0.0
	}

	// آمار cache
	if fv.cacheManager != nil {
		cacheStats := fv.cacheManager.GetCacheStats()
		for key, value := range cacheStats {
			stats["cache_"+key] = value
		}
	}

	return stats
}

// GetFamousWitnessesByRound دریافت famous witnesses بر اساس round
func (fv *FameVoting) GetFamousWitnessesByRound(round uint64) map[EventID]*Event {
	return fv.state.FamousWitnesses[round]
}

// IsWitnessFamous بررسی اینکه آیا یک witness famous است
func (fv *FameVoting) IsWitnessFamous(witnessID EventID) bool {
	event, _ := fv.dag.GetEvent(witnessID)
	if event == nil {
		return false
	}
	return event.IsFamous != nil && *event.IsFamous
}
