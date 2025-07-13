package core

import "sort"

func (d *DAG) FinalizeEvents() {
	for round, info := range d.Rounds {
		for eid := range info.Witnesses {
			e, ok := d.GetEvent(eid)
			if !ok || !e.IsClotho || e.Atropos != [32]byte{} {
				continue
			}

			// بررسی Witnessهای round+2
			rp2 := round + 2
			nextInfo, exists := d.Rounds[rp2]
			if !exists {
				continue
			}

			decided := true
			trueVotes := 0
			total := 0

			for wid := range nextInfo.Witnesses {
				voter, _ := d.GetEvent(wid)
				if !d.IsAncestor(eid, wid) {
					continue
				}
				voteMap := d.Votes[wid]
				vote, exists := voteMap[eid]
				if !exists || !vote.Decided {
					decided = false
					break
				}
				if vote.Vote {
					trueVotes++
				}
				total++
			}

			if !decided || total == 0 {
				continue
			}

			if float64(trueVotes) >= (2.0 / 3.0 * float64(total)) {
				//  تبدیل به Atropos
				e.Atropos = e.Hash()
				e.RoundReceived = rp2

				// محاسبه MedianTime از timestamp تمام رأی‌دهندگان
				times := []uint64{}
				for wid := range nextInfo.Witnesses {
					if d.IsAncestor(eid, wid) {
						voter, _ := d.GetEvent(wid)
						times = append(times, voter.Lamport)
					}
				}
				e.AtroposTime = median(times)
			}
		}
	}
}


func median(values []uint64) uint64 {
	if len(values) == 0 {
		return 0
	}
	sort.Slice(values, func(i, j int) bool {
		return values[i] < values[j]
	})
	mid := len(values) / 2
	if len(values)%2 == 0 {
		return (values[mid-1] + values[mid]) / 2
	}
	return values[mid]
}
