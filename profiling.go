package main

import (
	"fmt"
	"log/slog"
	"os"
	"runtime"

	"github.com/grafana/pyroscope-go"
)

type profilerCloser struct {
	stop func() error
}

func (p profilerCloser) Close() error {
	return p.stop()
}

func pyroscopeServerAddress() string {
	if address := os.Getenv("PYROSCOPE_SERVER_ADDRESS"); address != "" {
		return address
	}

	return "http://127.0.0.1:4040"
}

func startDevelopmentProfiler() (profilerCloser, error) {
	// Collect the full set of profiles Pyroscope supports for Go when running locally.
	runtime.SetMutexProfileFraction(1)
	runtime.SetBlockProfileRate(1)

	profiler, err := pyroscope.Start(pyroscope.Config{
		ApplicationName: "focusd.backend",
		ServerAddress:   pyroscopeServerAddress(),
		Logger:          pyroscope.StandardLogger,
		Tags: map[string]string{
			"env": "dev",
		},
		ProfileTypes: []pyroscope.ProfileType{
			pyroscope.ProfileCPU,
			pyroscope.ProfileInuseObjects,
			pyroscope.ProfileAllocObjects,
			pyroscope.ProfileInuseSpace,
			pyroscope.ProfileAllocSpace,
			pyroscope.ProfileGoroutines,
			pyroscope.ProfileMutexCount,
			pyroscope.ProfileMutexDuration,
			pyroscope.ProfileBlockCount,
			pyroscope.ProfileBlockDuration,
		},
	})
	if err != nil {
		return profilerCloser{}, fmt.Errorf("start pyroscope profiler: %w", err)
	}

	slog.Info(
		"pyroscope profiling enabled",
		"server_address", pyroscopeServerAddress(),
		"application_name", "focusd.backend",
	)

	return profilerCloser{stop: profiler.Stop}, nil
}
