// Package retry ...
// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package retry

import (
	"fmt"
	"time"
)

// RetriableFunc is a function to be retried in order to get a successful
// execution. In general this are functions which success depend on external
// conditions that can eventually be met.
type RetriableFunc func() error

// OnRetryFunc is executed after a RetrieableFunc fails and receives the
// returned error as argument.
type OnRetryFunc func(err error)

type config struct {
	delay   time.Duration
	timeout time.Duration
	onRetry OnRetryFunc
}

// Option to be applied to the retry config.
type Option func(*config)

// Delay is the time to wait after a failed execution before retrying.
func Delay(delay time.Duration) Option {
	return func(c *config) {
		c.delay = delay
	}
}

// Timeout sets the time duration to wait before aborting the retries if there
// are not successful executions of the function.
func Timeout(timeout time.Duration) Option {
	return func(c *config) {
		c.timeout = timeout
	}
}

// OnRetry sets a new function to be applied to the error returned by the
// function execution.
func OnRetry(fn OnRetryFunc) Option {
	return func(c *config) {
		c.onRetry = fn
	}
}

// Do retries the execution of the given function until it finishes
// successfully, it returns a nil error, or a timeout is reached.
//
// If the function returns a non-nil error, a delay will be applied before
// executing the onRetry function on the error and retrying the function.
func Do(fn RetriableFunc, opts ...Option) error {
	var nRetries int
	c := &config{
		delay:   2 * time.Second,
		timeout: 2 * time.Minute,
		onRetry: func(err error) {},
	}
	for _, opt := range opts {
		opt(c)
	}
	tRetry := time.NewTicker(c.delay)
	tTimeout := time.NewTicker(c.timeout)
	for {
		lastError := fn()
		if lastError == nil {
			return nil
		}

		select {
		case <-tTimeout.C:
			tRetry.Stop()
			tTimeout.Stop()
			return fmt.Errorf("timeout reached, %d retries executed. last error: %s", nRetries, lastError)
		case <-tRetry.C:
			c.onRetry(lastError)
			nRetries++
		}
	}
}
