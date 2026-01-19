//go:build darwin

package sysload

// FIXME: Would this file work for other BSDs as well? If so, update the build
// directives.

/*
#include <stdlib.h>
#include <stdint.h>
#include <mach/mach.h>
#include <mach/mach_host.h>
#include <mach/vm_statistics.h>

typedef struct {
    uint64_t free_count;
    uint64_t active_count;
    uint64_t inactive_count;
    uint64_t wire_count;
    uint64_t compressor_page_count;
    uint64_t anonymous_count;
    uint64_t purgeable_count;
} ptop_vm_counts_t;

int ptop_get_vm_counts(ptop_vm_counts_t *out) {
    vm_statistics64_data_t stats;
    mach_msg_type_number_t count = HOST_VM_INFO64_COUNT;
    kern_return_t kr = host_statistics64(mach_host_self(), HOST_VM_INFO64, (host_info_t)&stats, &count);
    if (kr != KERN_SUCCESS) return (int)kr;

    out->free_count = stats.free_count;
    out->active_count = stats.active_count;
    out->inactive_count = stats.inactive_count;
    out->wire_count = stats.wire_count;
    out->compressor_page_count = stats.compressor_page_count;

    // internal_page_count is documented in the header as "# of pages that are anonymous"
    out->anonymous_count = stats.internal_page_count;

    out->purgeable_count = stats.purgeable_count;

    return (int)KERN_SUCCESS;
}

int ptop_get_page_size(uint32_t *out_page_size) {
    vm_size_t page_size;
    kern_return_t kr = host_page_size(mach_host_self(), &page_size);
    if (kr != KERN_SUCCESS) return (int)kr;
    *out_page_size = (uint32_t)page_size;
    return (int)KERN_SUCCESS;
}

*/
import "C"

import (
	"fmt"

	"golang.org/x/sys/unix"
)

func getMemoryUsage() (usedBytes uint64, totalBytes uint64, err error) {
	var c_page_size C.uint32_t
	if kr := C.ptop_get_page_size(&c_page_size); kr != C.KERN_SUCCESS {
		err = fmt.Errorf("ptop_get_page_size failed: %d", kr)
		return
	}

	var counts C.ptop_vm_counts_t
	if kr := C.ptop_get_vm_counts(&counts); kr != C.KERN_SUCCESS {
		err = fmt.Errorf("ptop_get_vm_counts failed: %d", kr)
		return
	}

	page_size_bytes := uint64(c_page_size)

	pages_free := uint64(counts.free_count)
	pages_active := uint64(counts.active_count)
	pages_inactive := uint64(counts.inactive_count)
	pages_wired := uint64(counts.wire_count)
	pages_compressed := uint64(counts.compressor_page_count)

	pages_anonymous := uint64(counts.anonymous_count)
	pages_purgeable := uint64(counts.purgeable_count)

	// In experiments, this has added up well to the amount of physical RAM in
	// my machine
	total_pages := (pages_free + pages_active + pages_inactive + pages_wired + pages_compressed)
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

func getCpuCoreCounts() (coresLogical int, coresPhysical int, err error) {
	physicalCount, err := unix.SysctlUint32("hw.physicalcpu")
	if err != nil {
		return 0, 0, err
	}

	logicalCount, err := unix.SysctlUint32("hw.logicalcpu")
	if err != nil {
		return 0, 0, err
	}

	coresPhysical = int(physicalCount)
	coresLogical = int(logicalCount)
	return
}
