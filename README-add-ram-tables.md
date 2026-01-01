# Adding RAM Sorted Tables to the ptop UI

Logically, this is what we have to do to render:

1. Decide on the heights of the new sections.
1. Decide on the contents of all four sections
1. Make one large table with columns for both the left (per-process) and right
   (per-user) sides.
1. Decide on column widths for the combined table.
1. Render the top (by CPU) part first
1. Render a divider row
1. Render the bottom (by RAM) part second
1. Render a status row at the bottom of the screen
