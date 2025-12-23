/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2022 Red Hat, Inc.
 *
 */

package circuitbreaker

import (
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// CircuitBreaker is a minimal implementation of the [circuit breaker pattern].
//
// [circuit breaker pattern]: https://martinfowler.com/bliki/CircuitBreaker.html
type CircuitBreaker struct {
	mutex        sync.RWMutex
	lastErr      error
	open         bool
	blockedUntil time.Time
	retryAfter   time.Duration
	shouldOpen   func(err error) bool
}

func NewCircuitBreaker(retryAfter time.Duration, shouldOpen func(err error) bool) *CircuitBreaker {
	if retryAfter <= 0 {
		panic(fmt.Errorf("retryAfter <= 0: %v", retryAfter))
	}
	if shouldOpen == nil {
		panic(fmt.Errorf("shouldOpen is nil"))
	}
	return &CircuitBreaker{retryAfter: retryAfter, shouldOpen: shouldOpen}
}

var log *logrus.Entry

func init() {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.DebugLevel)
	log = logger.WithField("component", "circuitbreaker")
}

func Log() *logrus.Logger {
	return log.Logger
}

// WrapRetryableFunc wraps the target retry.RetryableFunc into a new function that transforms the result of the original
// call into the state for the circuit breaker, which is then updated accordingly.
//
// If the circuit breaker is closed, the wrapped function will be called. If the wrapped function returns an error,
// the time of the occurrence and the error will be recorded, and the circuit breaker will be opened.
// If the circuit breaker is open, the wrapped function will only be called if the retryAfter period has been reached,
// otherwise the last occurred error will be returned directly. If the wrapped function is called and does not return
// an error, the circuit breaker will be closed.
func (g *CircuitBreaker) WrapRetryableFunc(retryableFunc func() error) func() error {
	return func() error {
		if shouldStayOpen, lastErr := g.isOpenAndNotFeasibleForRetry(); shouldStayOpen {
			log.WithField("lastErr", lastErr).Debugf("should stay open, returning last error")
			return lastErr
		}
		log.Debugf("calling retryableFunc")
		err := retryableFunc()
		log.Debugf("updating state")
		g.updateState(err)
		return err
	}
}

func (g *CircuitBreaker) isOpenAndNotFeasibleForRetry() (bool, error) {
	g.mutex.RLock()
	defer g.mutex.RUnlock()
	return g.open && g.isRetryBlocked(), g.lastErr
}

func (g *CircuitBreaker) updateState(err error) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	if g.isRetryBlocked() {
		log.WithField("g.blockedUntil", g.blockedUntil).WithField("g.lastErr", g.lastErr).Info("retry is blocked")
		return
	}
	if err != nil && g.shouldOpen(err) {
		g.open = true
		g.blockedUntil = time.Now().Add(g.retryAfter)
		log.Info("circuit breaker is open")
	} else {
		g.open = false
		log.Info("circuit breaker is closed")
	}
	g.lastErr = err
	log.WithField("g.open", g.open).WithField("g.blockedUntil", g.blockedUntil).WithField("g.lastErr", g.lastErr).Debugf("state updated")
}

func (g *CircuitBreaker) isRetryBlocked() bool {
	return !g.blockedUntil.IsZero() && !time.Now().After(g.blockedUntil)
}
