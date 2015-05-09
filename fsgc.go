// Written in 2015 by Dmitry Chestnykh.
//
// To the extent possible under law, the author have dedicated all copyright
// and related and neighboring rights to this software to the public domain
// worldwide. This software is distributed without any warranty.
// http://creativecommons.org/publicdomain/zero/1.0/

// Package fsgc provides a garbage collector for gorilla/session FilesystemStore.
//
// It collects old sessions based on file modification timestamps.
//
// Example:
//
//   path := "/path/to/sessions/"
//   store := sessions.NewFilesystemStore(path, []byte("secret"))
//   gc := fsgc.New(path).MaxAge(12 * time.Hour).Interval(30 * time.Minute)
//   gc.Start()
//   //
//   // Every 30 minutes gc will remove any sessions files older than 12 hours
//   // from /path/to/sessions/. When shutting down server, stop the collector.
//   //
//   gc.Stop()
//
package fsgc

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// GC is a garbage collector.
type GC struct {
	mu       sync.Mutex
	dir      string
	maxAge   time.Duration
	interval time.Duration
	ticker   *time.Ticker
}

const (
	// DefaultMaxAge is a default age of session file.
	// If the session is older, it will be removed.
	DefaultMaxAge = 7 * 24 * time.Hour

	// DefaultInterval is the default interval between garbage collections
	// (the collector will run every hour).
	DefaultInterval = 1 * time.Hour
)

// New returns a new collector, which will remove expired sessions
// from the given directory. It must be started by calling Start.
//
// The session is considered expired when its file modification date
// is older than DefaultMaxAge. To set a different age, call MaxAge.
//
// The garbage collector will try to collect every DefaultInterval.
// To set a different interval between collections, call Interval.
func New(dir string) *GC {
	return &GC{
		dir:      dir,
		maxAge:   DefaultMaxAge,
		interval: DefaultInterval,
	}
}

// MaxAge sets the max age for the session and returns the same GC.
func (gc *GC) MaxAge(dur time.Duration) *GC {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	gc.maxAge = dur
	return gc
}

// Interval sets the interval between collections and returns the same GC.
func (gc *GC) Interval(dur time.Duration) *GC {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	gc.interval = dur
	return gc
}

// Start starts the garbage collector. It returns the same GC.
//
// The collector runs on its own goroutine, and must be stopped by calling Stop
// when it is no longer needed.
//
// The first collection will happen after the set interval.
func (gc *GC) Start() *GC {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	if gc.ticker != nil {
		return gc // already started
	}
	gc.ticker = time.NewTicker(gc.interval)
	go func() {
		for _ = range gc.ticker.C {
			gc.Collect() // ignore error
		}
	}()
	return gc
}

// Stop stops the garbage collector.
// It can be restarted again by calling Start.
func (gc *GC) Stop() {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	if gc.ticker == nil {
		return // not started
	}
	gc.ticker.Stop()
	gc.ticker = nil
}

// Collect runs the garbage collection immediately.
func (gc *GC) Collect() error {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	f, err := os.Open(gc.dir)
	if err != nil {
		return err
	}
	defer f.Close()
	fis, err := f.Readdir(0)
	if err != nil {
		return err
	}
	now := time.Now()
	for _, fi := range fis {
		if fi.IsDir() || !strings.HasPrefix(fi.Name(), "session_") {
			continue
		}
		if now.Sub(fi.ModTime()) > gc.maxAge {
			// Session file expired, delete it.
			// Ignore errors.
			os.Remove(filepath.Join(gc.dir, fi.Name()))
		}
	}
	return nil
}
