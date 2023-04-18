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

	It("removes run test", func() {
		Expect(runFilter(
			&map[string]map[string]int{
				"[sig-storage] [rfe_id:6364][[Serial]Guestfs Run libguestfs on PVCs with root Should successfully run guestfs command on a filesystem-based PVC with root": {
					"test-kubevirt-cnv-4.12-compute-ocs":     1,
					"test-kubevirt-cnv-4.12-network-ovn-ocs": 1,
					"test-kubevirt-cnv-4.12-operator-ocs":    1,
					"test-kubevirt-cnv-4.12-quarantined-ocs": 1,
					"test-kubevirt-cnv-4.12-storage-ocs":     2,
				},
			},
		)).To(BeEmpty())
	})

	It("does not remove not run test", func() {
		Expect(runFilter(
			&map[string]map[string]int{
				"[sig-storage] [rfe_id:6364][[Serial]Guestfs Run libguestfs on PVCs with root Should successfully run guestfs command on a filesystem-based PVC with root": {
					"test-kubevirt-cnv-4.12-compute-ocs":     1,
					"test-kubevirt-cnv-4.12-network-ovn-ocs": 1,
					"test-kubevirt-cnv-4.12-operator-ocs":    1,
					"test-kubevirt-cnv-4.12-quarantined-ocs": 1,
					"test-kubevirt-cnv-4.12-storage-ocs":     1,
				},
			},
		)).To(BeEquivalentTo(
			[]string{
				"[sig-storage] [rfe_id:6364][[Serial]Guestfs Run libguestfs on PVCs with root Should successfully run guestfs command on a filesystem-based PVC with root",
			},
		))
	})

})
