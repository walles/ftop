package ui

import (
	"math"
	"unicode"
)

// Given some rows with some number of columns, return the widths required for
// each column for the sum to reach target width.
//
// If growFirstColumn is false, the first column will be ignored when there is
// extra space to distribute.
func ColumnWidths(rows [][]string, targetWidth int, growFirstColumn bool) []int {
	widths := []int{}

	// Collect maximum column widths
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
		return growColumns(widths, targetWidth, growFirstColumn)
	}

	// Shrink columns
	for sumWidths(widths) > targetWidth {
		// Now, let's say we shorten one width. How much information would we
		// lose?
		narrowingCosts := make([]float64, len(widths))
		for _, row := range rows {
			for column, cell := range row {
				if widths[column] == 0 {
					// Already at 0 width, can't shrink this column
					narrowingCosts[column] = math.MaxFloat64
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

				// Dropping 100% of the column (dropping column 1 of 1) is a
				// total loss, and we give that a cost of 1.0. Dropping 1 out of
				// 2 is half as bad, etc.
				cost := 1.0 / float64(dropColumn+1)
				narrowingCosts[column] += cost
			}
		}

		// Find the column with the least cost to narrow
		minCost := math.MaxFloat64
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

func growColumns(widths []int, targetWidth int, growFirstColumn bool) []int {
	if !growFirstColumn {
		return append(widths[:1], growColumns(widths[1:], targetWidth-widths[0], true)...)
	}

	currentWidth := sumWidths(widths)
	missingWidth := targetWidth - currentWidth
	addPerColumn := missingWidth / len(widths)
	for i := range widths {
		widths[i] += addPerColumn
	}

	remainingToAdd := targetWidth - sumWidths(widths)
	for i := range remainingToAdd {
		widths[i%len(widths)]++
	}

	return widths
}
