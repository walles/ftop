package processes

/*
#include <libproc.h>
#include <sys/fcntl.h>
#include <sys/proc_info.h>

typedef struct {
	uint32_t openflags;
	uint64_t handle;
} ftop_pipe_info_t;

// Asks the kernel how fd of pid is open and which pipe it is an end of. Returns
// 0 on success and -1 if the kernel won't say, which covers a process owned by
// somebody else, a process that has exited, and a descriptor that is no pipe.
int ftop_get_pipe_info(int pid, int fd, ftop_pipe_info_t *out) {
	struct pipe_fdinfo info;
	int size = proc_pidfdinfo(pid, fd, PROC_PIDFDPIPEINFO, &info, sizeof(info));
	if (size < (int)sizeof(info)) {
		return -1;
	}

	out->openflags = info.pfi.fi_openflags;
	out->handle = info.pipeinfo.pipe_handle;
	return 0;
}
*/
import "C"

import (
	"strconv"
	"strings"
)

// Fills in the access mode of every anonymous pipe end in the map, macOS lsof
// reporting none for those, and leaves the ends lsof did report a mode for
// alone. Modifies the PipeEnds in place.
//
// This is what gets macOS its arrows: without it every anonymous pipe is
// DirectionUnknown, so `tail -f /etc/services | sort` renders
// `tail(40504) ◀?▶ sort(40505)` where the truth is plainly `tail ──▶ sort`.
//
// Not a matter of asking lsof for the right field. lsof's file-flag option +fg
// fills that column in for CHR and REG while leaving it blank for PIPE, and the
// FD number carries no "r"/"w" suffix either. The kernel does know, and
// proc_pidfdinfo() hands over the very pipe end lsof already named rather than a
// second identity to join on: its pipe_handle is the same number lsof prints as
// the DEVICE, measured equal on 307 of 307 pipe ends on a quiet laptop.
//
// Cheap enough not to think about. One syscall per anonymous pipe end, 317 of
// them on that same laptop, came in under the resolution of /usr/bin/time
// against the 0.39 s the lsof listing that produced the map spends.
//
// Degrades to PipeAccessUnknown per end, which is what the map already held, so
// a failure costs the arrow and nothing else. proc_pidfdinfo() enforces a
// same-uid-or-root check and returns EPERM for another user's process — 30 of 30
// probes of root's PID 1, as a normal user — but no end we are handed can hit
// that: non-root lsof reports no other user's pipes in the first place, and as
// root the check passes. What does happen is ESRCH, 6 of 317 on that laptop,
// from processes that exited between the lsof fork and these calls.
func fillInAccessModes(pipeEndsByPid map[int][]PipeEnd) {
	for pid, ends := range pipeEndsByPid {
		for i := range ends {
			end := &ends[i]
			if end.Device == "" {
				// A named FIFO, which lsof reports the access mode for itself.
				// Only a macOS anonymous pipe carries a kernel address here,
				// see PipeEnd.Device.
				continue
			}

			access := kernelAccessMode(pid, *end)
			if access == PipeAccessUnknown {
				// Leave whatever lsof reported rather than overwriting it with
				// nothing. Nothing is what it reported for these ends today, so
				// this costs no arrow anybody would otherwise have seen.
				continue
			}

			end.Access = access
		}
	}
}

// How the kernel says the descriptor end arrived on is open, or
// PipeAccessUnknown where it won't say.
//
// Unknown also for an end that is no longer the one lsof reported: the fd may
// have been closed and reopened since the fork, and then the flags describe some
// other file entirely. Requiring the pipe the kernel names to be the pipe lsof
// named is what catches that.
func kernelAccessMode(pid int, end PipeEnd) PipeAccess {
	fd, err := strconv.Atoi(end.Fd)
	if err != nil {
		// lsof names some descriptors rather than numbering them, "cwd" and
		// "txt" among them. No pipe arrives that way.
		return PipeAccessUnknown
	}

	// Parsed rather than compared as text, so that neither lsof's spelling of a
	// kernel address nor the width the kernel hands it back in can matter.
	lsofDevice, err := strconv.ParseUint(strings.TrimPrefix(end.Device, "0x"), 16, 64)
	if err != nil {
		return PipeAccessUnknown
	}

	var info C.ftop_pipe_info_t
	if C.ftop_get_pipe_info(C.int(pid), C.int(fd), &info) != 0 {
		return PipeAccessUnknown
	}

	if uint64(info.handle) != lsofDevice {
		return PipeAccessUnknown
	}

	// fi_openflags carries more than these two — O_NONBLOCK and
	// FCNTL_FS_SPECIFIC_BASE both turned up on that laptop — so mask rather than
	// compare.
	canRead := info.openflags&C.FREAD != 0
	canWrite := info.openflags&C.FWRITE != 0

	if canRead && canWrite {
		// Not a shape an anonymous pipe end comes in: 0 of 307 reported both.
		// A FIFO can be opened that way, but no FIFO reaches this far.
		return PipeAccessReadWrite
	}

	if canRead {
		return PipeAccessRead
	}

	if canWrite {
		return PipeAccessWrite
	}

	return PipeAccessUnknown
}
