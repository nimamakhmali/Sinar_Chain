package main

// SelectClothos انتخاب Witnessهایی که تبدیل به Clotho می‌شن
func (d *DAG) SelectClothos() {
	for round, info := range d.Rounds {
		for wid := range info.Witnesses {
			w, ok := d.GetEvent(wid)
			if !ok || w.IsClotho || w.IsFamous == nil || !*w.IsFamous {
				continue
			}
			// بررسی Witnessهای round+1
			nextRound := round + 1
			nextInfo, exists := d.Rounds[nextRound]
			if !exists {
				continue
			}

			allDecided := true
			for voterID := range nextInfo.Witnesses {
				voteMap, ok := d.Votes[voterID]
				if !ok {
					allDecided = false
					break
				}
				vote, ok := voteMap[wid]
				if !ok || !vote.Decided {
					allDecided = false
					break
				}
			}

			if allDecided {
				w.IsClotho = true
			}
		}
	}
}
