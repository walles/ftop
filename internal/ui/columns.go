package ui

import (
	"math"
	"unicode"
)

// Given some rows with some number of columns, return the widths required for
// each column for the sum to reach target width.
func ColumnWidths(rows [][]string, targetWidth int) []int {
	widths := []int{}

	for _, row := range rows {
		for cellIndex, cell := range row {
			if cellIndex >= len(widths) {
				widths = append(widths, 0)
			}

			if len(cell) > widths[cellIndex] {
				widths[cellIndex] = len(cell)
			}
		}
	}

	if sumWidths(widths) == targetWidth {
		return widths
	}

	if sumWidths(widths) < targetWidth {
		return growColumns(widths, targetWidth)
	}

	for {
		if sumWidths(widths) == targetWidth {
			break
		}

		// Now, let's say we shorten one width. How many character containing
		// cells would we lose?
		narrowingCosts := make([]int, len(widths))
		for _, row := range rows {
			for column, cell := range row {
				if widths[column] == 0 {
					// Already at 0 width, can't shrink this column
					narrowingCosts[column] = math.MaxInt
					continue
				}

				if len(cell) < widths[column] {
					// Contents is shorter than the current width, no cost in
					// this cell for shrinking this column
					continue
				}

				dropColumn := widths[column] - 1
				dropRune := cell[dropColumn]
				if unicode.IsSpace(rune(dropRune)) {
					// Dropping whitespace is free
					continue
				}

				narrowingCosts[column]++
			}
		}

		// Find the column with the least cost to narrow
		minCost := math.MaxInt
		minCostColumn := -1
		for column, cost := range narrowingCosts {
			if cost > minCost {
				continue
			}

			if cost < minCost {
				minCost = cost
				minCostColumn = column
				continue
			}

			// Tie-breaker: Narrow the wider column
			if widths[column] > widths[minCostColumn] {
				minCostColumn = column
			}
		}

		if minCostColumn == -1 {
			panic("cannot narrow any more")
		}

		widths[minCostColumn]--
	}

	return widths
}

func sumWidths(widths []int) int {
	total := 0
	for _, w := range widths {
		total += w
	}
	return total
}

func growColumns(widths []int, targetWidth int) []int {
	currentWidth := sumWidths(widths)
	missingWidth := targetWidth - currentWidth
	addPerColumn := missingWidth / len(widths)
	for i := range widths {
		widths[i] += addPerColumn
	}

	remainingToAdd := targetWidth - sumWidths(widths)
	for i := 0; i < remainingToAdd; i++ {
		widths[i%len(widths)]++
	}

	return widths
}
