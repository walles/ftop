package ptop

import (
	"cmp"
	"slices"
	"time"

	"github.com/walles/ptop/internal/processes"
)

func ProcessesByScore(procs []processes.Process) []processes.Process {
	sorted := make([]processes.Process, len(procs))
	copy(sorted, procs)

	maxCpuTime := time.Duration(0)
	maxRssKb := 0
	for _, p := range procs {
		if p.CpuTime != nil && *p.CpuTime > maxCpuTime {
			maxCpuTime = *p.CpuTime
		}
		if p.RssKb > maxRssKb {
			maxRssKb = p.RssKb
		}
	}

	slices.SortFunc(sorted, func(pi processes.Process, pj processes.Process) int {
		var cpuScoreI float64
		if pi.CpuTime != nil {
			cpuScoreI = pi.CpuTime.Seconds() / maxCpuTime.Seconds()
		}
		memScoreI := float64(pi.RssKb) / float64(maxRssKb)

		var cpuScoreJ float64
		if pj.CpuTime != nil {
			cpuScoreJ = pj.CpuTime.Seconds() / maxCpuTime.Seconds()
		}
		memScoreJ := float64(pj.RssKb) / float64(maxRssKb)

		primaryI := max(memScoreI, cpuScoreI)
		secondaryI := min(memScoreI, cpuScoreI)

		primaryJ := max(memScoreJ, cpuScoreJ)
		secondaryJ := min(memScoreJ, cpuScoreJ)

		primaryCmp := cmp.Compare(primaryI, primaryJ)
		if primaryCmp != 0 {
			return -primaryCmp
		}

		secondaryCmp := cmp.Compare(secondaryI, secondaryJ)
		if secondaryCmp != 0 {
			return -secondaryCmp
		}

		// Fall back to command name comparison for stability at the top of the list
		nameCmp := cmp.Compare(pi.Command, pj.Command)
		if nameCmp != 0 {
			return nameCmp
		}

		// Finally, sort by PID to get a stable sort
		return cmp.Compare(pi.Pid, pj.Pid)
	})

	return sorted
}

func UsersByScore(processes []processes.Process) []userStats {
	sorted := aggregatePerUser(processes)

	maxCpuTime := time.Duration(0)
	maxRssKb := 0
	for _, u := range sorted {
		if u.cpuTime > maxCpuTime {
			maxCpuTime = u.cpuTime
		}
		if u.rssKb > maxRssKb {
			maxRssKb = u.rssKb
		}
	}

	slices.SortFunc(sorted, func(ui userStats, uj userStats) int {
		cpuScoreI := ui.cpuTime.Seconds() / maxCpuTime.Seconds()
		memScoreI := float64(ui.rssKb) / float64(maxRssKb)

		cpuScoreJ := uj.cpuTime.Seconds() / maxCpuTime.Seconds()
		memScoreJ := float64(uj.rssKb) / float64(maxRssKb)

		primaryI := max(memScoreI, cpuScoreI)
		secondaryI := min(memScoreI, cpuScoreI)

		primaryJ := max(memScoreJ, cpuScoreJ)
		secondaryJ := min(memScoreJ, cpuScoreJ)

		primaryCmp := cmp.Compare(primaryI, primaryJ)
		if primaryCmp != 0 {
			return -primaryCmp
		}

		secondaryCmp := cmp.Compare(secondaryI, secondaryJ)
		if secondaryCmp != 0 {
			return -secondaryCmp
		}

		// Fall back to name comparison for stability at the bottom of the list
		return cmp.Compare(ui.username, uj.username)
	})

	return sorted
}

func BinariesByScore(processes []processes.Process) []binaryStats {
	sorted := aggregatePerBinary(processes)

	maxCpuTime := time.Duration(0)
	maxRssKb := 0
	for _, b := range sorted {
		if b.cpuTime > maxCpuTime {
			maxCpuTime = b.cpuTime
		}
		if b.rssKb > maxRssKb {
			maxRssKb = b.rssKb
		}
	}

	slices.SortFunc(sorted, func(bi binaryStats, bj binaryStats) int {
		cpuScoreI := bi.cpuTime.Seconds() / maxCpuTime.Seconds()
		memScoreI := float64(bi.rssKb) / float64(maxRssKb)

		cpuScoreJ := bj.cpuTime.Seconds() / maxCpuTime.Seconds()
		memScoreJ := float64(bj.rssKb) / float64(maxRssKb)

		primaryI := max(memScoreI, cpuScoreI)
		secondaryI := min(memScoreI, cpuScoreI)

		primaryJ := max(memScoreJ, cpuScoreJ)
		secondaryJ := min(memScoreJ, cpuScoreJ)

		primaryCmp := cmp.Compare(primaryI, primaryJ)
		if primaryCmp != 0 {
			return -primaryCmp
		}

		secondaryCmp := cmp.Compare(secondaryI, secondaryJ)
		if secondaryCmp != 0 {
			return -secondaryCmp
		}

		// Fall back to name comparison for stability at the bottom of the list
		return cmp.Compare(bi.binaryName, bj.binaryName)
	})

	return sorted
}

func aggregatePerUser(processes []processes.Process) []userStats {
	userMap := make(map[string]userStats)
	for _, p := range processes {
		userStat, exists := userMap[p.Username]
		if !exists {
			userStat = userStats{username: p.Username}
		}

		if p.CpuTime != nil {
			userStat.cpuTime += *p.CpuTime
		}
		userStat.rssKb += p.RssKb

		userStat.processCount++

		userMap[p.Username] = userStat
	}

	var returnMe []userStats
	for _, stats := range userMap {
		returnMe = append(returnMe, stats)
	}

	return returnMe
}

func aggregatePerBinary(processes []processes.Process) []binaryStats {
	binaryMap := make(map[string]binaryStats)
	for _, p := range processes {
		binaryStat, exists := binaryMap[p.Command]
		if !exists {
			binaryStat = binaryStats{binaryName: p.Command}
		}

		if p.CpuTime != nil {
			binaryStat.cpuTime += *p.CpuTime
		}
		binaryStat.rssKb += p.RssKb

		binaryStat.processCount++
		binaryMap[p.Command] = binaryStat
	}

	var returnMe []binaryStats
	for _, stats := range binaryMap {
		returnMe = append(returnMe, stats)
	}

	return returnMe
}
