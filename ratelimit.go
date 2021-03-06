// Copyright (c) 2016,2020 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package ratelimit // import "go.uber.org/ratelimit"

import (
	"time"

	"github.com/andres-erbsen/clock"
)

// Note: This file is inspired by:
// https://github.com/prashantv/go-bench/blob/master/ratelimit

// Limiter is used to rate-limit some process, possibly across goroutines.
// The process is expected to call Take() before every iteration, which
// may block to throttle the goroutine.
type Limiter interface {
	// Take should block to make sure that the RPS is met.
	Take() time.Time
}

// Clock is the minimum necessary interface to instantiate a rate limiter with
// a clock or mock clock, compatible with clocks created using
// github.com/andres-erbsen/clock.
type Clock interface {
	Now() time.Time
	Sleep(time.Duration)
}

// config configures a limiter.
type config struct {
	clock    Clock
	maxSlack time.Duration
	per      time.Duration
	lock     bool
}

// New returns a Limiter that will limit to the given RPS.
func New(rate int, opts ...Option) Limiter {
	config := buildConfig(opts)
	if !config.lock {
		return newUnsafeBased(rate, opts...)
	}
	return newAtomicBased(rate, opts...)
}

// buildConfig combines defaults with options.
func buildConfig(opts []Option) config {
	c := config{
		clock:    clock.New(),
		maxSlack: 10,
		per:      time.Second,
		lock:     true,
	}

	for _, opt := range opts {
		opt.apply(&c)
	}
	return c
}

// Option configures a Limiter.
type Option interface {
	apply(*config)
}

type clockOption struct {
	clock Clock
}

func (o clockOption) apply(c *config) {
	c.clock = o.clock
}

// WithClock returns an option for ratelimit.New that provides an alternate
// Clock implementation, typically a mock Clock for testing.
func WithClock(clock Clock) Option {
	return clockOption{clock: clock}
}

type unLockOption struct {
}

func (uo unLockOption) apply(c *config) {
	c.lock = false
}

// WithoutLock returns an option for single thread(goroutine)
func WithoutLock() Option {
	return unLockOption{}
}

type slackOption int

func (o slackOption) apply(c *config) {
	c.maxSlack = time.Duration(o)
}

// WithoutSlack is an Option for ratelimit.New that initializes the limiter
// without any initial tolerance for bursts of traffic.
var WithoutSlack Option = slackOption(0)

type perOption time.Duration

func (p perOption) apply(c *config) {
	c.per = time.Duration(p)
}

// Per allows configuring limits for different time windows.
//
// The default window is one second, so New(100) produces a one hundred per
// second (100 Hz) rate limiter.
//
// New(2, Per(60*time.Second)) creates a 2 per minute rate limiter.
func Per(per time.Duration) Option {
	return perOption(per)
}

type unlimited struct{}

// NewUnlimited returns a RateLimiter that is not limited.
func NewUnlimited() Limiter {
	return unlimited{}
}

func (unlimited) Take() time.Time {
	return time.Now()
}
