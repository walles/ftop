package ui

import (
	"testing"

	"github.com/walles/ptop/internal/assert"
)

func TestColumnWidths(t *testing.T) {
	assert.SliceEqual(t, ColumnWidths([][]string{{"PID"}}, 50), []int{3})
	assert.SliceEqual(t, ColumnWidths([][]string{{"PID"}}, 3), []int{3})
	assert.SliceEqual(t, ColumnWidths([][]string{{"PID"}}, 2), []int{2})
}
