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
)

// CircuitBreaker is a minimal implementation of the [circuit breaker pattern].
//
// [circuit breaker pattern]: https://martinfowler.com/bliki/CircuitBreaker.html
type CircuitBreaker struct {
	mutex      sync.Mutex
	lastErr    error
	open       bool
	occurred   time.Time
	retryAfter time.Duration
}

func NewCircuitBreaker(retryAfter time.Duration) *CircuitBreaker {
	if retryAfter <= 0 {
		panic(fmt.Errorf("retryAfter <= 0: %v", retryAfter))
	}
	return &CircuitBreaker{retryAfter: retryAfter}
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
			return lastErr
		}
		err := retryableFunc()
		g.updateState(err)
		return err
	}
}

func (g *CircuitBreaker) isOpenAndNotFeasibleForRetry() (bool, error) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	return g.open && !g.occurred.Add(g.retryAfter).Before(time.Now()), g.lastErr
}

func (g *CircuitBreaker) updateState(err error) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	if err != nil {
		g.open = true
		g.occurred = time.Now()
	} else {
		g.open = false
	}
	g.lastErr = err
}
