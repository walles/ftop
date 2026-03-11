package log

import "sync"

var panicShutdownHook func()
var panicShutdownHookLock sync.Mutex
var panicShutdownOnce sync.Once

// Register a callback to trigger on recovered panic.
//
// Intended for fail-fast shutdown in case any goroutine panics.
func SetPanicShutdownHook(hook func()) {
	panicShutdownHookLock.Lock()
	defer panicShutdownHookLock.Unlock()

	panicShutdownHook = hook
}

// goroutineName will be shown in crash reports
//
// recoverResult should be the result of a recover() call. This is either nil,
// meaning there was no panic, or it will be logged as an explanation of what
// went wrong.
//
// Usage example, put this first in all goroutines. Make sure to use a unique
// and helpful goroutine name:
//
//	defer func() {
//	    log.PanicHandler("mygoroutine", recover(), debug.Stack())
//	}()
func PanicHandler(goroutineName string, recoverResult any, stackTrace []byte) {
	if recoverResult == nil {
		return
	}

	Errorf("Goroutine <%s> crashed: %s\n%s", goroutineName, recoverResult, string(stackTrace))

	panicShutdownOnce.Do(func() {
		panicShutdownHookLock.Lock()
		hook := panicShutdownHook
		panicShutdownHookLock.Unlock()
		if hook == nil {
			return
		}

		hook()
	})
}
