package io

import (
	"runtime/debug"
	"sync"
	"time"

	"github.com/walles/ptop/internal/log"
)

type Tracker struct {
	mutex sync.Mutex

	// Maps are from device names to the number of bytes transferred since
	// whenever.
	baseline map[string]uint64
	current  map[string]uint64
}

func NewTracker() *Tracker {
	tracker := &Tracker{}

	// Periodically update the IO stats
	go func() {
		defer func() {
			log.PanicHandler("IO tracker", recover(), debug.Stack())
		}()

		tracker.update() // Initial update
		for range time.NewTicker(3 * time.Second).C {
			tracker.update()
		}
	}()

	return tracker
}

func (tracker *Tracker) update() {
	networkStats, err := GetNetworkStats()
	if err != nil {
		log.Errorf("failed to get network stats: %v", err)
		return
	}

	tracker.mutex.Lock()
	if tracker.baseline == nil {
		// First iteration
		tracker.baseline = networkStats
	}
	tracker.current = networkStats
	tracker.mutex.Unlock()
}
