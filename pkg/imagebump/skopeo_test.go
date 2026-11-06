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
			Entry("new unprefixed plugin tags alongside old v-prefixed tags",
				"quay.io/kubevirtci/phased",
				[]string{"202611050920-2a81b11", "v202607231917-ca75c4e"},
				"202611050920-2a81b11", false, map[string]time.Time{},
			),
			Entry("a newer v-prefixed tag in unordered registry output",
				"quay.io/kubevirtci/phased",
				[]string{"v202611050920-2a81b11", "202607231917-ca75c4e", "v20260801-1234567"},
				"v202611050920-2a81b11", false, map[string]time.Time{},
			),
			Entry("unordered minute timestamps",
				"quay.io/kubevirtci/phased",
				[]string{"202611050920-2a81b11", "202611050800-ca75c4e", "202611040552-be2ec11"},
				"202611050920-2a81b11", false, map[string]time.Time{},
			),
			Entry("a day tag and a minute tag on the same day",
				"quay.io/kubevirtci/phased",
				[]string{"v202611050920-2a81b11", "v20261105-ca75c4e"},
				"v202611050920-2a81b11", false, map[string]time.Time{},
			),
			Entry("legacy two-digit years alongside modern timestamps",
				"quay.io/kubevirtci/gocli",
				[]string{"2611052237-e2b9d71c", "20250204-istiobump"},
				"2611052237-e2b9d71c", false, map[string]time.Time{},
			),
			Entry("ignores malformed dates and non-release tags",
				"quay.io/kubevirtci/phased",
				[]string{"latest", "v20261305-ca75c4e", "v20261105-2a81b11-amd64", "202611050920-2a81b11"},
				"202611050920-2a81b11", false, map[string]time.Time{},
			),
			Entry("no parseable timestamps",
				"quay.io/kubevirtci/phased",
				[]string{"latest", "v20261305-ca75c4e"},
				"", true, map[string]time.Time{},
			),
		)
	})

	DescribeTable("parses tag timestamps",
		func(tag string, expected time.Time, valid bool) {
			stamp, ok := kubevirtCITagTime(tag)
			Expect(ok).To(Equal(valid))
			Expect(stamp).To(Equal(expected))
		},
		Entry("prefixed day", "v20261105-2a81b11", time.Date(2026, 11, 5, 0, 0, 0, 0, time.UTC), true),
		Entry("unprefixed minute", "202611050920-2a81b11", time.Date(2026, 11, 5, 9, 20, 0, 0, time.UTC), true),
		Entry("legacy gocli", "2611052237-e2b9d71c", time.Date(2026, 11, 5, 22, 37, 0, 0, time.UTC), true),
		Entry("invalid date", "v20260230-2a81b11", time.Time{}, false),
		Entry("unsupported length", "v202611050-2a81b11", time.Time{}, false),
		Entry("non-release tag", "latest", time.Time{}, false),
	)
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
