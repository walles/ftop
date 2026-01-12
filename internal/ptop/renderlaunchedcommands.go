package ptop

import (
	"strconv"

	"github.com/walles/moor/v2/twin"
	"github.com/walles/ptop/internal/processes"
)

func renderLaunchedCommands(screen twin.Screen, launches *processes.LaunchNode, y0, y1 int) {
	width, _ := screen.Size()
	rightBorder := width - 1
	defer renderFrame(screen, 0, y0, rightBorder, y1, "Launched Commands")

	if launches == nil {
		return
	}

	launchSlices := launches.Flatten()

	x0s := []int{}     // Starting screen X positions for each column
	xAfters := []int{} // Screen X positions after each column

	// FIXME: Should we color the text using a topBottomRamp like the other tables?

	x0 := 1

	// Precompute min/max row indices for each node at each column
	maxCols := 0
	for _, p := range launchSlices {
		if len(p) > maxCols {
			maxCols = len(p)
		}
	}

	// maps[col][node] = [minRow, maxRow]
	maps := make([]map[*processes.LaunchNode][2]int, maxCols)
	for c := 0; c < maxCols; c++ {
		maps[c] = make(map[*processes.LaunchNode][2]int)
	}

	for r, p := range launchSlices {
		for c, node := range p {
			if rng, ok := maps[c][node]; ok {
				if r < rng[0] {
					rng[0] = r
				}
				if r > rng[1] {
					rng[1] = r
				}
				maps[c][node] = rng
			} else {
				maps[c][node] = [2]int{r, r}
			}
		}
	}

	for rowIndex, path := range launchSlices {
		x := x0
		y := y0 + 1 + rowIndex
		if y >= y1 {
			// Reached bottom of our allocated area
			return
		}

		hidingRedundantColumns := true
		for column, node := range path {
			if hidingRedundantColumns {
				if rowIndex > 0 {
					// Check if we're still drawing the same column as the row above
					rowAbove := launchSlices[rowIndex-1]
					if column < len(rowAbove) && path[column] == rowAbove[column] {
						// Same node pointer as above, draw vertical continuation if needed
						if column < len(x0s) {
							// Only draw verticals for non-root columns
							if column > 0 {
								if rng, ok := maps[column][node]; ok {
									// draw vertical if current row is within the node's min..max range (after first occurrence)
									if rowIndex > rng[0] && rowIndex <= rng[1] {
										// Prefer drawing the vertical under the parent's separator if available
										if column > 0 && column-1 < len(xAfters) {
											screen.SetCell(xAfters[column-1], y, twin.StyledRune{Rune: '│', Style: twin.StyleDefault})
										} else {
											screen.SetCell(x0s[column], y, twin.StyledRune{Rune: '│', Style: twin.StyleDefault})
										}
									}
								}
							}

							// advance to the column's after position
							x = xAfters[column]
							continue
						}
					}
				}

				hidingRedundantColumns = false
				x0s = x0s[:column]
				xAfters = xAfters[:column]
			}

			// Invariant: No more columns to hide, draw this!

			// If not first column, decide separator/marker to draw before this column.
			if column > 0 {
				// Ensure x starts at previous column's after, if known
				if column-1 < len(xAfters) {
					x = xAfters[column-1]
				}

				sepRune := '─'

				// Determine whether parent (previous column) is hidden in this row
				parentHidden := false
				if rowIndex > 0 && column-1 < len(launchSlices[rowIndex-1]) && path[column-1] == launchSlices[rowIndex-1][column-1] {
					parentHidden = true
				}

				if parentHidden {
					// Parent is hidden: use sibling markers (├ or └) depending on parent's child index
					parent := path[column-1]
					if parent != nil && len(parent.Children) > 1 {
						// find child's index among parent's children by pointer
						lastIdx := len(parent.Children) - 1
						childIdx := 0
						for i, ch := range parent.Children {
							if ch == node {
								childIdx = i
								break
							}
						}

						if childIdx == lastIdx {
							sepRune = '└'
						} else {
							sepRune = '├'
						}
					}
				} else {
					// Parent printed on this row: if parent has multiple children, parent shows a fork mark (┬)
					parent := path[column-1]
					if parent != nil && len(parent.Children) > 1 {
						sepRune = '┬'
					}
				}

				sep := string(sepRune) + "▶"
				x += drawText(screen, x, y, rightBorder, sep, twin.StyleDefault)
			}

			style := twin.StyleDefault
			if node.LaunchCount > 0 {
				style = style.WithAttr(twin.AttrBold)
			}

			// Remember start of this column, draw command and optional count, record after position.
			x0s = append(x0s, x)
			x += drawText(screen, x, y, rightBorder, node.Command, style)

			if node.LaunchCount > 0 {
				x += drawText(screen, x, y, rightBorder, "("+strconv.Itoa(node.LaunchCount)+")", twin.StyleDefault)
			}
			xAfters = append(xAfters, x)
		}
	}
}
