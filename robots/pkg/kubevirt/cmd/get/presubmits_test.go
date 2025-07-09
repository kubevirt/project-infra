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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"math/rand"
	"sigs.k8s.io/prow/pkg/config"
	"sort"
	"testing"
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
				toSort := make(presubmits, len(expected))
				copy(toSort, expected)

				for k := 0; k < 10; k++ {
					// randomize order
					list := rand.Perm(len(toSort))
					for i := range toSort {
						j := list[i]
						toSort[j], toSort[i] = toSort[i], toSort[j]
					}

					expectedNames := names(expected)
					sort.Sort(toSort)
					actualNames := names(toSort)
					Expect(actualNames).To(BeEquivalentTo(expectedNames))
				}
			},
			Entry("small", presubmits{
				config.Presubmit{JobBase: config.JobBase{Name: "a-ra___"}, AlwaysRun: true, RunBeforeMerge: false, RegexpChangeMatcher: config.RegexpChangeMatcher{RunIfChanged: "", SkipIfOnlyChanged: ""}, Optional: false},
				config.Presubmit{JobBase: config.JobBase{Name: "a-o____"}, AlwaysRun: false, RunBeforeMerge: true, RegexpChangeMatcher: config.RegexpChangeMatcher{RunIfChanged: "", SkipIfOnlyChanged: ""}, Optional: true},
			}),
			Entry("medium", presubmits{
				config.Presubmit{JobBase: config.JobBase{Name: "a-ra___"}, AlwaysRun: true, RunBeforeMerge: false, RegexpChangeMatcher: config.RegexpChangeMatcher{RunIfChanged: "", SkipIfOnlyChanged: ""}, Optional: false},
				config.Presubmit{JobBase: config.JobBase{Name: "a-r_b__"}, AlwaysRun: false, RunBeforeMerge: true, RegexpChangeMatcher: config.RegexpChangeMatcher{RunIfChanged: "", SkipIfOnlyChanged: ""}, Optional: false},
				config.Presubmit{JobBase: config.JobBase{Name: "a-o___s"}, AlwaysRun: false, RunBeforeMerge: false, RegexpChangeMatcher: config.RegexpChangeMatcher{RunIfChanged: "", SkipIfOnlyChanged: "37"}, Optional: true},
				config.Presubmit{JobBase: config.JobBase{Name: "a-o____"}, AlwaysRun: false, RunBeforeMerge: false, RegexpChangeMatcher: config.RegexpChangeMatcher{RunIfChanged: "", SkipIfOnlyChanged: ""}, Optional: true},
			}),
			Entry("big", presubmits{
				config.Presubmit{JobBase: config.JobBase{Name: "a-ra___"}, AlwaysRun: true, RunBeforeMerge: false, RegexpChangeMatcher: config.RegexpChangeMatcher{RunIfChanged: "", SkipIfOnlyChanged: ""}, Optional: false},
				config.Presubmit{JobBase: config.JobBase{Name: "a-r_b__"}, AlwaysRun: false, RunBeforeMerge: true, RegexpChangeMatcher: config.RegexpChangeMatcher{RunIfChanged: "", SkipIfOnlyChanged: ""}, Optional: false},
				config.Presubmit{JobBase: config.JobBase{Name: "a-oa___"}, AlwaysRun: true, RunBeforeMerge: false, RegexpChangeMatcher: config.RegexpChangeMatcher{RunIfChanged: "", SkipIfOnlyChanged: ""}, Optional: true},
				config.Presubmit{JobBase: config.JobBase{Name: "a-o_b__"}, AlwaysRun: false, RunBeforeMerge: true, RegexpChangeMatcher: config.RegexpChangeMatcher{RunIfChanged: "", SkipIfOnlyChanged: ""}, Optional: true},
				config.Presubmit{JobBase: config.JobBase{Name: "a-o__r_"}, AlwaysRun: false, RunBeforeMerge: false, RegexpChangeMatcher: config.RegexpChangeMatcher{RunIfChanged: "42", SkipIfOnlyChanged: ""}, Optional: true},
				config.Presubmit{JobBase: config.JobBase{Name: "a-o___s"}, AlwaysRun: false, RunBeforeMerge: false, RegexpChangeMatcher: config.RegexpChangeMatcher{RunIfChanged: "", SkipIfOnlyChanged: "37"}, Optional: true},
				config.Presubmit{JobBase: config.JobBase{Name: "a-o____"}, AlwaysRun: false, RunBeforeMerge: false, RegexpChangeMatcher: config.RegexpChangeMatcher{RunIfChanged: "", SkipIfOnlyChanged: ""}, Optional: true},
			}),
			Entry("lex big", presubmits{
				config.Presubmit{JobBase: config.JobBase{Name: "a-ra___"}, AlwaysRun: true, RunBeforeMerge: false, RegexpChangeMatcher: config.RegexpChangeMatcher{RunIfChanged: "", SkipIfOnlyChanged: ""}, Optional: false},
				config.Presubmit{JobBase: config.JobBase{Name: "b-ra___"}, AlwaysRun: true, RunBeforeMerge: false, RegexpChangeMatcher: config.RegexpChangeMatcher{RunIfChanged: "", SkipIfOnlyChanged: ""}, Optional: false},
				config.Presubmit{JobBase: config.JobBase{Name: "a-r_b__"}, AlwaysRun: false, RunBeforeMerge: true, RegexpChangeMatcher: config.RegexpChangeMatcher{RunIfChanged: "", SkipIfOnlyChanged: ""}, Optional: false},
				config.Presubmit{JobBase: config.JobBase{Name: "b-r_b__"}, AlwaysRun: false, RunBeforeMerge: true, RegexpChangeMatcher: config.RegexpChangeMatcher{RunIfChanged: "", SkipIfOnlyChanged: ""}, Optional: false},
				config.Presubmit{JobBase: config.JobBase{Name: "a-oa___"}, AlwaysRun: true, RunBeforeMerge: false, RegexpChangeMatcher: config.RegexpChangeMatcher{RunIfChanged: "", SkipIfOnlyChanged: ""}, Optional: true},
				config.Presubmit{JobBase: config.JobBase{Name: "b-oa___"}, AlwaysRun: true, RunBeforeMerge: false, RegexpChangeMatcher: config.RegexpChangeMatcher{RunIfChanged: "", SkipIfOnlyChanged: ""}, Optional: true},
				config.Presubmit{JobBase: config.JobBase{Name: "a-o_b__"}, AlwaysRun: false, RunBeforeMerge: true, RegexpChangeMatcher: config.RegexpChangeMatcher{RunIfChanged: "", SkipIfOnlyChanged: ""}, Optional: true},
				config.Presubmit{JobBase: config.JobBase{Name: "b-o_b__"}, AlwaysRun: false, RunBeforeMerge: true, RegexpChangeMatcher: config.RegexpChangeMatcher{RunIfChanged: "", SkipIfOnlyChanged: ""}, Optional: true},
				config.Presubmit{JobBase: config.JobBase{Name: "a-o__r_"}, AlwaysRun: false, RunBeforeMerge: false, RegexpChangeMatcher: config.RegexpChangeMatcher{RunIfChanged: "42", SkipIfOnlyChanged: ""}, Optional: true},
				config.Presubmit{JobBase: config.JobBase{Name: "b-o__r_"}, AlwaysRun: false, RunBeforeMerge: false, RegexpChangeMatcher: config.RegexpChangeMatcher{RunIfChanged: "42", SkipIfOnlyChanged: ""}, Optional: true},
				config.Presubmit{JobBase: config.JobBase{Name: "a-o___s"}, AlwaysRun: false, RunBeforeMerge: false, RegexpChangeMatcher: config.RegexpChangeMatcher{RunIfChanged: "", SkipIfOnlyChanged: "37"}, Optional: true},
				config.Presubmit{JobBase: config.JobBase{Name: "b-o___s"}, AlwaysRun: false, RunBeforeMerge: false, RegexpChangeMatcher: config.RegexpChangeMatcher{RunIfChanged: "", SkipIfOnlyChanged: "37"}, Optional: true},
				config.Presubmit{JobBase: config.JobBase{Name: "a-o____"}, AlwaysRun: false, RunBeforeMerge: false, RegexpChangeMatcher: config.RegexpChangeMatcher{RunIfChanged: "", SkipIfOnlyChanged: ""}, Optional: true},
				config.Presubmit{JobBase: config.JobBase{Name: "b-o____"}, AlwaysRun: false, RunBeforeMerge: false, RegexpChangeMatcher: config.RegexpChangeMatcher{RunIfChanged: "", SkipIfOnlyChanged: ""}, Optional: true},
			}),
			Entry("k8sVersions", presubmits{
				config.Presubmit{JobBase: config.JobBase{Name: "pull-kubevirt-e2e-k8s-1.33-sig-compute"}, AlwaysRun: true, RunBeforeMerge: false, RegexpChangeMatcher: config.RegexpChangeMatcher{RunIfChanged: "", SkipIfOnlyChanged: ""}, Optional: false},
				config.Presubmit{JobBase: config.JobBase{Name: "pull-kubevirt-e2e-k8s-1.33-sig-storage"}, AlwaysRun: true, RunBeforeMerge: false, RegexpChangeMatcher: config.RegexpChangeMatcher{RunIfChanged: "", SkipIfOnlyChanged: ""}, Optional: false},
				config.Presubmit{JobBase: config.JobBase{Name: "pull-kubevirt-e2e-k8s-1.32-sig-compute"}, AlwaysRun: false, RunBeforeMerge: true, RegexpChangeMatcher: config.RegexpChangeMatcher{RunIfChanged: "", SkipIfOnlyChanged: ""}, Optional: false},
			}),
			Entry("k8sVersions blah", presubmits{
				NewAlwaysRun("pull-kubevirt-e2e-k8s-1.33-sig-compute"),
				NewAlwaysRun("pull-kubevirt-e2e-k8s-1.33-sig-compute-serial"),
				NewAlwaysRun("pull-kubevirt-e2e-k8s-1.33-sig-network"),
				NewAlwaysRun("pull-kubevirt-e2e-k8s-1.32-sig-compute-migrations"),
			}),
		)
	})
})

func names(p presubmits) []string {
	var result []string
	for _, presubmit := range p {
		result = append(result, presubmit.Name)
	}
	return result
}

type presubmitConfig func(p *config.Presubmit)

func alwaysRun() presubmitConfig {
	return func(p *config.Presubmit) {
		p.AlwaysRun = true
	}
}

func setName(name string) presubmitConfig {
	return func(p *config.Presubmit) {
		p.Name = name
	}
}

func NewAlwaysRun(name string) config.Presubmit {
	return NewPresubmit(setName(name), alwaysRun())
}

func NewPresubmit(configs ...presubmitConfig) config.Presubmit {
	result := &config.Presubmit{JobBase: config.JobBase{Name: ""}, AlwaysRun: false, RunBeforeMerge: false, RegexpChangeMatcher: config.RegexpChangeMatcher{RunIfChanged: "", SkipIfOnlyChanged: ""}, Optional: false}
	for _, c := range configs {
		c(result)
	}
	return *result
}
