package ui

import (
	"math"
	"unicode"
)

// Given some rows with some number of columns, return the widths required for
// each column.
func ColumnWidths(rows [][]string, maxWidth int) []int {
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

	for {
		totalWidth := 0
		for _, w := range widths {
			totalWidth += w
		}
		if totalWidth <= maxWidth {
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
					// Nothing here, we can shrink this cell for free
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
			if cost < minCost {
				minCost = cost
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
