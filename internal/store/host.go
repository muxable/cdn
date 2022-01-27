package store

import (
	"fmt"
	"os"
)

// GetLocalAddress returns the local address of the host.
func GetLocalAddress() (string, error) {
	// Check if we're deployed on fly.io.
	flyAllocID := os.Getenv("FLY_ALLOC_ID")
	flyAppName := os.Getenv("FLY_APP_NAME")
	if flyAllocID != "" && flyAppName != "" {
		return fmt.Sprintf("%s.vm.%s.internal", flyAppName, flyAllocID), nil
	}

	return "", fmt.Errorf("unable to determine local address")
}