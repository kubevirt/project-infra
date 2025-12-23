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
 * Copyright The KubeVirt Authors.
 */

package git

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("git blame", func() {

	DescribeTable("blameRegex parses",
		func(valueToParse string) {
			strings := blameRegex.FindAllStringSubmatch(valueToParse, -1)
			Expect(strings).To(HaveLen(1))
			Expect(strings[0]).To(HaveLen(7))
		},
		Entry("basic 1", "749cf0488 (Ben Oukhanov 2023-02-15 18:24:49 +0200  26) var _ = Describe(\"VM Console Proxy Operand\", func() {"),
		Entry("basic 2", `a32051d928 tests/operator_test.go     (fossedihelm   2022-11-30 09:45:57 +0100  111) var _ = Describe("[Serial][sig-operator]Operator", Serial, decorators.SigOperator, func() {`),
		Entry("special chars in name", `0df3f3c5129 (João Vilaça 2023-03-22 10:45:53 +0000 55) var _ = Describe("[Serial][sig-monitoring]VM Monitoring", Serial, decorators.SigMonitoring, func() {`),

		// It may happen for shallow clones that boundary commits are encountered
		Entry("git boundary commit containing caret", "^49cf0488 (Ben Oukhanov 2023-02-15 18:24:49 +0200  26) var _ = Describe(\"VM Console Proxy Operand\", func() {"),
	)

	Context("extractBlameInfo", func() {

		DescribeTable("extracts info as expected",

			func(info *expectedInfo, line string) {
				info.ExpectEquivalentTo(extractBlameInfo([]string{line})[0])
			},

			Entry("basic case line 1",
				newExpectedGitBlameInfo("749cf0488",
					"Ben Oukhanov",
					mustParse("2023-02-15 18:24:49 +0200"),
					26,
					`var _ = Describe("VM Console Proxy Operand", func() {`),
				`749cf0488 (Ben Oukhanov 2023-02-15 18:24:49 +0200  26) var _ = Describe("VM Console Proxy Operand", func() {`),
			Entry("basic case line 2",
				newExpectedGitBlameInfo("749cf0488",
					"Ben Oukhanov",
					mustParse("2023-02-15 18:24:49 +0200"),
					179,
					`\tContext("Resource change", func() {`),
				`749cf0488 (Ben Oukhanov 2023-02-15 18:24:49 +0200 179) \tContext("Resource change", func() {`),
			Entry("basic case line 3",
				newExpectedGitBlameInfo("749cf0488",
					"Ben Oukhanov",
					mustParse("2023-02-15 18:24:49 +0200"),
					208,
					`\t\tDescribeTable("should restore modified app labels", expectAppLabelsRestoreAfterUpdate,`),
				`749cf0488 (Ben Oukhanov 2023-02-15 18:24:49 +0200 208) \t\tDescribeTable("should restore modified app labels", expectAppLabelsRestoreAfterUpdate,`),
			Entry("basic case line 4",
				newExpectedGitBlameInfo("749cf0488",
					"Ben Oukhanov",
					mustParse("2023-02-15 18:24:49 +0200"),
					213,
					`\t\t\tEntry("[test_id:TODO] deployment", \u0026deploymentResource),`),
				`749cf0488 (Ben Oukhanov 2023-02-15 18:24:49 +0200 213) \t\t\tEntry("[test_id:TODO] deployment", \u0026deploymentResource),`),

			Entry("with changed filename line 1",
				newExpectedGitBlameInfo("a32051d928",
					"fossedihelm",
					mustParse("2022-11-30 09:45:57 +0100"),
					111,
					`var _ = Describe("[Serial][sig-operator]Operator", Serial, decorators.SigOperator, func() {`),
				`a32051d928 tests/operator_test.go     (fossedihelm   2022-11-30 09:45:57 +0100  111) var _ = Describe("[Serial][sig-operator]Operator", Serial, decorators.SigOperator, func() {`),
			Entry("with changed filename line 2",
				newExpectedGitBlameInfo("eb6dc428cd",
					"Igor Bezukh",
					mustParse("2022-03-15 18:18:47 +0200"),
					2928,
					`	Context("Obsolete ConfigMap", func() {`),
				`eb6dc428cd tests/operator_test.go     (Igor Bezukh   2022-03-15 18:18:47 +0200 2928) 	Context("Obsolete ConfigMap", func() {`),
			Entry("with changed filename line 3",
				newExpectedGitBlameInfo("52e637192d",
					"Daniel Hiller",
					mustParse("2023-05-26 10:21:35 +0200"),
					2942,
					`		It("[QUARANTINE] should emit event if the obsolete kubevirt-config configMap still exists", func() {`),
				`52e637192d tests/operator/operator.go (Daniel Hiller 2023-05-26 10:21:35 +0200 2942) 		It("[QUARANTINE] should emit event if the obsolete kubevirt-config configMap still exists", func() {`),

			Entry("with non standard chars in author name line 1",
				newExpectedGitBlameInfo("0df3f3c5129",
					"João Vilaça",
					mustParse("2023-03-22 10:45:53 +0000"),
					55,
					`var _ = Describe("[Serial][sig-monitoring]VM Monitoring", Serial, decorators.SigMonitoring, func() {`),
				`0df3f3c5129 (João Vilaça 2023-03-22 10:45:53 +0000 55) var _ = Describe("[Serial][sig-monitoring]VM Monitoring", Serial, decorators.SigMonitoring, func() {`),
			Entry("with non standard chars in author name line 2",
				newExpectedGitBlameInfo("ba2fdf5f25a",
					"João Vilaça",
					mustParse("2023-05-11 10:45:18 +0100"),
					63,
					`	Context("Cluster VM metrics", func() {`),
				`ba2fdf5f25a (João Vilaça 2023-05-11 10:45:18 +0100 63) 	Context("Cluster VM metrics", func() {`),
			Entry("with non standard chars in author name line 3",
				newExpectedGitBlameInfo("ba2fdf5f25a",
					"João Vilaça",
					mustParse("2023-05-11 10:45:18 +0100"),
					64,
					`		It("kubevirt_number_of_vms should reflect the number of VMs", func() {`),
				`ba2fdf5f25a (João Vilaça 2023-05-11 10:45:18 +0100 64) 		It("kubevirt_number_of_vms should reflect the number of VMs", func() {`),
		)

	})

	When("blame lines", Ordered, func() {
		var blameLines []*BlameLine
		BeforeAll(func() {
			var err error
			blameLines, err = GetBlameLinesForFile("testdata/repo/tests/test.go")
			Expect(err).ToNot(HaveOccurred())
		})
		It("works", func() {
			Expect(len(blameLines)).ToNot(BeEquivalentTo(0))
		})
	})
})

