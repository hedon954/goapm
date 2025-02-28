package apm

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mosn.io/holmes"
)

func TestNewHomes_Creation(t *testing.T) {
	// Create a temporary directory for profiles
	tmpDir, err := os.MkdirTemp("", "holmes-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create Holmes with options
	h, err := NewHomes(
		&AutoPProfOpt{
			EnableCPU:       true,
			EnableMem:       true,
			EnableGoroutine: true,
		},
		holmes.WithDumpPath(tmpDir),
	)

	// Verify Holmes was created successfully
	assert.NoError(t, err)
	assert.NotNil(t, h)
}

func TestAutoPProfReporter(t *testing.T) {
	apr := &autoPProfReporter{}
	err := apr.Report(
		"cpu",
		"cpu.pprof",
		holmes.ReasonCurlLessMin,
		"123456",
		time.Now(),
		[]byte("test"),
		holmes.Scene{},
	)
	assert.NoError(t, err)
}
