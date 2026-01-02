package ui

import (
	"testing"

	"github.com/walles/ptop/internal/assert"
)

func TestColumnWidthsSingle(t *testing.T) {
	assert.SliceEqual(t, ColumnWidths([][]string{{"PID"}}, 50), []int{3})
	assert.SliceEqual(t, ColumnWidths([][]string{{"PID"}}, 3), []int{3})
	assert.SliceEqual(t, ColumnWidths([][]string{{"PID"}}, 2), []int{2})
}

func TestColumnWidthDouble(t *testing.T) {
	assert.SliceEqual(t, ColumnWidths([][]string{{"PID", "POD"}}, 6), []int{3, 3})
	assert.SliceEqual(t, ColumnWidths([][]string{{"PID", "POD"}}, 4), []int{2, 2})
	assert.SliceEqual(t, ColumnWidths([][]string{{"PID", "POD"}}, 2), []int{1, 1})
}
