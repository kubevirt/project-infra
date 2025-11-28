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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package test_label_analyzer

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("time.go", func() {
	Context("Since", func() {
		DescribeTable("Since",
			func(since time.Time, expected string) {
				Expect(Since(since)).To(BeEquivalentTo(expected))
			},
			Entry("now", time.Now(), "0 hours"),
			Entry("two hours", time.Now().Add(-2*time.Hour), "2 hours"),
			Entry("less than two days", time.Now().Add(-47*time.Hour), "47 hours"),
			Entry("two days", time.Now().Add(-48*time.Hour), "2 days"),
			Entry("7 days", time.Now().Add(7*-24*time.Hour), "7 days"),
			Entry("10 days", time.Now().Add(10*-24*time.Hour), "10 days"),
			Entry("14 days are two weeks", time.Now().Add(14*-24*time.Hour), "2 weeks"),
			Entry("fiftynine days are eight weeks", time.Now().Add(59*-24*time.Hour), "8 weeks"),
			Entry("sixty days are two months", time.Now().Add(60*-24*time.Hour), "2 months"),
			Entry("365 days are 12 months", time.Now().Add(365*-24*time.Hour), "12 months"),
			Entry("730 days are 2 years", time.Now().Add(730*-24*time.Hour), "2 years"),
		)
	})
})
