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

package filter

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("runFilter", func() {

	const expectedTestName = "[sig-storage] [rfe_id:6364][[Serial]Guestfs Run libguestfs on PVCs with root Should successfully run guestfs command on a filesystem-based PVC with root"

	Context("simple", func() {

		It("removes run test", func() {
			Expect(runFilter(&map[string]map[string]int{
				expectedTestName: {
					"test-kubevirt-cnv-4.12-compute-ocs":     1,
					"test-kubevirt-cnv-4.12-network-ovn-ocs": 1,
					"test-kubevirt-cnv-4.12-operator-ocs":    1,
					"test-kubevirt-cnv-4.12-quarantined-ocs": 1,
					"test-kubevirt-cnv-4.12-storage-ocs":     2,
				},
			}, defaultGroupConfigs)).To(BeEmpty())
		})

		It("does not remove not run test", func() {
			Expect(runFilter(&map[string]map[string]int{
				expectedTestName: {
					"test-kubevirt-cnv-4.12-compute-ocs":     1,
					"test-kubevirt-cnv-4.12-network-ovn-ocs": 1,
					"test-kubevirt-cnv-4.12-operator-ocs":    1,
					"test-kubevirt-cnv-4.12-quarantined-ocs": 1,
					"test-kubevirt-cnv-4.12-storage-ocs":     1,
				},
			}, defaultGroupConfigs)).To(BeEquivalentTo(map[string]map[string][]string{
				"storage": {
					"4.12": []string{expectedTestName},
				},
			}))
		})

	})

	Context("multiple", func() {

		const computeExpectedTestName = "[Serial][ref_id:2717][sig-compute]KubeVirt control plane resilience pod eviction evicting pods of control plane [test_id:2799]last eviction should fail for multi-replica virt-api pods"
		It("does encounter run test per version", func() {
			Expect(runFilter(&map[string]map[string]int{
				computeExpectedTestName: {
					"test-kubevirt-cnv-4.11-compute-ocs":     2,
					"test-kubevirt-cnv-4.11-network-ovn-ocs": 1,
					"test-kubevirt-cnv-4.11-operator-ocs":    1,
					"test-kubevirt-cnv-4.11-quarantined-ocs": 1,
					"test-kubevirt-cnv-4.11-storage-ocs":     1,
					"test-kubevirt-cnv-4.12-compute-ocs":     2,
					"test-kubevirt-cnv-4.12-network-ovn-ocs": 1,
					"test-kubevirt-cnv-4.12-operator-ocs":    1,
					"test-kubevirt-cnv-4.12-quarantined-ocs": 1,
					"test-kubevirt-cnv-4.12-storage-ocs":     1,
					"test-kubevirt-cnv-4.13-compute-ocs":     2,
					"test-kubevirt-cnv-4.13-network-ovn-ocs": 1,
					"test-kubevirt-cnv-4.13-operator-ocs":    1,
					"test-kubevirt-cnv-4.13-quarantined-ocs": 1,
					"test-kubevirt-cnv-4.13-storage-ocs":     1,
				},
			}, defaultGroupConfigs)).To(BeEmpty())
		})

		It("doesn't encounter run test for all versions", func() {
			Expect(runFilter(&map[string]map[string]int{
				computeExpectedTestName: {
					"test-kubevirt-cnv-4.11-compute-ocs":     2,
					"test-kubevirt-cnv-4.11-network-ovn-ocs": 1,
					"test-kubevirt-cnv-4.11-operator-ocs":    1,
					"test-kubevirt-cnv-4.11-quarantined-ocs": 1,
					"test-kubevirt-cnv-4.11-storage-ocs":     1,
					"test-kubevirt-cnv-4.12-compute-ocs":     1,
					"test-kubevirt-cnv-4.12-network-ovn-ocs": 1,
					"test-kubevirt-cnv-4.12-operator-ocs":    1,
					"test-kubevirt-cnv-4.12-quarantined-ocs": 1,
					"test-kubevirt-cnv-4.12-storage-ocs":     1,
					"test-kubevirt-cnv-4.13-compute-ocs":     1,
					"test-kubevirt-cnv-4.13-network-ovn-ocs": 1,
					"test-kubevirt-cnv-4.13-operator-ocs":    1,
					"test-kubevirt-cnv-4.13-quarantined-ocs": 1,
					"test-kubevirt-cnv-4.13-storage-ocs":     1,
				},
			}, defaultGroupConfigs)).To(BeEquivalentTo(map[string]map[string][]string{
				"virtualization": {
					"4.12": []string{computeExpectedTestName},
					"4.13": []string{computeExpectedTestName},
				},
			}))
		})

	})

	Context("ssp", func() {

		It("removes run ssp tests", func() {
			Expect(runFilter(&map[string]map[string]int{
				"DataSources without DataImportCron templates with added CDI label [test_id:8294] should remove CDI label from DataSource": {
					"test-ssp-cnv-4.11": 2,
					"test-ssp-cnv-4.12": 2,
				},
			}, defaultGroupConfigs)).To(BeEmpty())
		})

		It("doesn't remove skipped ssp tests", func() {
			Expect(runFilter(&map[string]map[string]int{
				"DataSources with DataImportCron template without existing PVC [QUARANTINE][test_id:8112] should restore DataSource if DataImportCron removed from SSP CR": {
					"test-ssp-cnv-4.12": 1,
				},
			}, defaultGroupConfigs)).To(BeEquivalentTo(map[string]map[string][]string{
				"ssp": {
					"4.12": []string{
						"DataSources with DataImportCron template without existing PVC [QUARANTINE][test_id:8112] should restore DataSource if DataImportCron removed from SSP CR",
					},
				},
			}))
		})

	})

	Context("unsupported", func() {

		It("removes unsupported tests", func() {
			Expect(runFilter(&map[string]map[string]int{
				"[Serial][sig-compute] Hyper-V enlightenments VMI with HyperV re-enlightenment enabled TSC frequency is not exposed on the cluster should be marked as non-migratable": {
					"test-kubevirt-cnv-4.11-compute-ocs":     2,
					"test-kubevirt-cnv-4.11-network-ovn-ocs": 1,
					"test-kubevirt-cnv-4.11-operator-ocs":    1,
					"test-kubevirt-cnv-4.11-quarantined-ocs": 1,
					"test-kubevirt-cnv-4.11-storage-ocs":     1,
					"test-kubevirt-cnv-4.12-compute-ocs":     3,
					"test-kubevirt-cnv-4.12-network-ovn-ocs": 3,
					"test-kubevirt-cnv-4.12-operator-ocs":    3,
					"test-kubevirt-cnv-4.12-quarantined-ocs": 3,
					"test-kubevirt-cnv-4.12-storage-ocs":     3,
					"test-kubevirt-cnv-4.13-compute-ocs":     2,
					"test-kubevirt-cnv-4.13-network-ovn-ocs": 1,
					"test-kubevirt-cnv-4.13-operator-ocs":    1,
					"test-kubevirt-cnv-4.13-quarantined-ocs": 1,
					"test-kubevirt-cnv-4.13-storage-ocs":     1,
				},
			}, defaultGroupConfigs)).To(BeEmpty())
		})

		It("does not remove skipped tests", func() {
			Expect(runFilter(&map[string]map[string]int{
				"[Serial][sig-compute] Hyper-V enlightenments VMI with HyperV re-enlightenment enabled TSC frequency is not exposed on the cluster should be marked as non-migratable": {
					"test-kubevirt-cnv-4.11-compute-ocs":     2,
					"test-kubevirt-cnv-4.11-network-ovn-ocs": 1,
					"test-kubevirt-cnv-4.11-operator-ocs":    1,
					"test-kubevirt-cnv-4.11-quarantined-ocs": 1,
					"test-kubevirt-cnv-4.11-storage-ocs":     1,
					"test-kubevirt-cnv-4.12-compute-ocs":     1,
					"test-kubevirt-cnv-4.12-network-ovn-ocs": 1,
					"test-kubevirt-cnv-4.12-operator-ocs":    1,
					"test-kubevirt-cnv-4.12-quarantined-ocs": 1,
					"test-kubevirt-cnv-4.12-storage-ocs":     1,
					"test-kubevirt-cnv-4.13-compute-ocs":     2,
					"test-kubevirt-cnv-4.13-network-ovn-ocs": 1,
					"test-kubevirt-cnv-4.13-operator-ocs":    1,
					"test-kubevirt-cnv-4.13-quarantined-ocs": 1,
					"test-kubevirt-cnv-4.13-storage-ocs":     1,
				},
			}, defaultGroupConfigs)).To(BeEquivalentTo(map[string]map[string][]string{
				"virtualization": {
					"4.12": []string{
						"[Serial][sig-compute] Hyper-V enlightenments VMI with HyperV re-enlightenment enabled TSC frequency is not exposed on the cluster should be marked as non-migratable",
					},
				},
			}))
		})

	})

})
