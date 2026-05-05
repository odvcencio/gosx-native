//go:build smoke && !darwin

package e2e

import "testing"

func TestCounterEndToEndSkipped(t *testing.T) {
	t.Skip("iOS smoke test requires darwin, xcodegen, and xcodebuild")
}
