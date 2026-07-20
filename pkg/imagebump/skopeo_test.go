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
 *
 */

package imagebump

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("skopeo", func() {
	Context("listTags", func() {
		var oldlistTagsCommand listTagsCommand
		var oldinspectCommand inspectCommand
		BeforeEach(func() {
			oldlistTagsCommand = listTagsCmd
			oldinspectCommand = inspectCmd
		})
		AfterEach(func() {
			listTagsCmd = oldlistTagsCommand
			inspectCmd = oldinspectCommand
		})
		DescribeTable("sorts",
			func(imageRef string, listTagsOutput []string, expectedTag string, expectsListTagsErr bool, inspectCreatedByFullImageRef map[string]time.Time) {
				listTagsCmd = newListTagsCommand(listTagsOutput)
				inspectCmd = newInspectCommand(inspectCreatedByFullImageRef)
				tag, err := LatestSkopeoTag(imageRef)
				if expectsListTagsErr {
					Expect(err).To(HaveOccurred())
				} else {
					Expect(tag, err).To(BeEquivalentTo(expectedTag))
					Expect(err).ToNot(HaveOccurred())
				}
			},
			Entry("none",
				"quay.io/kubevirtci/ginkgo-tests",
				[]string{},
				"",
				true,
				map[string]time.Time{},
			),
			Entry("different by day",
				"quay.io/kubevirtci/ginkgo-tests",
				[]string{
					"v20260711-7a66da0",
					"v20260716-a2b9efe",
				},
				"v20260716-a2b9efe",
				false,
				map[string]time.Time{},
			),
			Entry("different by hour",
				"quay.io/kubevirtci/ginkgo-tests",
				[]string{
					"v20260711-7a66da0",
					"v20260716-a2b9efe",
					"v20260716-ff8cd14",
				},
				"v20260716-a2b9efe",
				false,
				map[string]time.Time{
					"quay.io/kubevirtci/ginkgo-tests:v20260716-a2b9efe": time.Date(2026, 07, 16, 10, 0, 0, 0, time.UTC),
					"quay.io/kubevirtci/ginkgo-tests:v20260716-ff8cd14": time.Date(2026, 07, 16, 1, 0, 0, 0, time.UTC)},
			),
		)
	})
})

func newListTagsCommand(output []string) listTagsCommand {
	return func(imageRef string) ([]string, error) {
		return output, nil
	}
}

func newInspectCommand(inspectCreatedByFullImageRef map[string]time.Time) inspectCommand {
	return func(fullImageRef string) (time.Time, error) {
		return inspectCreatedByFullImageRef[fullImageRef], nil
	}
}
