package processes

import "sort"

func ByCpuUsage(processes []*Process) []*Process {
	sorted := make([]*Process, len(processes))
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
		return pi.rss_kb > pj.rss_kb
	})

	return sorted
}
