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

package get

import (
	"math/rand"
	"sort"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/prow/pkg/config"
)

func TestPresubmits(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "presubmits suite")
}

var _ = Describe("presubmits", func() {
	When("isGating", func() {
		DescribeTable("isGating",
			func(a config.Presubmit, expected bool) {
				Expect(isGating(a)).To(BeEquivalentTo(expected))
			},
			Entry("always_run", config.Presubmit{AlwaysRun: true}, true),
			Entry("run_before_merge", config.Presubmit{RunBeforeMerge: true}, true),
			Entry("always_run optional", config.Presubmit{AlwaysRun: true, Optional: true}, false),
			Entry("run_before_merge optional", config.Presubmit{RunBeforeMerge: true, Optional: true}, false),
		)
	})
	When("sorting", func() {

		DescribeTable("Less",
			func(a, b config.Presubmit, expected bool) {
				Expect(presubmits.Less(presubmits{a, b}, 0, 1)).To(BeEquivalentTo(expected))
			},
			Entry("always_run < !always_run",
				config.Presubmit{AlwaysRun: true},
				config.Presubmit{},
				true,
			),
			Entry("always_run < optional always_run",
				config.Presubmit{AlwaysRun: true},
				config.Presubmit{AlwaysRun: true, Optional: true},
				true,
			),
			Entry("always_run < run_before_merge",
				config.Presubmit{AlwaysRun: true},
				config.Presubmit{RunBeforeMerge: true},
				true,
			),
			Entry("run_before_merge < optional always_run",
				config.Presubmit{RunBeforeMerge: true},
				config.Presubmit{AlwaysRun: false, Optional: true},
				true,
			),
			Entry("run_before_merge < optional run_before_merge",
				config.Presubmit{RunBeforeMerge: true},
				config.Presubmit{RunBeforeMerge: true, Optional: true},
				true,
			),
			Entry("abc < bac",
				config.Presubmit{JobBase: config.JobBase{Name: "abc"}},
				config.Presubmit{JobBase: config.JobBase{Name: "bac"}},
				true,
			),
			Entry("bac >= abc",
				config.Presubmit{JobBase: config.JobBase{Name: "bac"}},
				config.Presubmit{JobBase: config.JobBase{Name: "abc"}},
				false,
			),
		)

		DescribeTable("sorts entries as expected",
			func(expected presubmits) {
				for k := 0; k < 10; k++ {
					toSort := shuffle(expected)
					sort.Sort(toSort)
					actualNames := names(toSort)
					Expect(actualNames).To(BeEquivalentTo(names(expected)))
				}
			},
			Entry("small", presubmits{
				newPresubmit(withName("a-ra___"), alwaysRun()),
				newPresubmit(withName("a-o____"), runBeforeMerge(), optional()),
			}),
			Entry("medium", presubmits{
				newPresubmit(withName("a-ra___"), alwaysRun()),
				newPresubmit(withName("a-r_b__"), runBeforeMerge()),
				newPresubmit(withName("a-o___s"), skipIfOnlyChanged("37"), optional()),
				newPresubmit(withName("a-o____"), optional()),
			}),
			Entry("big", presubmits{
				newPresubmit(withName("a-ra___"), alwaysRun()),
				newPresubmit(withName("a-r_b__"), runBeforeMerge()),
				newPresubmit(withName("a-oa___"), alwaysRun(), optional()),
				newPresubmit(withName("a-o_b__"), runBeforeMerge(), optional()),
				newPresubmit(withName("a-o__r_"), runIfChanged("42"), optional()),
				newPresubmit(withName("a-o___s"), skipIfOnlyChanged("37"), optional()),
				newPresubmit(withName("a-o____"), optional()),
			}),
			Entry("lex big", presubmits{
				newPresubmit(withName("a-ra___"), alwaysRun()),
				newPresubmit(withName("b-ra___"), alwaysRun()),
				newPresubmit(withName("a-r_b__"), runBeforeMerge()),
				newPresubmit(withName("b-r_b__"), runBeforeMerge()),
				newPresubmit(withName("a-oa___"), alwaysRun(), optional()),
				newPresubmit(withName("b-oa___"), alwaysRun(), optional()),
				newPresubmit(withName("a-o_b__"), runBeforeMerge(), optional()),
				newPresubmit(withName("b-o_b__"), runBeforeMerge(), optional()),
				newPresubmit(withName("a-o__r_"), runIfChanged("42"), optional()),
				newPresubmit(withName("b-o__r_"), runIfChanged("42"), optional()),
				newPresubmit(withName("a-o___s"), skipIfOnlyChanged("37"), optional()),
				newPresubmit(withName("b-o___s"), skipIfOnlyChanged("37"), optional()),
				newPresubmit(withName("a-o____"), optional()),
				newPresubmit(withName("b-o____"), optional()),
			}),
			Entry("k8sVersions in different groups", presubmits{
				newPresubmit(withName("pull-kubevirt-e2e-k8s-1.33-sig-compute"), alwaysRun()),
				newPresubmit(withName("pull-kubevirt-e2e-k8s-1.33-sig-storage"), alwaysRun()),
				newPresubmit(withName("pull-kubevirt-e2e-k8s-1.32-sig-compute"), runBeforeMerge()),
				newPresubmit(withName("pull-kubevirt-e2e-k8s-1.32-sig-storage"), runBeforeMerge()),
			}),
			Entry("k8sVersions in same group (always_run)", presubmits{
				newPresubmit(withName("pull-kubevirt-e2e-k8s-1.33-sig-compute"), alwaysRun()),
				newPresubmit(withName("pull-kubevirt-e2e-k8s-1.33-sig-compute-serial"), alwaysRun()),
				newPresubmit(withName("pull-kubevirt-e2e-k8s-1.33-sig-network"), alwaysRun()),
				newPresubmit(withName("pull-kubevirt-e2e-k8s-1.32-sig-compute-migrations"), alwaysRun()),
			}),
		)
	})
})

// shuffle returns a slice with all elements from the original in random order
func shuffle(source presubmits) presubmits {
	var result = make(presubmits, len(source))
	copy(result, source)
	list := rand.Perm(len(result))
	for i := range result {
		j := list[i]
		result[j], result[i] = result[i], result[j]
	}
	return result
}

func names(p presubmits) []string {
	var result []string
	for _, presubmit := range p {
		result = append(result, presubmit.Name)
	}
	return result
}

type presubmitConfig func(p *config.Presubmit)

func withName(name string) presubmitConfig {
	return func(p *config.Presubmit) {
		p.Name = name
	}
}

func alwaysRun() presubmitConfig {
	return func(p *config.Presubmit) {
		p.AlwaysRun = true
	}
}

func runBeforeMerge() presubmitConfig {
	return func(p *config.Presubmit) {
		p.RunBeforeMerge = true
	}
}

func optional() presubmitConfig {
	return func(p *config.Presubmit) {
		p.Optional = true
	}
}

func skipIfOnlyChanged(regex string) presubmitConfig {
	return func(p *config.Presubmit) {
		p.SkipIfOnlyChanged = regex
	}
}

func runIfChanged(regex string) presubmitConfig {
	return func(p *config.Presubmit) {
		p.RunIfChanged = regex
	}
}

func newPresubmit(configs ...presubmitConfig) config.Presubmit {
	result := &config.Presubmit{JobBase: config.JobBase{Name: ""}, AlwaysRun: false, RunBeforeMerge: false, RegexpChangeMatcher: config.RegexpChangeMatcher{RunIfChanged: "", SkipIfOnlyChanged: ""}, Optional: false}
	for _, c := range configs {
		c(result)
	}
	return *result
}
