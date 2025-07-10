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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"net/http"
	"net/http/httptest"
	"os"
)

var _ = Describe("searchci", func() {

	const testName = `[test_id:3007] Should force restart a VM with terminationGracePeriodSeconds\u003e0`

	Context("ScrapeImpact extracts", func() {

		const text = `<td colspan="4"><a target="_blank" href="https://prow.ci.kubevirt.io/job-history/kubevirt-prow/pr-logs/directory/pull-kubevirt-e2e-k8s-1.30-sig-compute">pull-kubevirt-e2e-k8s-1.30-sig-compute</a> <a href="/?context=1&amp;excludeName=&amp;groupByJob=job&amp;maxAge=672h0m0s&amp;maxBytes=20971520&amp;maxMatches=5&amp;mode=text&amp;name=%5Epull-kubevirt-e2e-k8s-1%5C.30-sig-compute%24&amp;search=%5C%5Btest_id%3A3007%5D+Should+force+restart+a+VM+with+terminationGracePeriodSeconds%5Cu003e0&amp;searchType=junit&amp;wrapLines=false">(all)</a> - <em title="115 runs, 8 failures, 6 matching runs">115 runs, 7% failed, 75% of failures match = 5% impact</em></td>`

		It("something", func() {
			Expect(ScrapeImpact(text)).To(Not(BeNil()))
		})
		It("content", func() {
			Expect(ScrapeImpact(text)[0]).To(BeEquivalentTo(Impact{
				URL:          "https://prow.ci.kubevirt.io/job-history/kubevirt-prow/pr-logs/directory/pull-kubevirt-e2e-k8s-1.30-sig-compute",
				Percent:      5.0,
				URLToDisplay: "pull-kubevirt-e2e-k8s-1.30-sig-compute",
			}))
		})
	})
	Context("NewScrapeURL", func() {

		It("escapes the test name correctly", func() {
			Expect(NewScrapeURL(testName, FourteenDays)).To(
				BeEquivalentTo(`https://search.ci.kubevirt.io/?search=%5C%5Btest_id%3A3007%5D+Should+force+restart+a+VM+with+terminationGracePeriodSeconds%5Cu003e0&maxAge=336h&context=1&type=junit&name=&excludeName=periodic-.*&maxMatches=1&maxBytes=20971520&groupBy=job`),
			)
		})
	})
	DescribeTable("FilterRelevantImpacts",
		func(impacts []Impact, timeRange TimeRange, relevantimpacts []Impact) {
			Expect(FilterRelevantImpacts(impacts, timeRange)).To(BeEquivalentTo(relevantimpacts))
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
	Context("ScrapeRelevantImpacts", func() {
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
			impacts, err := ScrapeRelevantImpacts(testName, FourteenDays)
			Expect(err).ToNot(HaveOccurred())
			Expect(impacts).ToNot(BeNil())
		})
	})
})
