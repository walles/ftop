//go:build darwin && cgo

package sysload

/*
#include <stdlib.h>
*/
import "C"

import "fmt"

func getLoadAverages() (load1M float64, load5M float64, load15M float64, err error) {
	// Use libc getloadavg(3) via cgo. It returns number of samples retrieved
	// or -1 on error. We'll request 3 samples.
	var loads [3]C.double
	n := C.getloadavg(&loads[0], 3)
	if n == -1 {
		err = fmt.Errorf("getloadavg failed")
		return
	}

	// n may be 1..3 depending on system; set available values and keep others zero
	if n >= 1 {
		load1M = float64(loads[0])
	}

	if n >= 2 {
		load5M = float64(loads[1])
	}

	if n >= 3 {
		load15M = float64(loads[2])
	}

	return
}
