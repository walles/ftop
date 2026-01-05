package ui

import (
	"testing"

	"github.com/walles/ptop/internal/assert"
)

func TestColumnWidthsSingleNarrowing(t *testing.T) {
	assert.SlicesEqual(t, ColumnWidths([][]string{{"PID"}}, 3, true), []int{3})
	assert.SlicesEqual(t, ColumnWidths([][]string{{"PID"}}, 2, true), []int{2})
}

func TestColumnWidthsSingleWidening(t *testing.T) {
	assert.SlicesEqual(t, ColumnWidths([][]string{{"PID"}}, 4, true), []int{4})
}

func TestColumnWidthDoubleNarrowing(t *testing.T) {
	assert.SlicesEqual(t, ColumnWidths([][]string{{"PID", "POD"}}, 6, true), []int{3, 3})
	assert.SlicesEqual(t, ColumnWidths([][]string{{"PID", "POD"}}, 4, true), []int{2, 2})
	assert.SlicesEqual(t, ColumnWidths([][]string{{"PID", "POD"}}, 2, true), []int{1, 1})
}

func TestColumnWidthsDoubleWidening(t *testing.T) {
	assert.SlicesEqual(t, ColumnWidths([][]string{{"PID", "POD"}}, 8, true), []int{4, 4})
	assert.SlicesEqual(t, ColumnWidths([][]string{{"PID", "POD"}}, 10, true), []int{5, 5})

	uneven := ColumnWidths([][]string{{"PID", "POD"}}, 9, true)
	assert.Equal(t, 2, len(uneven))
	assert.Equal(t, 9, uneven[0]+uneven[1])
}
