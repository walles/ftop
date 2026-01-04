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

	baselineTime time.Time
	currentTime  time.Time

	// Highest throughput seen so far per device
	peakThroughputs map[string]float64
}

type Stat struct {
	DeviceName     string
	BytesPerSecond float64
	HighWatermark  float64 // Max value of BytesPerSecond seen so far
}

func NewTracker() *Tracker {
	tracker := &Tracker{
		peakThroughputs: make(map[string]float64),
	}

	// Periodically update the IO stats
	go func() {
		defer func() {
			log.PanicHandler("IO tracker", recover(), debug.Stack())
		}()

		tracker.update() // Initial update

		// Get the first deltas quickly
		time.Sleep(800 * time.Millisecond)
		tracker.update()

		// Periodic updates
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

	// FIXME: Get disk IO stats as well

	now := time.Now()

	tracker.mutex.Lock()
	if tracker.baseline == nil {
		// First iteration
		tracker.baseline = networkStats
		tracker.baselineTime = now
	}

	tracker.current = networkStats
	tracker.currentTime = now

	tracker.mutex.Unlock()
}

func (tracker *Tracker) Stats() []Stat {
	tracker.mutex.Lock()
	defer tracker.mutex.Unlock()

	var returnMe []Stat
	elapsedSeconds := tracker.currentTime.Sub(tracker.baselineTime).Seconds()
	for deviceName, currentBytes := range tracker.current {
		baselineBytes, ok := tracker.baseline[deviceName]
		if !ok {
			baselineBytes = 0
		}
		bytesPerSecond := 0.0
		if elapsedSeconds > 0 {
			bytesPerSecond = float64(currentBytes-baselineBytes) / elapsedSeconds
		}

		peakThroughput := tracker.peakThroughputs[deviceName]
		if bytesPerSecond > peakThroughput {
			peakThroughput = bytesPerSecond
			tracker.peakThroughputs[deviceName] = peakThroughput
		}

		returnMe = append(returnMe, Stat{
			DeviceName:     deviceName,
			BytesPerSecond: bytesPerSecond,
			HighWatermark:  peakThroughput,
		})
	}

	return returnMe
}
