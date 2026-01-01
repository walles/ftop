package processes

import (
	"cmp"
	"slices"
	"sort"
)

func ProcessesByCpuUsage(processes []Process) []Process {
	sorted := make([]Process, len(processes))
	copy(sorted, processes)

	sort.Slice(sorted, func(i, j int) bool {
		pi := sorted[i]
		pj := sorted[j]

		if pi.cpuTime != nil && pj.cpuTime != nil {
			// For a stable(r) sort, compare by how much CPU time they've used
			if *pi.cpuTime != *pj.cpuTime {
				return *pi.cpuTime > *pj.cpuTime
			}
		}

		// Unknown CPU usage, sort by RAM usage. We could go for CPU percentage,
		// but that's very unstable and makes for a jarring user experience.
		return pi.rssKb > pj.rssKb
	})

	return sorted
}

func UsersByCpuUsage(processes []Process) []userStats {
	perUser := aggregatePerUser(processes)

	slices.SortFunc(perUser, func(i, j userStats) int {
		byCpuTime := cmp.Compare(i.cpuTime, j.cpuTime)
		if byCpuTime != 0 {
			return -byCpuTime
		}

		// Before we have any CPU times, the count will give a similar ordering
		byProcessCount := cmp.Compare(i.processCount, j.processCount)
		if byProcessCount != 0 {
			return -byProcessCount
		}

		// Sorting by memory usage stabilizes the bottom of the list
		byMemoryUsage := cmp.Compare(i.rssKb, j.rssKb)
		if byMemoryUsage != 0 {
			return -byMemoryUsage
		}

		// Fall back on user names to get a stable result at the very end of the
		// list.
		return cmp.Compare(i.username, j.username)
	})

	return perUser
}

func ProcessesByMemoryUsage(processes []Process) []Process {
	sorted := make([]Process, len(processes))
	copy(sorted, processes)

	sort.Slice(sorted, func(i, j int) bool {
		pi := sorted[i]
		pj := sorted[j]

		return pi.rssKb > pj.rssKb
	})

	return sorted
}

func UsersByMemoryUsage(processes []Process) []userStats {
	perUser := aggregatePerUser(processes)

	slices.SortFunc(perUser, func(i, j userStats) int {
		return -cmp.Compare(i.rssKb, j.rssKb)
	})

	return perUser
}
