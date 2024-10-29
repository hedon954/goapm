package goapm

import (
	"context"
	"errors"

	"github.com/google/gops/agent"
	"mosn.io/holmes"
)

// AutoPProfOpt is the options for auto pprof.
type AutoPProfOpt struct {
	// EnableCPU enables cpu pprof.
	EnableCPU bool
	// EnableMem enables mem pprof.
	EnableMem bool
	// EnableGoroutine enables goroutine pprof.
	EnableGoroutine bool
}

type autoPProfReporter struct{}

func (a *autoPProfReporter) Report(pType string, buf []byte, reason, eventID string) error {
	Logger.Error(context.TODO(), "homesGen", errors.New("auto record running state failed"),
		map[string]any{
			"pType":   pType,
			"reason":  reason,
			"eventID": eventID,
		},
	)
	return nil
}

// NewHomes creates a holmes dumper.
func NewHomes(autoPProfOpts *AutoPProfOpt, opts ...holmes.Option) (*holmes.Holmes, error) {
	if err := agent.Listen(agent.Options{
		ShutdownCleanup: true,
	}); err != nil {
		return nil, err
	}

	h, err := holmes.New(append(opts, holmes.WithProfileReporter(&autoPProfReporter{}))...)
	if err != nil {
		return nil, err
	}

	if autoPProfOpts != nil {
		if autoPProfOpts.EnableCPU {
			h.EnableCPUDump()
		}
		if autoPProfOpts.EnableMem {
			h.EnableMemDump()
		}
		if autoPProfOpts.EnableGoroutine {
			h.EnableGoroutineDump()
		}
	}
	return h, nil
}
