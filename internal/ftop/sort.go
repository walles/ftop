package ftop

import (
	"cmp"
	"slices"
	"time"

	"github.com/walles/ftop/internal/processes"
)

func SortByScore[T any](unordered []T, asStats func(t T) stats) []T {

	FIXME: I'm unconvinced nativity sorting works. Or maybe it's the collection? Add a test!

	sorted := make([]T, len(unordered))
	copy(sorted, unordered)

	maxCpuTime := time.Duration(0)
	maxRssKb := 0
	maxNativity := 0.0
	for _, u := range unordered {
		stat := asStats(u)
		if stat.cpuTime > maxCpuTime {
			maxCpuTime = stat.cpuTime
		}
		if stat.rssKb > maxRssKb {
			maxRssKb = stat.rssKb
		}
		if stat.nativity > maxNativity {
			maxNativity = stat.nativity
		}
	}

	// Avoid division by zero later
	if maxCpuTime == 0 {
		maxCpuTime = 1
	}
	if maxRssKb == 0 {
		maxRssKb = 1
	}
	if maxNativity == 0 {
		maxNativity = 1
	}

	scoresI := make([]float64, 3)
	scoresJ := make([]float64, 3)
	slices.SortFunc(sorted, func(ui T, uj T) int {
		statsI := asStats(ui)
		statsJ := asStats(uj)

		scoresI[0] = float64(statsI.cpuTime) / float64(maxCpuTime)
		scoresI[1] = float64(statsI.rssKb) / float64(maxRssKb)
		scoresI[2] = statsI.nativity / maxNativity

		scoresJ[0] = float64(statsJ.cpuTime) / float64(maxCpuTime)
		scoresJ[1] = float64(statsJ.rssKb) / float64(maxRssKb)
		scoresJ[2] = statsJ.nativity / maxNativity

		slices.SortFunc(scoresI, func(si, sj float64) int {
			// Negate to put highest scores first
			return -cmp.Compare(si, sj)
		})
		slices.SortFunc(scoresJ, func(si, sj float64) int {
			// Negate to put highest scores first
			return -cmp.Compare(si, sj)
		})

		for k := range 3 {
			if scoresI[k] != scoresJ[k] {
				// Negate to put highest scores first
				return -cmp.Compare(scoresI[k], scoresJ[k])
			}
		}

		// Fall back to name comparison for stability when the other scores are equal
		return cmp.Compare(statsI.name, statsJ.name)
	})

	// Put the top CPU process at the top of the list. I believe this is what
	// people expect. I do for example.
	topThree := sorted
	if len(sorted) > 3 {
		topThree = sorted[:3]
	}
	slices.SortFunc(topThree, func(ui T, uj T) int {
		si := asStats(ui)
		sj := asStats(uj)
		return -cmp.Compare(si.cpuTime, sj.cpuTime)
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
		stat.nativity += p.Nativity

		statsMap[getGroup(p)] = stat
	}

	var returnMe []T
	for _, stats := range statsMap {
		returnMe = append(returnMe, cast(stats))
	}

	return returnMe
}
