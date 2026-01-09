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

	// Avoid division by zero later
	if maxCpuTime == 0 {
		maxCpuTime = 1
	}
	if maxRssKb == 0 {
		maxRssKb = 1
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

func UsersByScore(procs []processes.Process) []userStats {
	sorted := aggregate(
		procs,
		func(p processes.Process) string { return p.Username },
		func(s stats) userStats { return userStats{stats: s} },
	)

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
		return cmp.Compare(ui.name, uj.name)
	})

	return sorted
}

func BinariesByScore(procs []processes.Process) []binaryStats {
	sorted := aggregate(
		procs,
		func(p processes.Process) string { return p.Command },
		func(s stats) binaryStats { return binaryStats{stats: s} },
	)

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
		return cmp.Compare(bi.name, bj.name)
	})

	return sorted
}

func aggregate[T any](processes []processes.Process, getGroup func(p processes.Process) string, cast func(stat stats) T) []T {
	statsMap := make(map[string]stats)
	for _, p := range processes {
		stat, exists := statsMap[getGroup(p)]
		if !exists {
			stat = stats{name: getGroup(p)}
		}

		if p.CpuTime != nil {
			stat.cpuTime += *p.CpuTime
		}
		stat.rssKb += p.RssKb

		stat.processCount++
		statsMap[getGroup(p)] = stat
	}

	var returnMe []T
	for _, stats := range statsMap {
		returnMe = append(returnMe, cast(stats))
	}

	return returnMe
}
