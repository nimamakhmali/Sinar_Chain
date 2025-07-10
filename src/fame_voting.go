package core

import (
	"crypto/sha256"
)

type Vote struct {
	Vote    bool
	Decided bool
}

type VoteRecord map[EventID]map[EventID]*Vote // [voter][witness]

func (d *DAG) StartFameVoting() {
	maxRound := uint64(0)
	for r := range d.Rounds {
		if r > maxRound {
			maxRound = r
		}
	}

	for r := uint64(0); r < maxRound; r++ {
		witnesses := d.Rounds[r].Witnesses
		for wid := range witnesses {
			w, _ := d.GetEvent(wid)
			if w.IsFamous != nil {
				continue
			}

			for round := r + 1; round <= maxRound; round++ {
				voters := d.Rounds[round].Witnesses
				trueVotes := 0
				falseVotes := 0

				for voterID := range voters {
					voter, _ := d.GetEvent(voterID)

					// رأی موجود؟
					if d.Votes[voterID] == nil {
						d.Votes[voterID] = make(map[EventID]*Vote)
					}
					if v, exists := d.Votes[voterID][wid]; exists && v.Decided {
						continue
					}

					// رأی‌گیری: آیا voter می‌تونه witness رو ببینه؟
					canSee := d.IsAncestor(wid, voterID)

					// ثبت رأی
					d.Votes[voterID][wid] = &Vote{Vote: canSee, Decided: false}

					// شمارش
					if canSee {
						trueVotes++
					} else {
						falseVotes++
					}
				}

				totalVotes := trueVotes + falseVotes
				if totalVotes == 0 {
					continue
				}

				twoThirds := (2 * totalVotes) / 3

				if trueVotes > twoThirds {
					t := true
					w.IsFamous = &t
					markAllDecided(d, wid)
					break
				} else if falseVotes > twoThirds {
					f := false
					w.IsFamous = &f
					markAllDecided(d, wid)
					break
				}

				// Coin Round
				if round%2 == 0 && totalVotes > 0 {
					h := sha256.Sum256(wid[:])
					randomBit := h[len(h)-1] & 1
					if randomBit == 1 {
						t := true
						w.IsFamous = &t
					} else {
						f := false
						w.IsFamous = &f
					}
					markAllDecided(d, wid)
					break
				}
			}
		}
	}
}

func markAllDecided(d *DAG, wid EventID) {
	for voterID := range d.Votes {
		if v, ok := d.Votes[voterID][wid]; ok {
			v.Decided = true
		}
	}
}
