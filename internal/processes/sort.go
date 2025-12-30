package processes

import "sort"

func ByCpuPercent(processes []*Process) []*Process {
	sorted := make([]*Process, len(processes))
	copy(sorted, processes)

	sort.Slice(sorted, func(i, j int) bool {
		pi := sorted[i]
		pj := sorted[j]

		var cpuI, cpuJ float64
		if pi.cpuPercent != nil {
			cpuI = *pi.cpuPercent
		}
		if pj.cpuPercent != nil {
			cpuJ = *pj.cpuPercent
		}

		return cpuI > cpuJ
	})

	return sorted
}
