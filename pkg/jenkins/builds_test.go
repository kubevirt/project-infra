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

package jenkins

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bndr/gojenkins"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"kubevirt.io/project-infra/pkg/circuitbreaker"
)

func TestJenkins(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "jenkins suite")
}

type SimpleMockBuildDataGetter struct {
	callCounter int
	build       []*gojenkins.Build
	err         []error
}

func (d *SimpleMockBuildDataGetter) GetBuild(int64) (build *gojenkins.Build, err error) {
	build, err = d.build[d.callCounter], d.err[d.callCounter]
	d.callCounter++
	return build, err
}

type DurationBasedMockBuildDataGetter struct {
	callCounter   uint32
	start         time.Time
	durationIndex []time.Duration
	build         []*gojenkins.Build
	err           []error
}

func (d *DurationBasedMockBuildDataGetter) GetBuild(int64) (build *gojenkins.Build, err error) {
	atomic.AddUint32(&d.callCounter, 1)
	for index, durationInterval := range d.durationIndex {
		if time.Now().Before(d.start.Add(durationInterval)) {
			return d.build[index], d.err[index]
		}
	}
	panic(fmt.Errorf("no interval was matching!"))
}

func (d *DurationBasedMockBuildDataGetter) GetCallCounter() uint32 {
	return d.callCounter
}

var _ = Describe("builds.go", func() {

	BeforeEach(func() {
		retryDelay = 150 * time.Millisecond
		maxJitter = 10 * time.Millisecond
		circuitBreakerBuildDataGetter = circuitbreaker.NewCircuitBreaker(retryDelay, openOnStatusGateWayTimeout)
	})

	When("checking circuitbreaker function", func() {

		It("should open on status gateway timeout", func() {
			Expect(openOnStatusGateWayTimeout(fmt.Errorf("%d", http.StatusGatewayTimeout))).To(BeTrue())
		})

		It("should not open on status not found", func() {
			Expect(openOnStatusGateWayTimeout(fmt.Errorf("%d", http.StatusNotFound))).To(BeFalse())
		})

		It("should not open on non status errors", func() {
			Expect(openOnStatusGateWayTimeout(fmt.Errorf("whatever else may happen"))).To(BeFalse())
		})
	})

	When("retrying", func() {

		entry := logrus.WithField("dummy", "blah")

		It("should return build directly", func() {
			expectedBuild := &gojenkins.Build{}
			build, statusCode, err := getBuildFromGetterWithRetry(&SimpleMockBuildDataGetter{build: []*gojenkins.Build{expectedBuild}, err: []error{nil}}, int64(42), entry)
			Expect(build).To(BeIdenticalTo(expectedBuild))
			Expect(statusCode).To(BeEquivalentTo(0))
			Expect(err).To(BeNil())
		})

		It("should return nil if 404", func() {
			err2 := fmt.Errorf("%d", http.StatusNotFound)
			build, statusCode, err := getBuildFromGetterWithRetry(&SimpleMockBuildDataGetter{build: []*gojenkins.Build{nil}, err: []error{err2}}, int64(42), entry)
			Expect(build).To(BeNil())
			Expect(statusCode).To(BeEquivalentTo(http.StatusNotFound))
			Expect(err).To(BeIdenticalTo(err2))
		})

		It("should return nil if 403", func() {
			err2 := fmt.Errorf("%d", http.StatusForbidden)
			build, statusCode, err := getBuildFromGetterWithRetry(&SimpleMockBuildDataGetter{build: []*gojenkins.Build{nil}, err: []error{err2}}, int64(42), entry)
			Expect(build).To(BeNil())
			Expect(statusCode).To(BeEquivalentTo(http.StatusForbidden))
			Expect(err).To(BeIdenticalTo(err2))
		})

		It("should return build after one retry with gateway timeout", func() {
			expectedBuild := &gojenkins.Build{}
			build, statusCode, err := getBuildFromGetterWithRetry(&SimpleMockBuildDataGetter{build: []*gojenkins.Build{nil, expectedBuild}, err: []error{fmt.Errorf("%d", http.StatusGatewayTimeout), nil}}, int64(42), entry)
			Expect(build).To(BeIdenticalTo(expectedBuild))
			Expect(statusCode).To(BeEquivalentTo(http.StatusGatewayTimeout))
			Expect(err).To(BeNil())
		})

		It("should not call any of the getters more than twice", func() {
			numberOfThreads := 5
			var wg sync.WaitGroup
			wg.Add(numberOfThreads)
			var buildDataGetters []*DurationBasedMockBuildDataGetter
			for i := 0; i < numberOfThreads; i++ {
				buildDataGetter := &DurationBasedMockBuildDataGetter{start: time.Now(), durationIndex: []time.Duration{100 * time.Millisecond, 1000 * time.Millisecond}, build: []*gojenkins.Build{nil, {}}, err: []error{fmt.Errorf("%d", http.StatusGatewayTimeout), nil}}
				go func() {
					defer wg.Done()
					_, _, _ = getBuildFromGetterWithRetry(buildDataGetter, int64(42), entry)
				}()
				buildDataGetters = append(buildDataGetters, buildDataGetter)
			}
			wg.Wait()
			for _, b := range buildDataGetters {
				Expect(b.GetCallCounter()).To(BeNumerically("<=", 2))
			}
		})

		It("should call the service only once per thread in case of 503, since the circuit only opens on 504, and 503 is not valid for retry", func() {
			buildDataGetter := &DurationBasedMockBuildDataGetter{start: time.Now(), durationIndex: []time.Duration{100 * time.Millisecond, 1000 * time.Millisecond}, build: []*gojenkins.Build{nil, {}}, err: []error{fmt.Errorf("%d", http.StatusServiceUnavailable), nil}}
			var wg sync.WaitGroup
			numberOfThreads := 5
			wg.Add(numberOfThreads)
			for i := 0; i < numberOfThreads; i++ {
				go func() {
					defer wg.Done()
					_, _, _ = getBuildFromGetterWithRetry(buildDataGetter, int64(42), entry)
				}()
			}
			wg.Wait()
			Expect(buildDataGetter.GetCallCounter()).To(BeEquivalentTo(uint32(numberOfThreads)))
		})

	})

	When("filtering builds", func() {

		It("should treat 404 as missing build status", func() {
			Expect(isMissingBuildStatus(http.StatusNotFound)).To(BeTrue())
		})

		It("should treat 403 as missing build status", func() {
			// This test validates that getFilteredBuild will treat 403 the same as 404,
			// returning (nil, false) instead of calling Fatalf. The actual getFilteredBuild
			// function uses isMissingBuildStatus internally, so testing this helper function
			// validates the behavior.
			Expect(isMissingBuildStatus(http.StatusForbidden)).To(BeTrue())
		})

		It("should not treat other status codes as missing builds", func() {
			Expect(isMissingBuildStatus(http.StatusOK)).To(BeFalse())
			Expect(isMissingBuildStatus(http.StatusInternalServerError)).To(BeFalse())
			Expect(isMissingBuildStatus(http.StatusGatewayTimeout)).To(BeFalse())
			Expect(isMissingBuildStatus(http.StatusUnauthorized)).To(BeFalse())
		})
	})

})
