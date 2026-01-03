package ptop

import (
	"testing"

	"github.com/walles/ptop/internal/assert"
)

func TestAveragesToGraphString(t *testing.T) {
	assert.Equal(t, averagesToGraphString(0.0, 0.0, 0.0), "⢀⣀⣀⣀⣀⣀⣀⣀")
}