func newExpectedGitBlameInfo(expectedCommitID string,
	expectedAuthor string,
	expectedDate time.Time,
	expectedLineNo int,
	expectedLine string) *expectedInfo {
	return &expectedInfo{
		expectedCommitID: expectedCommitID,
		expectedAuthor:   expectedAuthor,
		expectedDate:     expectedDate,
		expectedLineNo:   expectedLineNo,
		expectedLine:     expectedLine,
	}
}

type expectedInfo struct {
	expectedCommitID string
	expectedAuthor   string
	expectedDate     time.Time
	expectedLineNo   int
	expectedLine     string
}

func (e expectedInfo) ExpectEquivalentTo(actual *BlameLine) {
	Expect(actual.CommitID).To(BeEquivalentTo(e.expectedCommitID))
	Expect(actual.Author).To(BeEquivalentTo(e.expectedAuthor))
	Expect(actual.Date).To(BeEquivalentTo(e.expectedDate))
	Expect(actual.LineNo).To(BeEquivalentTo(e.expectedLineNo))
	Expect(actual.Line).To(BeEquivalentTo(e.expectedLine))
}

func mustParse(gitDateValue string) time.Time {
	expectedDate, err := time.Parse(BlameDateLayout, gitDateValue)
	Expect(err).ToNot(HaveOccurred())
	return expectedDate
}
