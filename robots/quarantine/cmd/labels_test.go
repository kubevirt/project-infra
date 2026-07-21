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

package cmd

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("labels", func() {
	When("loading valid groups from labels file", func() {
		var tempFile *os.File
		BeforeEach(func() {
			var err error
			tempFile, err = os.CreateTemp("", "labels-*.yaml")
			Expect(err).ToNot(HaveOccurred())
		})
		AfterEach(func() {
			Expect(os.Remove(tempFile.Name())).ToNot(HaveOccurred())
		})

		It("extracts sig and wg labels from default section", func() {
			_, err := tempFile.WriteString(`---
default:
  labels:
    - name: sig/compute
      color: c5def5
    - name: sig/network
      color: c5def5
    - name: sig/storage
      color: c5def5
    - name: wg/aie
      color: E3F582
    - name: wg/arch-s390x
      color: E3F582
    - name: kind/bug
      color: ee0701
    - name: lgtm
      color: 15dd18
`)
			Expect(err).ToNot(HaveOccurred())
			Expect(tempFile.Close()).ToNot(HaveOccurred())

			sigs, wgs, err := loadValidGroupsFromLabelsFile(tempFile.Name())
			Expect(err).ToNot(HaveOccurred())
			Expect(sigs).To(Equal(map[string]bool{
				"compute": true,
				"network": true,
				"storage": true,
			}))
			Expect(wgs).To(Equal(map[string]bool{
				"aie":        true,
				"arch-s390x": true,
			}))
		})

		It("fails when file does not exist", func() {
			_, _, err := loadValidGroupsFromLabelsFile("/nonexistent/labels.yaml")
			Expect(err).To(HaveOccurred())
		})

		It("fails when no SIG labels are found", func() {
			_, err := tempFile.WriteString(`---
default:
  labels:
    - name: kind/bug
      color: ee0701
`)
			Expect(err).ToNot(HaveOccurred())
			Expect(tempFile.Close()).ToNot(HaveOccurred())

			_, _, err = loadValidGroupsFromLabelsFile(tempFile.Name())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no SIG labels found"))
		})
	})

	When("resolving prow commands", func() {
		var validSIGs, validWGs map[string]bool

		BeforeEach(func() {
			validSIGs = map[string]bool{
				"compute":       true,
				"network":       true,
				"storage":       true,
				"observability": true,
				"scale":         true,
			}
			validWGs = map[string]bool{
				"arch-s390x": true,
				"arch-arm":   true,
				"aie":        true,
			}
		})

		DescribeTable("it",
			func(ginkgoLabel, expected string) {
				Expect(resolveProwCommand(ginkgoLabel, validSIGs, validWGs)).To(Equal(expected))
			},
			Entry("direct SIG match", "sig-compute", "sig compute"),
			Entry("direct SIG match for network", "sig-network", "sig network"),
			Entry("direct SIG match for storage", "sig-storage", "sig storage"),
			Entry("prefix match for sig-compute-migrations", "sig-compute-migrations", "sig compute"),
			Entry("prefix match for sig-compute-instancetype", "sig-compute-instancetype", "sig compute"),
			Entry("prefix match for sig-compute-realtime", "sig-compute-realtime", "sig compute"),
			Entry("alias for sig-monitoring", "sig-monitoring", "sig observability"),
			Entry("alias for sig-performance", "sig-performance", "sig scale"),
			Entry("compound alias for sig-monitoring-alerts", "sig-monitoring-alerts", "sig observability"),
			Entry("compound alias for sig-performance-latency", "sig-performance-latency", "sig scale"),
			Entry("alias for wg-s390x", "wg-s390x", "wg arch-s390x"),
			Entry("alias for wg-arm64", "wg-arm64", "wg arch-arm"),
			Entry("compound alias for wg-s390x-ci", "wg-s390x-ci", "wg arch-s390x"),
			Entry("fallback for sig-operator", "sig-operator", "sig compute"),
			Entry("direct WG match", "wg-aie", "wg aie"),
			Entry("fallback for unknown sig", "sig-unknown", "sig compute"),
			Entry("fallback for unknown wg", "wg-unknown", "sig compute"),
		)
	})
})
