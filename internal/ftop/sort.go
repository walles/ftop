package ftop

import (
	"cmp"
	"slices"
	"time"

	"github.com/walles/ftop/internal/processes"
)

func SortByScore[T any](unordered []T, asStats func(t T) stats) []T {
	sorted := make([]T, len(unordered))
	copy(sorted, unordered)

	maxCpuTime := time.Duration(0)
	maxRssKb := 0
	for _, u := range unordered {
		stat := asStats(u)
		if stat.cpuTime > maxCpuTime {
			maxCpuTime = stat.cpuTime
		}
		if stat.rssKb > maxRssKb {
			maxRssKb = stat.rssKb
		}
	}

	// Avoid division by zero later
	if maxCpuTime == 0 {
		maxCpuTime = 1
	}
	if maxRssKb == 0 {
		maxRssKb = 1
	}

	slices.SortFunc(sorted, func(ui T, uj T) int {
		si := asStats(ui)
		sj := asStats(uj)

		cpuScoreI := si.cpuTime.Seconds() / maxCpuTime.Seconds()
		memScoreI := float64(si.rssKb) / float64(maxRssKb)

		cpuScoreJ := sj.cpuTime.Seconds() / maxCpuTime.Seconds()
		memScoreJ := float64(sj.rssKb) / float64(maxRssKb)

		primaryI := max(memScoreI, cpuScoreI)
		secondaryI := min(memScoreI, cpuScoreI)

		primaryJ := max(memScoreJ, cpuScoreJ)
		secondaryJ := min(memScoreJ, cpuScoreJ)

		primaryCmp := cmp.Compare(primaryI, primaryJ)
		if primaryCmp != 0 {
			return -primaryCmp
		}

		// If primary scores are equal, prefer the process where CPU is the
		// dominant contributor (cpuScore >= memScore). This puts the top CPU
		// process at the top of the list, which I believe is what people
		// expect. I do for example.
		isCpuPrimaryI := cpuScoreI >= memScoreI
		isCpuPrimaryJ := cpuScoreJ >= memScoreJ
		if isCpuPrimaryI != isCpuPrimaryJ {
			if isCpuPrimaryI {
				return -1
			}
			return 1
		}

		secondaryCmp := cmp.Compare(secondaryI, secondaryJ)
		if secondaryCmp != 0 {
			return -secondaryCmp
		}

		// Fall back to name comparison for stability when the other scores are equal
		return cmp.Compare(si.name, sj.name)
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

		statsMap[getGroup(p)] = stat
	}

	var returnMe []T
	for _, stats := range statsMap {
		returnMe = append(returnMe, cast(stats))
	}

	return returnMe
}
