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
#include <sys/types.h>
#include <sys/sysctl.h>
#include <unistd.h>
#include <stdbool.h>

// Provide a small helper that calls sysctl for VM_SWAPUSAGE and returns the
// xsu_used field in bytes. Returns UINT64_MAX on failure.
uint64_t ftop_get_swap_used() {
	int mib[2];
	mib[0] = CTL_VM;
	mib[1] = VM_SWAPUSAGE;

	struct xsw_usage xsu;
	size_t size = sizeof(xsu);
	if (sysctl(mib, 2, &xsu, &size, NULL, 0) != 0) {
		return (uint64_t)UINT64_MAX;
	}

	// xsu.xsu_used is provided in bytes already (uint64_t)
	return (uint64_t)xsu.xsu_used;
}

typedef struct {
	uint64_t free_count;
	uint64_t active_count;
	uint64_t inactive_count;
	uint64_t wire_count;
	uint64_t compressor_page_count;
	uint64_t anonymous_count;
	uint64_t purgeable_count;
} ftop_vm_counts_t;

int ftop_get_vm_counts(ftop_vm_counts_t *out) {
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

int ftop_get_page_size(uint32_t *out_page_size) {
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
	if kr := C.ftop_get_page_size(&c_page_size); kr != C.KERN_SUCCESS {
		err = fmt.Errorf("ftop_get_page_size failed: %d", kr)
		return
	}

	var counts C.ftop_vm_counts_t
	if kr := C.ftop_get_vm_counts(&counts); kr != C.KERN_SUCCESS {
		err = fmt.Errorf("ftop_get_vm_counts failed: %d", kr)
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
	wanted_ram_pages := (pages_anonymous - pages_purgeable + pages_wired + pages_compressed)
	usedBytes = wanted_ram_pages * page_size_bytes

	// Swapped out memory is also in use. If it was not in use, the OS would
	// just throw it away, not swap it to disk.
	swapUsedBytes, err := getSwapUsedBytes()
	if err != nil {
		return
	}
	usedBytes += swapUsedBytes

	return
}

func getSwapUsedBytes() (uint64, error) {
	used := uint64(C.ftop_get_swap_used())
	if used == ^uint64(0) {
		return 0, fmt.Errorf("ftop_get_swap_used returned 0 and textual fallback failed")
	}

	return used, nil
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
