// Copyright © 2021 Joel Baranick <jbaranick@gmail.com>
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
// 	  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package lib

import (
	"context"
	"fmt"
	"github.com/jwalton/gchalk"
	"github.com/spf13/cobra"
	"github.com/thecodeteam/goodbye"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	pprofTrace "runtime/trace"
	"strings"
	"sync/atomic"
)

type ProfileMode int

const (
	cpu ProfileMode = iota
	memHeap
	memAllocs
	mutex
	block
	trace
	threadCreate
	goroutine
)

var (
	started     uint32
	modes       ProfileModes
	profilePath string
)

type ProfileModes struct {
	value   *[]ProfileMode
	changed bool
}

func (r *ProfileModes) Len() int {
	if !r.changed {
		return 0
	}
	return len(*r.value)
}

func (r *ProfileModes) Get() []ProfileMode {
	if !r.changed {
		return make([]ProfileMode, 0)
	}
	result := make([]ProfileMode, len(*r.value))
	for i, value := range *r.value {
		result[i] = value
	}
	return result
}

func (r *ProfileModes) Type() string {
	return "profileMode"
}

func (r *ProfileModes) String() string {
	result := ""
	if r.changed {
		for _, value := range *r.value {
			if len(result) > 0 {
				result = fmt.Sprintf("%s,", result)
			}
			result = fmt.Sprintf("%s%s", result, value)
		}
	}
	return result
}

func (r *ProfileModes) Set(value string) error {
	if !r.changed {
		value := make([]ProfileMode, 0)
		r.value = &value
		r.changed = true
	}
	parts := strings.Split(value, ",")
	for _, part := range parts {
		segment := strings.TrimSpace(part)
		profileMode, err := ProfileModeString(segment)
		if err != nil {
			return fmt.Errorf("%s is not one of %s", segment, strings.Join(ProfileModeStrings(), ","))
		}
		*r.value = append(*r.value, profileMode)
	}
	return nil
}

// Profile represents an active profiling session.
type Profile struct {
	closers []func()

	stopped uint32
}

func (p *Profile) Stop() {
	if !atomic.CompareAndSwapUint32(&p.stopped, 0, 1) {
		return
	}
	for _, closer := range p.closers {
		closer()
	}
	atomic.StoreUint32(&started, 0)
}

func AddProfileFlags(cmd *cobra.Command) {
	workingDir, _ := os.Getwd()
	cmd.Flags().Var(&modes, "profile-mode", "Profiling modes to enable (repeatable)")
	cmd.Flags().StringVar(&profilePath, "profile-path", workingDir, "Path to write profile results")
}

