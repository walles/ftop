package sysload

// FIXME: Maybe this file should be for different BSD flavors as well?

func getMemoryUsage() (usedBytes uint64, totalBytes uint64, err error) {
	page_size_bytes := FIXME
	pages_free := FIXME
	pages_active := FIXME
	pages_inactive := FIXME
	pages_speculative := FIXME
	pages_wired := FIXME
	pages_compressed := FIXME
	pages_anonymous := FIXME
	pages_purgeable := FIXME

	// In experiments, this has added up well to the amount of physical RAM in
	// my machine
	total_pages := (pages_free +
		pages_active +
		pages_inactive +
		pages_speculative +
		pages_wired +
		pages_compressed)
	totalBytes = total_pages * page_size_bytes

	// This matches what the Activity Monitor shows in macOS 10.15.6
	//
	// For anonymous - purgeable: https://stackoverflow.com/a/36721309/473672
	//
	// FIXME: We want to add swapped out pages to this as well, since those also
	// represent a want for pages.
	wanted_ram_pages := (pages_anonymous - pages_purgeable + pages_wired + pages_compressed)
	usedBytes = wanted_ram_pages * page_size_bytes

	return
}
