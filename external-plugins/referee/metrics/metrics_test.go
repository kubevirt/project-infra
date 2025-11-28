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
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("metrics", func() {
	Context("prometheus handler", func() {
		const host = "localhost"
		const port = 9798
		var srv *http.Server
		BeforeEach(func() {
			l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
			Expect(err).NotTo(HaveOccurred())
			Expect(l.Close()).ToNot(HaveOccurred())
			sm := http.NewServeMux()
			reset()
			AddMetricsHandler(sm)
			srv = &http.Server{
				Handler: sm,
				Addr:    fmt.Sprintf("%s:%d", host, port),
			}
			go func() {
				defer GinkgoRecover()
				err := srv.ListenAndServe()
				Expect(err).To(Equal(http.ErrServerClosed))
			}()

		})
		AfterEach(func() {
			err := srv.Shutdown(context.Background())
			Expect(err).NotTo(HaveOccurred())
		})
		type MetricsTestData struct {
			ExpectedMatchedLinesInBody []*regexp.Regexp
			Preparation                []func() error
		}
		DescribeTable("fetches metrics", func(td MetricsTestData) {

			for _, prep := range td.Preparation {
				Expect(prep()).ToNot(HaveOccurred())
			}
			client := http.Client{Timeout: 500 * time.Millisecond}
			defer client.CloseIdleConnections()

			var resp *http.Response
			Eventually(func(g Gomega) {
				var err error
				resp, err = client.Get(fmt.Sprintf("http://%s:%d%s", host, port, metricsPath))
				g.Expect(err).ToNot(HaveOccurred())
			}).WithTimeout(10 * time.Second).WithPolling(1 * time.Second).Should(Succeed())
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())

			bodyStr := string(body)
			for _, expected := range td.ExpectedMatchedLinesInBody {
				matched := false
				for _, line := range strings.Split(bodyStr, "\n") {
					if expected.MatchString(line) {
						matched = true
						break
					}
				}
				if !matched {
					Fail(fmt.Sprintf("Matching value %q not found in exposed metrics %q", expected, bodyStr))
				}
			}
		},
			Entry("initial state - only the total retests counter exists", MetricsTestData{
				ExpectedMatchedLinesInBody: []*regexp.Regexp{
					regexp.MustCompile(fmt.Sprintf("^%s 0$", generateRetestsMetricsName(totalRetestsCounterName))),
				},
			}),
			Entry("IncForRepository: org/repo counter is recorded, also total is increased", MetricsTestData{
				Preparation: []func() error{
					func() error {
						IncForRepository("org", "repo")
						return nil
					},
				},
				ExpectedMatchedLinesInBody: []*regexp.Regexp{
					regexp.MustCompile(fmt.Sprintf("^%s 1$", generateRetestsMetricsName(totalRetestsCounterName))),
					regexp.MustCompile(fmt.Sprintf("^%s 1$", fmt.Sprintf(generateRetestsMetricsName(retestsPerRepoCounterName), "org", "repo"))),
				},
			}),
			Entry("SetForPullRequest: org repo per pr gauge recorded", MetricsTestData{
				Preparation: []func() error{
					func() error {
						SetForPullRequest("org", "repo1", 1742, 37)
						SetForPullRequest("org", "repo1", 4217, 66)
						SetForPullRequest("org", "repo2", 1234, 98)
						SetForPullRequest("org", "repo2", 5678, 87)
						return nil
					},
				},
				ExpectedMatchedLinesInBody: []*regexp.Regexp{
					regexp.MustCompile(fmt.Sprintf(`^%s{pull_request="1742"} 37$`, fmt.Sprintf(generateRetestsMetricsName(retestsPerPRGaugeName), "org", "repo1"))),
					regexp.MustCompile(fmt.Sprintf(`^%s{pull_request="4217"} 66$`, fmt.Sprintf(generateRetestsMetricsName(retestsPerPRGaugeName), "org", "repo1"))),
					regexp.MustCompile(fmt.Sprintf(`^%s{pull_request="1234"} 98$`, fmt.Sprintf(generateRetestsMetricsName(retestsPerPRGaugeName), "org", "repo2"))),
					regexp.MustCompile(fmt.Sprintf(`^%s{pull_request="5678"} 87$`, fmt.Sprintf(generateRetestsMetricsName(retestsPerPRGaugeName), "org", "repo2"))),
				},
			}),
		)
	})
})

func generateRetestsMetricsName(simpleName string) string {
	return fmt.Sprintf("%s_%s_%s", promNamespace, promSubsystemForRetests, simpleName)
}