func StartProfiling() error {
	if modes.Len() == 0 || !atomic.CompareAndSwapUint32(&started, 0, 1) {
		return nil
	}

	path, err := AbsolutePath(profilePath)
	if err != nil {
		return fmt.Errorf("profile: could not determine absolute profile path, %s: %v", profilePath, err)
	}

	var prof Profile
	err = os.MkdirAll(path, 0777)
	if err != nil {
		return fmt.Errorf("profile: could not create profile path, %s: %v", profilePath, err)
	}

	logf := func(format string, args ...interface{}) {
		fmt.Printf("%s\n", gchalk.WithGrey().Sprintf(format, args...))
	}

	logfClose := func(format string, args ...interface{}) {
		fmt.Printf("%s%s\n", ResetLine, gchalk.WithGrey().Sprintf(format, args...))
	}

	for _, mode := range modes.Get() {
		switch mode {
		case cpu:
			fn := filepath.Join(profilePath, "cpu.pprof")
			f, err := os.Create(fn)
			if err != nil {
				return fmt.Errorf("profile: could not create cpu profile %q: %v", fn, err)
			}
			logf("profile: cpu profiling enabled, %s", fn)
			_ = pprof.StartCPUProfile(f)
			prof.closers = append(prof.closers, func() {
				pprof.StopCPUProfile()
				_ = f.Close()
				logfClose("profile: cpu profiling disabled, %s", fn)
			})

		case memHeap:
			fn := filepath.Join(profilePath, "mem.pprof")
			f, err := os.Create(fn)
			if err != nil {
				return fmt.Errorf("profile: could not create memory profile %q: %v", fn, err)
			}
			old := runtime.MemProfileRate
			runtime.MemProfileRate = 4096
			logf("profile: memory profiling enabled (rate %d), %s", runtime.MemProfileRate, fn)
			prof.closers = append(prof.closers, func() {
				_ = pprof.Lookup("heap").WriteTo(f, 0)
				_ = f.Close()
				runtime.MemProfileRate = old
				logfClose("profile: memory profiling disabled, %s", fn)
			})

		case memAllocs:
			fn := filepath.Join(profilePath, "allocs.pprof")
			f, err := os.Create(fn)
			if err != nil {
				return fmt.Errorf("profile: could not create memory allocs profile %q: %v", fn, err)
			}
			old := runtime.MemProfileRate
			runtime.MemProfileRate = 4096
			logf("profile: memory allocs profiling enabled (rate %d), %s", runtime.MemProfileRate, fn)
			prof.closers = append(prof.closers, func() {
				_ = pprof.Lookup("allocs").WriteTo(f, 0)
				_ = f.Close()
				runtime.MemProfileRate = old
				logfClose("profile: memory profiling disabled, %s", fn)
			})

		case mutex:
			fn := filepath.Join(profilePath, "mutex.pprof")
			f, err := os.Create(fn)
			if err != nil {
				return fmt.Errorf("profile: could not create mutex profile %q: %v", fn, err)
			}
			runtime.SetMutexProfileFraction(1)
			logf("profile: mutex profiling enabled, %s", fn)
			prof.closers = append(prof.closers, func() {
				if mp := pprof.Lookup("mutex"); mp != nil {
					_ = mp.WriteTo(f, 0)
				}
				_ = f.Close()
				runtime.SetMutexProfileFraction(0)
				logfClose("profile: mutex profiling disabled, %s", fn)
			})

		case block:
			fn := filepath.Join(profilePath, "block.pprof")
			f, err := os.Create(fn)
			if err != nil {
				return fmt.Errorf("profile: could not create block profile %q: %v", fn, err)
			}
			runtime.SetBlockProfileRate(1)
			logf("profile: block profiling enabled, %s", fn)
			prof.closers = append(prof.closers, func() {
				_ = pprof.Lookup("block").WriteTo(f, 0)
				_ = f.Close()
				runtime.SetBlockProfileRate(0)
				logfClose("profile: block profiling disabled, %s", fn)
			})

		case threadCreate:
			fn := filepath.Join(profilePath, "threadcreation.pprof")
			f, err := os.Create(fn)
			if err != nil {
				return fmt.Errorf("profile: could not create thread creation profile %q: %v", fn, err)
			}
			logf("profile: thread creation profiling enabled, %s", fn)
			prof.closers = append(prof.closers, func() {
				if mp := pprof.Lookup("threadcreate"); mp != nil {
					_ = mp.WriteTo(f, 0)
				}
				_ = f.Close()
				logfClose("profile: thread creation profiling disabled, %s", fn)
			})

		case trace:
			fn := filepath.Join(profilePath, "trace.out")
			f, err := os.Create(fn)
			if err != nil {
				return fmt.Errorf("profile: could not create trace output file %q: %v", fn, err)
			}
			if err := pprofTrace.Start(f); err != nil {
				return fmt.Errorf("profile: could not start trace: %v", err)
			}
			logf("profile: trace enabled, %s", fn)
			prof.closers = append(prof.closers, func() {
				pprofTrace.Stop()
				logfClose("profile: trace disabled, %s", fn)
			})

		case goroutine:
			fn := filepath.Join(profilePath, "goroutine.pprof")
			f, err := os.Create(fn)
			if err != nil {
				return fmt.Errorf("profile: could not create goroutine profile %q: %v", fn, err)
			}
			logf("profile: goroutine profiling enabled, %s", fn)
			prof.closers = append(prof.closers, func() {
				if mp := pprof.Lookup("goroutine"); mp != nil {
					_ = mp.WriteTo(f, 0)
				}
				_ = f.Close()
				logfClose("profile: goroutine profiling disabled, %s", fn)
			})
		}
	}

	goodbye.Register(func(ctx context.Context, sig os.Signal) {
		logfClose("\nprofile: caught interrupt, stopping profiles")
		prof.Stop()
	})

	return nil
}
