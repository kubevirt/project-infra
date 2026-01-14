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

package searchci

import (
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("searchci", func() {

	const testName = `[test_id:3007] Should force restart a VM with terminationGracePeriodSeconds\u003e0`

	DescribeTable("ScrapeImpact extracts",

		func(text string, expected []Impact) {
			Expect(ScrapeImpact(text)).To(BeEquivalentTo(expected))
		},

		Entry("scrape impact with empty",
			"",
			nil,
		),
		Entry("basic scrape impact, no build urls",
			`<tr><td colspan="4"><a target="_blank" href="https://prow.ci.kubevirt.io/job-history/kubevirt-prow/pr-logs/directory/pull-kubevirt-e2e-k8s-1.30-sig-compute">pull-kubevirt-e2e-k8s-1.30-sig-compute</a> <a href="/?context=1&amp;excludeName=&amp;groupByJob=job&amp;maxAge=672h0m0s&amp;maxBytes=20971520&amp;maxMatches=5&amp;mode=text&amp;name=%5Epull-kubevirt-e2e-k8s-1%5C.30-sig-compute%24&amp;search=%5C%5Btest_id%3A3007%5D+Should+force+restart+a+VM+with+terminationGracePeriodSeconds%5Cu003e0&amp;searchType=junit&amp;wrapLines=false">(all)</a> - <em title="107 runs, 7 failures, 5 matching runs">107 runs, 7% failed, 71% of failures match = 5% impact</em></td></tr>
`,
			[]Impact{
				{
					URL:          "https://prow.ci.kubevirt.io/job-history/kubevirt-prow/pr-logs/directory/pull-kubevirt-e2e-k8s-1.30-sig-compute",
					Percent:      5.0,
					URLToDisplay: "pull-kubevirt-e2e-k8s-1.30-sig-compute",
				},
			},
		),
		Entry("scrape impact with build urls",
			`<tr><td colspan="4"><a target="_blank" href="https://prow.ci.kubevirt.io/job-history/kubevirt-prow/pr-logs/directory/pull-kubevirt-e2e-k8s-1.30-sig-compute">pull-kubevirt-e2e-k8s-1.30-sig-compute</a> <a href="/?context=1&amp;excludeName=&amp;groupByJob=job&amp;maxAge=672h0m0s&amp;maxBytes=20971520&amp;maxMatches=5&amp;mode=text&amp;name=%5Epull-kubevirt-e2e-k8s-1%5C.30-sig-compute%24&amp;search=%5C%5Btest_id%3A3007%5D+Should+force+restart+a+VM+with+terminationGracePeriodSeconds%5Cu003e0&amp;searchType=junit&amp;wrapLines=false">(all)</a> - <em title="107 runs, 7 failures, 5 matching runs">107 runs, 7% failed, 71% of failures match = 5% impact</em></td></tr>
<tr class="row-match"><td><a target="_blank" href="https://prow.ci.kubevirt.io/view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/14632/pull-kubevirt-e2e-k8s-1.30-sig-compute/1932345605980426240">#1932345605980426240</a></td><td>junit</td><td class="text-nowrap">36 hours ago</td><td class="col-12"></td></tr>
<tr class="row-match"><td class="" colspan="4"><pre class="small"># [sig-compute] [rfe_id:1177][crit:medium] VirtualMachine [test_id:3007] Should force restart a VM with terminationGracePeriodSeconds&gt;0
tests/compute/vm_lifecycle.go:51
</pre></td></tr>
<tr class="row-match"><td><a target="_blank" href="https://prow.ci.kubevirt.io/view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/14884/pull-kubevirt-e2e-k8s-1.30-sig-compute/1932327352512024576">#1932327352512024576</a></td><td>junit</td><td class="text-nowrap">37 hours ago</td><td class="col-12"></td></tr>
<tr class="row-match"><td class="" colspan="4"><pre class="small"># [sig-compute] [rfe_id:1177][crit:medium] VirtualMachine [test_id:3007] Should force restart a VM with terminationGracePeriodSeconds&gt;0
tests/compute/vm_lifecycle.go:51
</pre></td></tr>
<tr class="row-match"><td><a target="_blank" href="https://prow.ci.kubevirt.io/view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/14802/pull-kubevirt-e2e-k8s-1.30-sig-compute/1930397210671845376">#1930397210671845376</a></td><td>junit</td><td class="text-nowrap">6 days ago</td><td class="col-12"></td></tr>
`,
			[]Impact{
				{
					URL:          "https://prow.ci.kubevirt.io/job-history/kubevirt-prow/pr-logs/directory/pull-kubevirt-e2e-k8s-1.30-sig-compute",
					Percent:      5.0,
					URLToDisplay: "pull-kubevirt-e2e-k8s-1.30-sig-compute",
					BuildURLs: []JobBuildURL{
						{
							URL:      "https://prow.ci.kubevirt.io/view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/14632/pull-kubevirt-e2e-k8s-1.30-sig-compute/1932345605980426240",
							Interval: time.Hour * 36,
						},
						{
							URL:      "https://prow.ci.kubevirt.io/view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/14884/pull-kubevirt-e2e-k8s-1.30-sig-compute/1932327352512024576",
							Interval: time.Hour * 37,
						},
						{
							URL:      "https://prow.ci.kubevirt.io/view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/14802/pull-kubevirt-e2e-k8s-1.30-sig-compute/1930397210671845376",
							Interval: time.Hour * 24 * 6,
						},
					},
				},
			},
		),
		Entry("scrape with no failures",
			`<tr><td colspan="4"><a target="_blank" href="https://prow.ci.kubevirt.io/job-history/kubevirt-prow/pr-logs/directory/pull-kubernetes-nmstate-e2e-operator-k8s">pull-kubernetes-nmstate-e2e-operator-k8s</a> <a href="/?context=1&amp;excludeName=&amp;groupByJob=job&amp;maxAge=672h0m0s&amp;maxBytes=20971520&amp;maxMatches=1&amp;mode=text&amp;name=%5Epull-kubernetes-nmstate-e2e-operator-k8s%24&amp;search=AfterSuite&amp;searchType=junit&amp;wrapLines=false">(all)</a> - <em title="3 runs, 0 failures, 3 matching runs">3 runs, 0% failed, 100% of runs match</em></td></tr>
<tr class="row-match"><td><a target="_blank" href="https://prow.ci.kubevirt.io/view/gs/kubevirt-prow/pr-logs/pull/nmstate_kubernetes-nmstate/1354/pull-kubernetes-nmstate-e2e-operator-k8s/1947282574778830848">#1947282574778830848</a></td><td>junit</td><td class="text-nowrap">18 hours ago</td><td class="col-12"></td></tr>
<tr class="row-match"><td class="" colspan="4"><pre class="small"># Operator E2E Test Suite.[AfterSuite]
&gt; Enter [AfterSuite] TOP-LEVEL - /tmp/knmstate/kubernetes-nmstate/test/e2e/operator/main_test.go:76 @ 07/21/25 13:42:10.335

... 1 lines not shown

</pre></td></tr>
<tr class="row-match"><td><a target="_blank" href="https://prow.ci.kubevirt.io/view/gs/kubevirt-prow/pr-logs/pull/nmstate_kubernetes-nmstate/1354/pull-kubernetes-nmstate-e2e-operator-k8s/1947235763452121088">#1947235763452121088</a></td><td>junit</td><td class="text-nowrap">21 hours ago</td><td class="col-12"></td></tr>`,
			[]Impact{
				{
					URL:          "https://prow.ci.kubevirt.io/job-history/kubevirt-prow/pr-logs/directory/pull-kubernetes-nmstate-e2e-operator-k8s",
					Percent:      0,
					URLToDisplay: "pull-kubernetes-nmstate-e2e-operator-k8s",
					BuildURLs: []JobBuildURL{
						{
							URL:      "https://prow.ci.kubevirt.io/view/gs/kubevirt-prow/pr-logs/pull/nmstate_kubernetes-nmstate/1354/pull-kubernetes-nmstate-e2e-operator-k8s/1947282574778830848",
							Interval: time.Hour * 18,
						},
						{
							URL:      "https://prow.ci.kubevirt.io/view/gs/kubevirt-prow/pr-logs/pull/nmstate_kubernetes-nmstate/1354/pull-kubernetes-nmstate-e2e-operator-k8s/1947235763452121088",
							Interval: time.Hour * 21,
						},
					},
				},
			},
		),
		Entry("scrape with minutes ago",
			`<tr><td colspan="4"><a target="_blank" href="https://prow.ci.kubevirt.io/job-history/kubevirt-prow/pr-logs/directory/pull-kubevirt-e2e-k8s-1.32-sig-compute-migrations-1.6">pull-kubevirt-e2e-k8s-1.32-sig-compute-migrations-1.6</a> <a href="/?context=1&amp;excludeName=&amp;groupByJob=job&amp;maxAge=672h0m0s&amp;maxBytes=20971520&amp;maxMatches=1&amp;mode=text&amp;name=%5Epull-kubevirt-e2e-k8s-1%5C.32-sig-compute-migrations-1%5C.6%24&amp;search=%5C%5Brfe_id%3A393%5D%5C%5Bcrit%3Ahigh%5D%5C%5Bvendor%3Acnv-qe%40redhat.com%5D%5C%5Blevel%3Asystem%5D%5C%5Bsig-compute%5D+Live+Migration+across+namespaces+container+disk+should+live+migrate+a+container+disk+vm%2C+with+an+additional+PVC+mounted%2C+should+stay+mounted+after+migration&amp;searchType=junit&amp;wrapLines=false">(all)</a> - <em title="14 runs, 7 failures, 4 matching runs">14 runs, 50% failed, 57% of failures match = 29% impact</em></td></tr>
<tr class="row-match"><td><a target="_blank" href="https://prow.ci.kubevirt.io/view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15256/pull-kubevirt-e2e-k8s-1.32-sig-compute-migrations-1.6/1947961013684932608">#1947961013684932608</a></td><td>junit</td><td class="text-nowrap">34 minutes ago</td><td class="col-12"></td></tr>
<tr class="row-match"><td class="" colspan="4"><pre class="small"># [rfe_id:393][crit:high][vendor:cnv-qe@redhat.com][level:system][sig-compute] Live Migration across namespaces container disk should live migrate a container disk vm, with an additional PVC mounted, should stay mounted after migration
tests/migration/namespace.go:236
</pre></td></tr>
<tr class="row-match"><td><a target="_blank" href="https://prow.ci.kubevirt.io/view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15246/pull-kubevirt-e2e-k8s-1.32-sig-compute-migrations-1.6/1947958310749605888">#1947958310749605888</a></td><td>junit</td><td class="text-nowrap">44 minutes ago</td><td class="col-12"></td></tr>
<tr class="row-match"><td class="" colspan="4"><pre class="small"># [rfe_id:393][crit:high][vendor:cnv-qe@redhat.com][level:system][sig-compute] Live Migration across namespaces container disk should live migrate a container disk vm, with an additional PVC mounted, should stay mounted after migration
tests/migration/namespace.go:236
</pre></td></tr>
<tr class="row-match"><td><a target="_blank" href="https://prow.ci.kubevirt.io/view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15256/pull-kubevirt-e2e-k8s-1.32-sig-compute-migrations-1.6/1947915060424740864">#1947915060424740864</a></td><td>junit</td><td class="text-nowrap">4 hours ago</td><td class="col-12"></td></tr>
<tr class="row-match"><td class="" colspan="4"><pre class="small"># [rfe_id:393][crit:high][vendor:cnv-qe@redhat.com][level:system][sig-compute] Live Migration across namespaces container disk should live migrate a container disk vm, with an additional PVC mounted, should stay mounted after migration
tests/migration/namespace.go:236
</pre></td></tr>
<tr class="row-match"><td><a target="_blank" href="https://prow.ci.kubevirt.io/view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15256/pull-kubevirt-e2e-k8s-1.32-sig-compute-migrations-1.6/1947746664915668992">#1947746664915668992</a></td><td>junit</td><td class="text-nowrap">15 hours ago</td><td class="col-12"></td></tr>
<tr class="row-match"><td class="" colspan="4"><pre class="small"># [rfe_id:393][crit:high][vendor:cnv-qe@redhat.com][level:system][sig-compute] Live Migration across namespaces container disk should live migrate a container disk vm, with an additional PVC mounted, should stay mounted after migration
tests/migration/namespace.go:236
</pre></td></tr>`,
			[]Impact{
				{
					URL:          "https://prow.ci.kubevirt.io/job-history/kubevirt-prow/pr-logs/directory/pull-kubevirt-e2e-k8s-1.32-sig-compute-migrations-1.6",
					Percent:      29,
					URLToDisplay: "pull-kubevirt-e2e-k8s-1.32-sig-compute-migrations-1.6",
					BuildURLs: []JobBuildURL{
						{
							URL:      "https://prow.ci.kubevirt.io/view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15256/pull-kubevirt-e2e-k8s-1.32-sig-compute-migrations-1.6/1947961013684932608",
							Interval: time.Minute * 34,
						},
						{
							URL:      "https://prow.ci.kubevirt.io/view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15246/pull-kubevirt-e2e-k8s-1.32-sig-compute-migrations-1.6/1947958310749605888",
							Interval: time.Minute * 44,
						},
						{
							URL:      "https://prow.ci.kubevirt.io/view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15256/pull-kubevirt-e2e-k8s-1.32-sig-compute-migrations-1.6/1947915060424740864",
							Interval: time.Hour * 4,
						},
						{
							URL:      "https://prow.ci.kubevirt.io/view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15256/pull-kubevirt-e2e-k8s-1.32-sig-compute-migrations-1.6/1947746664915668992",
							Interval: time.Hour * 15,
						},
					},
				},
			},
		),
	)
	Context("NewScrapeURL", func() {

		It("escapes the test name correctly", func() {
			Expect(NewScrapeURL(testName, FourteenDays)).To(
				BeEquivalentTo(`https://search.ci.kubevirt.io/?search=%5C%5Btest_id%3A3007%5D+Should+force+restart+a+VM+with+terminationGracePeriodSeconds%5Cu003e0&maxAge=336h&context=1&type=junit&name=&excludeName=periodic-.*&maxMatches=1&maxBytes=20971520&groupBy=job`),
			)
		})
	})
	DescribeTable("FilterRelevantImpacts",
		func(impacts []Impact, timeRange TimeRange, relevantimpacts []Impact) {
			Expect(FilterImpacts(impacts, timeRange)).To(BeEquivalentTo(relevantimpacts))
		},
		Entry("filters a relevant impact for fourteen days",
			[]Impact{
				{
					URL:     "42",
					Percent: 3.0,
				},
				{
					URL:     "37",
					Percent: 5.0,
				},
			},
			FourteenDays,
			[]Impact{
				{
					URL:     "37",
					Percent: 5.0,
				},
			},
		),
		Entry("tit for tat",
			nil,
			FourteenDays,
			nil,
		),
		Entry("filters a relevant impact for three days",
			[]Impact{
				{
					URL:     "42",
					Percent: 20.0,
				},
				{
					URL:     "37",
					Percent: 17.0,
				},
			},
			ThreeDays,
			[]Impact{
				{
					URL:     "42",
					Percent: 20.0,
				},
			},
		),
	)
	Context("ScrapeImpacts", func() {
		var body []byte
		var server *httptest.Server
		BeforeEach(func() {
			var err error
			body, err = os.ReadFile("testdata/searchci.html")
			Expect(err).ToNot(HaveOccurred())
			server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				writer.Write(body)
				writer.WriteHeader(http.StatusOK)
			}))
			serviceURL = server.URL
		})
		AfterEach(func() {
			server.Close()
		})
		It("scrapes", func() {
			impacts, err := ScrapeImpacts(testName, FourteenDays)
			Expect(err).ToNot(HaveOccurred())
			Expect(impacts).ToNot(BeNil())
		})
	})
})
