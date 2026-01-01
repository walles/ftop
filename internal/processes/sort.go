package processes

import (
	"cmp"
	"slices"
	"time"
)

func ProcessesByScore(processes []Process) []Process {
	sorted := make([]Process, len(processes))
	copy(sorted, processes)

	maxCpuTime := time.Duration(0)
	maxRssKb := 0
	for _, p := range processes {
		if p.cpuTime != nil && *p.cpuTime > maxCpuTime {
			maxCpuTime = *p.cpuTime
		}
		if p.rssKb > maxRssKb {
			maxRssKb = p.rssKb
		}
	}

	slices.SortFunc(sorted, func(pi Process, pj Process) int {
		var cpuScoreI float64
		if pi.cpuTime != nil {
			cpuScoreI = pi.cpuTime.Seconds() / maxCpuTime.Seconds()
		}
		memScoreI := float64(pi.rssKb) / float64(maxRssKb)

		var cpuScoreJ float64
		if pj.cpuTime != nil {
			cpuScoreJ = pj.cpuTime.Seconds() / maxCpuTime.Seconds()
		}
		memScoreJ := float64(pj.rssKb) / float64(maxRssKb)

		primaryI := max(memScoreI, cpuScoreI)
		secondaryI := min(memScoreI, cpuScoreI)

		primaryJ := max(memScoreJ, cpuScoreJ)
		secondaryJ := min(memScoreJ, cpuScoreJ)

		primaryCmp := cmp.Compare(primaryI, primaryJ)
		if primaryCmp != 0 {
			return -primaryCmp
		}
		return -cmp.Compare(secondaryI, secondaryJ)
	})

	return sorted
}

func UsersByScore(processes []Process) []userStats {
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
		return -cmp.Compare(secondaryI, secondaryJ)
	})

	return sorted
}

func aggregatePerUser(processes []Process) []userStats {
	userMap := make(map[string]userStats)
	for _, p := range processes {
		stats, exists := userMap[p.username]
		if !exists {
			stats = userStats{username: p.username}
		}

		if p.cpuTime != nil {
			stats.cpuTime += *p.cpuTime
		}
		stats.rssKb += p.rssKb

		stats.processCount++

		userMap[p.username] = stats
	}

	var returnMe []userStats
	for _, stats := range userMap {
		returnMe = append(returnMe, stats)
	}

	return returnMe
}
