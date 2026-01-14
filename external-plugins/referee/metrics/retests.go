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
	"strconv"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	promSubsystemForRetests   = "retests"
	totalRetestsCounterName   = "total"
	retestsPerRepoCounterName = "%s_%s_total"
	retestsPerPRGaugeName     = "%s_%s_pr_since_last_commit"
)

type counter interface {
	Inc()
	Describe(chan<- *prometheus.Desc)
	Collect(chan<- prometheus.Metric)
}

type gaugeVec interface {
	SetWithLabelValues(value float64, values ...string)
	DeleteLabelValues(labelValues ...string) bool
}

var (
	totalRetests = createCounter(
		totalRetestsCounterName,
		"The total number of retests encountered so far",
	)
	retestsPerRepo             = map[string]counter{}
	retestsPerRepoMutex        = sync.RWMutex{}
	retestsPerPullRequest      = map[string]gaugeVecWrapper{}
	retestsPerPullRequestMutex = sync.RWMutex{}
	defaultCreateCounterFunc   = func(name, help string) counter {
		return promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "referee",
			Subsystem: "retests",
			Name:      name,
			Help:      help,
		})
	}
	createCounter             = defaultCreateCounterFunc
	defaultCreateGaugeVecFunc = func(name, help string) gaugeVecWrapper {
		return newGaugeVecWrapper(
			promauto.NewGaugeVec(prometheus.GaugeOpts{
				Namespace: promNamespace,
				Subsystem: promSubsystemForRetests,
				Name:      name,
				Help:      help,
			},
				[]string{
					"pull_request",
				},
			),
		)
	}
	createGaugeVec = defaultCreateGaugeVecFunc
)

type gaugeVecWrapper interface {
	SetWithLabelValues(value float64, values ...string)
	DeleteLabelValues(labelValues ...string) bool
}
type simpleGaugeVecWrapper struct {
	gaugeVec *prometheus.GaugeVec
}

func newGaugeVecWrapper(v *prometheus.GaugeVec) gaugeVecWrapper {
	return simpleGaugeVecWrapper{gaugeVec: v}
}

func (w simpleGaugeVecWrapper) SetWithLabelValues(value float64, labelValues ...string) {
	w.gaugeVec.WithLabelValues(labelValues...).Set(value)
}

func (w simpleGaugeVecWrapper) DeleteLabelValues(labelValues ...string) bool {
	return w.gaugeVec.DeleteLabelValues(labelValues...)
}

func reset() {
	prometheus.Unregister(totalRetests)
	totalRetests = createCounter(
		totalRetestsCounterName,
		"The total number of retests encountered so far",
	)
	for _, counter := range retestsPerRepo {
		prometheus.Unregister(counter)
	}
	retestsPerRepo = map[string]counter{}
	for _, wrapper := range retestsPerPullRequest {
		simple, ok := wrapper.(simpleGaugeVecWrapper)
		if ok {
			prometheus.Unregister(simple.gaugeVec)
		}
	}
	retestsPerPullRequest = map[string]gaugeVecWrapper{}
}

// SetForPullRequest sets the number of retests for a pull request since the last commit.
func SetForPullRequest(org, repo string, pr, value int) {
	retestsPerPRKey := fmt.Sprintf(retestsPerPRGaugeName, org, repo)
	retestsPerPullRequestMutex.Lock()
	defer retestsPerPullRequestMutex.Unlock()
	_, found := retestsPerPullRequest[retestsPerPRKey]
	if !found {
		retestsPerPullRequest[retestsPerPRKey] = createGaugeVec(retestsPerPRKey, fmt.Sprintf("The number of retests per PR since last commit in %s/%s encountered so far", org, repo))
	}
	retestsPerPullRequest[retestsPerPRKey].SetWithLabelValues(float64(value), strconv.Itoa(pr))
}

// DeleteForPullRequest removes the number of retests for a pull request since last commit from the metrics.
func DeleteForPullRequest(org, repo string, pr int) {
	retestsPerPRKey := fmt.Sprintf(retestsPerPRGaugeName, org, repo)
	retestsPerPullRequestMutex.Lock()
	defer retestsPerPullRequestMutex.Unlock()
	_, found := retestsPerPullRequest[retestsPerPRKey]
	if !found {
		retestsPerPullRequest[retestsPerPRKey] = createGaugeVec(retestsPerPRKey, fmt.Sprintf("The number of retests per PR since last commit in %s/%s encountered so far", org, repo))
	}
	retestsPerPullRequest[retestsPerPRKey].DeleteLabelValues(strconv.Itoa(pr))
}

// IncForRepository increases the number of retests encountered inside pull requests for a repository.
func IncForRepository(org, repo string) {
	totalRetests.Inc()
	increaseRetestsPerRepoCounter(org, repo)
}

func increaseRetestsPerRepoCounter(org string, repo string) {
	retestsPerRepoKey := fmt.Sprintf(retestsPerRepoCounterName, org, repo)
	retestsPerRepoMutex.Lock()
	defer retestsPerRepoMutex.Unlock()
	_, found := retestsPerRepo[retestsPerRepoKey]
	if !found {
		retestsPerRepo[retestsPerRepoKey] = createCounter(retestsPerRepoKey, fmt.Sprintf("The total number of retests for %s encountered so far", retestsPerRepoKey))
	}
	retestsPerRepo[retestsPerRepoKey].Inc()
}
