package layout

// Given some rows with some number of columns, return the widths required for
// each column.
func ColumnWidths(rows [][]string) []int {
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

	return widths
}
