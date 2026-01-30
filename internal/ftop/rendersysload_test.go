package ftop

import (
	"testing"

	"github.com/walles/ftop/internal/assert"
)

func TestAveragesToGraphString(t *testing.T) {
	assert.Equal(t, averagesToGraphString(0.0, 0.0, 0.0), "⢀⣀⣀⣀⣀⣀⣀⣀")
}
