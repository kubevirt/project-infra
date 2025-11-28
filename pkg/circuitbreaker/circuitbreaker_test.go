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

package circuitbreaker_test

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"kubevirt.io/project-infra/pkg/circuitbreaker"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCircuitBreaker(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "jenkins suite")
}

var _ = Describe("circuitbreaker.go", func() {

	BeforeEach(func() {
		circuitbreaker.Log().SetOutput(io.Discard)
	})

	alwaysOpenCircuit := func(err error) bool {
		return true
	}

	When("creating", func() {

		It("should panic on non positive retryAfter", func() {
			Expect(func() {
				circuitbreaker.NewCircuitBreaker(0, alwaysOpenCircuit)
			}).To(Panic())
		})

		It("should panic on missing func", func() {
			Expect(func() { circuitbreaker.NewCircuitBreaker(time.Millisecond, nil) }).To(Panic())
		})

		It("should return a breaker if the retry interval is positive", func() {
			Expect(circuitbreaker.NewCircuitBreaker(time.Millisecond, alwaysOpenCircuit)).To(Not(BeNil()))
		})

	})

	When("calling with one instance", func() {

		var circuitBreaker *circuitbreaker.CircuitBreaker
		retryAfter := 10 * time.Millisecond
		everythingOpensExceptTheAnswer := func(err error) bool {
			return !strings.Contains(err.Error(), "42")
		}

		BeforeEach(func() {
			circuitBreaker = circuitbreaker.NewCircuitBreaker(retryAfter, everythingOpensExceptTheAnswer)
		})

		It("should call the target", func() {
			mockRetryableFunc := &MockRetryableFunc{errors: []error{nil}}
			retryableFunc := circuitBreaker.WrapRetryableFunc(mockRetryableFunc.target())
			retryableFunc()
			Expect(mockRetryableFunc.calls).To(BeEquivalentTo(1))
		})

		It("should not call the target a second time before the retry period has passed", func() {
			mockRetryableFunc := &MockRetryableFunc{errors: []error{fmt.Errorf("test"), nil}}
			retryableFunc := circuitBreaker.WrapRetryableFunc(mockRetryableFunc.target())
			retryableFunc()
			retryableFunc()
			Expect(mockRetryableFunc.calls).To(BeEquivalentTo(1))
		})

		It("should call the target a second time if the error should not open the circuit", func() {
			mockRetryableFunc := &MockRetryableFunc{errors: []error{fmt.Errorf("42"), nil}}
			retryableFunc := circuitBreaker.WrapRetryableFunc(mockRetryableFunc.target())
			retryableFunc()
			retryableFunc()
			Expect(mockRetryableFunc.calls).To(BeEquivalentTo(2))
		})

		It("should call the target a second time after retry period has passed", func() {
			mockRetryableFunc := &MockRetryableFunc{errors: []error{fmt.Errorf("test"), nil}}
			retryableFunc := circuitBreaker.WrapRetryableFunc(mockRetryableFunc.target())
			retryableFunc()
			time.Sleep(retryAfter)
			retryableFunc()
			Expect(mockRetryableFunc.calls).To(BeEquivalentTo(2))
		})

	})

	When("calling with two instances", func() {

		var circuitBreaker *circuitbreaker.CircuitBreaker
		retryAfter := 10 * time.Millisecond

		BeforeEach(func() {
			circuitBreaker = circuitbreaker.NewCircuitBreaker(retryAfter, alwaysOpenCircuit)
		})

		It("should not call the second target if the call to first target failed", func() {
			targetFuncA := &MockRetryableFunc{errors: []error{fmt.Errorf("test")}}
			targetFuncB := &MockRetryableFunc{errors: []error{nil}}
			retryableFuncA := circuitBreaker.WrapRetryableFunc(targetFuncA.target())
			retryableFuncB := circuitBreaker.WrapRetryableFunc(targetFuncB.target())
			retryableFuncA()
			retryableFuncB()
			Expect(targetFuncA.calls).To(BeEquivalentTo(1))
			Expect(targetFuncB.calls).To(BeEquivalentTo(0))
		})

		It("should call the second target after retry period has passed", func() {
			targetFuncA := &MockRetryableFunc{errors: []error{fmt.Errorf("test")}}
			targetFuncB := &MockRetryableFunc{errors: []error{nil}}
			retryableFuncA := circuitBreaker.WrapRetryableFunc(targetFuncA.target())
			retryableFuncB := circuitBreaker.WrapRetryableFunc(targetFuncB.target())
			retryableFuncA()
			time.Sleep(retryAfter)
			retryableFuncB()
			Expect(targetFuncA.calls).To(BeEquivalentTo(1))
			Expect(targetFuncB.calls).To(BeEquivalentTo(1))
		})

	})

	When("calling with go routines", func() {

		const numberOfThreads = 4

		var circuitBreaker *circuitbreaker.CircuitBreaker
		retryAfter := 10 * time.Millisecond

		BeforeEach(func() {
			circuitBreaker = circuitbreaker.NewCircuitBreaker(retryAfter, alwaysOpenCircuit)
		})

		It("should call only one target func in case of error if target func is called before others", func() {
			var wg sync.WaitGroup
			wg.Add(numberOfThreads - 1)
			var targetFuncs []*MockRetryableFunc
			for i := 0; i < numberOfThreads; i++ {
				targetFunc := &MockRetryableFunc{errors: []error{fmt.Errorf("test")}}
				retryableFunc := circuitBreaker.WrapRetryableFunc(targetFunc.target())
				if i > 0 {
					go func(retryableFunc func() error, index int) {
						defer wg.Done()
						_ = retryableFunc()
					}(retryableFunc, i)
				} else {
					_ = retryableFunc()
				}
				targetFuncs = append(targetFuncs, targetFunc)
			}
			wg.Wait()
			totalCalls := 0
			for i := 0; i < numberOfThreads; i++ {
				totalCalls += targetFuncs[i].calls
			}
			Expect(totalCalls).To(BeEquivalentTo(1))
		})

		It("may call more than one target func in case of error if target funcs are called asynchronously", func() {
			var wg sync.WaitGroup
			wg.Add(numberOfThreads)
			var targetFuncs []*MockRetryableFunc
			for i := 0; i < numberOfThreads; i++ {
				targetFunc := &MockRetryableFunc{errors: []error{fmt.Errorf("test")}}
				retryableFunc := circuitBreaker.WrapRetryableFunc(targetFunc.target())
				go func(retryableFunc func() error, index int) {
					defer wg.Done()
					_ = retryableFunc()
				}(retryableFunc, i)
				targetFuncs = append(targetFuncs, targetFunc)
			}
			wg.Wait()
			totalCalls := 0
			for i := 0; i < numberOfThreads; i++ {
				totalCalls += targetFuncs[i].calls
			}
			Expect(totalCalls).To(BeNumerically(">=", 1))
		})

		It("should call all four target funcs in case of no error", func() {
			var wg sync.WaitGroup
			wg.Add(numberOfThreads)
			var targetFuncs []*MockRetryableFunc
			for i := 0; i < numberOfThreads; i++ {
				targetFunc := &MockRetryableFunc{errors: []error{nil}}
				targetFuncs = append(targetFuncs, targetFunc)
				go func(index int, targetFunc *MockRetryableFunc) {
					defer wg.Done()
					_ = circuitBreaker.WrapRetryableFunc(targetFunc.target())()
				}(i, targetFunc)
			}
			wg.Wait()
			totalCalls := 0
			for i := 0; i < numberOfThreads; i++ {
				totalCalls += targetFuncs[i].calls
			}
			Expect(totalCalls).To(BeEquivalentTo(4))
		})

		It("should call all four target funcs in case of the first caller errors and the others wait long enough", func() {
			var wg sync.WaitGroup
			wg.Add(numberOfThreads)

			// while the first call of target func happens directly and returns an error...
			var targetFuncs []*MockRetryableFunc
			targetFunc := &MockRetryableFunc{errors: []error{fmt.Errorf("test")}}
			targetFuncs = append(targetFuncs, targetFunc)
			go func(targetFunc *MockRetryableFunc) {
				defer wg.Done()
				_ = circuitBreaker.WrapRetryableFunc(targetFunc.target())()
			}(targetFunc)

			// ...the remaining functions all wait twice the retryAfter period before they call the target func
			for i := 1; i < numberOfThreads; i++ {
				targetFunc := &MockRetryableFunc{errors: []error{nil}}
				targetFuncs = append(targetFuncs, targetFunc)
				go func(index int, targetFunc *MockRetryableFunc) {
					defer wg.Done()
					time.Sleep(2 * retryAfter)
					_ = circuitBreaker.WrapRetryableFunc(targetFunc.target())()
				}(i, targetFunc)
			}

			wg.Wait()

			// this should result in all functions being called
			totalCalls := 0
			for i := 0; i < numberOfThreads; i++ {
				totalCalls += targetFuncs[i].calls
			}
			Expect(totalCalls).To(BeEquivalentTo(4))
		})

	})

})

// MockRetryableFunc is a mock that returns a func implementing RetryableFunc, which can return different error values
// and count the number of times the func got called.
type MockRetryableFunc struct {
	calls  int
	errors []error
}

// target returns a RetryableFunc that for each consecutive call returns the next element
// from the slice, thereafter it increments the counter indicating the number of times the target func got called by one.
func (m *MockRetryableFunc) target() func() error {
	return func() error {
		err := m.errors[m.calls]
		m.calls++
		return err
	}
}
