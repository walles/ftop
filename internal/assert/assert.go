package assert

import "testing"

func Equal[V comparable](t *testing.T, actual, expected V) {
	t.Helper()

	if expected != actual {
		t.Errorf(
			"\nExpected: %v\nActual:   %v", expected, actual)
	}
}
