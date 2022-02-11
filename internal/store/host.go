package store

import (
	"fmt"
	"os"

	"go.uber.org/zap"
)

// GetLocalAddress returns the local address of the host.
func GetLocalAddress() (string, error) {
	// Check if we're deployed on fly.io.
	flyAllocID := os.Getenv("FLY_ALLOC_ID")
	flyAppName := os.Getenv("FLY_APP_NAME")
	if flyAllocID != "" && flyAppName != "" {
		return fmt.Sprintf("%s.vm.%s.internal", flyAllocID, flyAppName), nil
	}

	zap.L().Warn("could not resolve address, assuming local")
	return "127.0.0.1", nil
}

func GetTag() string {
	region := os.Getenv("FLY_REGION")
	if region == "" {
		return "local"
	}
	return region
}