# Combined Display

Instead of one CPU and one Memory section, let's combine them.

# Ordering

1. Check max RAM and max CPU for all processes.
2. Compute a fraction for each process: (current CPU / max CPU) + (current RAM / max RAM)
3. Primary sort key is the highest fraction, secondary is the lowest fraction.

# Load Bars

Each row gets two load bars extending from the middle. CPU fraction extends to
the left, RAM fraction extends to the right.

This goes for both the per-process list and for the per-user list.
