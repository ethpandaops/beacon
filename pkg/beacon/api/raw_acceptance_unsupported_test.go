//go:build acceptance && !linux

package api

import "testing"

func TestAcceptanceRequiresLinux(t *testing.T) {
	t.Skip("acceptance RSS measurement requires Linux")
}
