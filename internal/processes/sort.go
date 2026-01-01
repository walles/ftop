package processes

import (
	"sort"
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

	sort.Slice(sorted, func(i int, j int) bool {
		pi := processes[i]
		pj := processes[j]

		var cpuScoreI float64
		if pi.cpuTime != nil {
			cpuScoreI = pi.cpuTime.Seconds() / maxCpuTime.Seconds()
		}
		memScoreI := float64(pi.rssKb) / float64(maxRssKb)

		var cpuScoreJ float64
		if pi.cpuTime != nil {
			cpuScoreJ = pj.cpuTime.Seconds() / maxCpuTime.Seconds()
		}
		memScoreJ := float64(pj.rssKb) / float64(maxRssKb)

		primaryI := max(memScoreI, cpuScoreI)
		secondaryI := min(memScoreI, cpuScoreI)

		primaryJ := max(memScoreJ, cpuScoreJ)
		secondaryJ := min(memScoreJ, cpuScoreJ)

		if primaryI < primaryJ {
			return true
		}
		return secondaryI < secondaryJ
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

	sort.Slice(sorted, func(i, j int) bool {
		ui := sorted[i]
		uj := sorted[j]

		cpuScoreI := ui.cpuTime.Seconds() / maxCpuTime.Seconds()
		memScoreI := float64(ui.rssKb) / float64(maxRssKb)

		cpuScoreJ := uj.cpuTime.Seconds() / maxCpuTime.Seconds()
		memScoreJ := float64(uj.rssKb) / float64(maxRssKb)

		primaryI := max(memScoreI, cpuScoreI)
		secondaryI := min(memScoreI, cpuScoreI)

		primaryJ := max(memScoreJ, cpuScoreJ)
		secondaryJ := min(memScoreJ, cpuScoreJ)

		if primaryI < primaryJ {
			return true
		}
		return secondaryI >= secondaryJ
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
