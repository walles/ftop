# Combined Display

Instead of one CPU and one Memory section, let's combine them.

# Ordering

1. Check max RAM and max CPU for all processes.
2. Compute a fraction for each process: (current CPU / max CPU) + (current RAM / max RAM)
3. Primary sort key is the highest fraction, secondary is the lowest fraction.

# Load Bars

Rather than having one bar in each direction as we did in the
`johan/go-with-combined-display` branch, let's try having two overlapping bars
from the left.

The shorter one will be drawn on top of the longer one so that both are visible.

Color wise, I'm thinking start at the same grayish color, but CPU should top out
at red and memory at green. Or whatever, just as long as then max colors are
clearly distinguishable.
