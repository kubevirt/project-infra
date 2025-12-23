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
 * Copyright the KubeVirt Authors.
 *
 */

package metrics

import (
	"fmt"
	"slices"
	"sync"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
)

type fakeCounter struct {
	CounterValue *atomic.Int32
}

func newFakeCounter() *fakeCounter {
	return &fakeCounter{
		CounterValue: &atomic.Int32{},
	}
}

func (c fakeCounter) Describe(_ chan<- *prometheus.Desc) {
	//not implemented
}

func (c fakeCounter) Collect(_ chan<- prometheus.Metric) {
	//not implemented
}

func (c fakeCounter) Inc() {
	c.CounterValue.Add(1)
}

type fakeGaugeVecWrapper struct {
	LabelValues []string
	Value       float64
}

func newFakeGaugeVecWrapper() *fakeGaugeVecWrapper {
	return &fakeGaugeVecWrapper{}
}

func (gv *fakeGaugeVecWrapper) SetWithLabelValues(value float64, labelValues ...string) {
	gv.LabelValues = labelValues
	gv.Value = value
}

func (gv *fakeGaugeVecWrapper) DeleteLabelValues(labelValues ...string) bool {
	if !slices.Equal(labelValues, gv.LabelValues) {
		panic("not implemented")
	}
	gv.LabelValues = nil
	gv.Value = 0
	return true
}

var _ = Describe("retests", func() {
	Context("counters", func() {
		var counters map[string]counter
		var countersMutex sync.RWMutex
		var gaugeVecs map[string]gaugeVec
		var gaugeVecsMutex sync.RWMutex
		BeforeEach(func() {
			counters = make(map[string]counter)
			countersMutex = sync.RWMutex{}
			createCounter = func(name, help string) counter {
				countersMutex.Lock()
				defer countersMutex.Unlock()
				counters[name] = newFakeCounter()
				return counters[name]
			}
			gaugeVecs = make(map[string]gaugeVec)
			gaugeVecsMutex = sync.RWMutex{}
			createGaugeVec = func(name, help string) gaugeVecWrapper {
				gaugeVecsMutex.Lock()
				defer gaugeVecsMutex.Unlock()
				gaugeVecs[name] = newFakeGaugeVecWrapper()
				return gaugeVecs[name]
			}
			reset()
		})
		It("total counter increased", func() {
			IncForRepository("myorg", "myrepo")
			actual := counters[totalRetestsCounterName].(*fakeCounter)
			Expect(actual.CounterValue.Load()).To(BeEquivalentTo(1))
		})
		It("repo counter created and increased", func() {
			IncForRepository("myorg", "myrepo")
			actual := counters[fmt.Sprintf(retestsPerRepoCounterName, "myorg", "myrepo")].(*fakeCounter)
			Expect(actual).ToNot(BeNil())
			Expect(actual.CounterValue.Load()).To(BeEquivalentTo(1))
		})
		It("pr gauge created and set", func() {
			SetForPullRequest("myorg", "myrepo", 1737, 42)
			actual := gaugeVecs[fmt.Sprintf(retestsPerPRGaugeName, "myorg", "myrepo")].(*fakeGaugeVecWrapper)
			Expect(actual).ToNot(BeNil())
			Expect(actual.LabelValues).To(BeEquivalentTo([]string{"1737"}))
			Expect(actual.Value).To(BeEquivalentTo(float64(42)))
		})
		It("pr gauge unset", func() {
			SetForPullRequest("myorg", "myrepo", 1737, 42)
			DeleteForPullRequest("myorg", "myrepo", 1737)
			actual := gaugeVecs[fmt.Sprintf(retestsPerPRGaugeName, "myorg", "myrepo")].(*fakeGaugeVecWrapper)
			Expect(actual).ToNot(BeNil())
			Expect(actual.LabelValues).To(BeEmpty())
		})
		AfterEach(func() {
			createCounter = defaultCreateCounterFunc
			createGaugeVec = defaultCreateGaugeVecFunc
		})
	})
})
