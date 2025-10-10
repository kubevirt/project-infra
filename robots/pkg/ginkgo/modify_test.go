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

package ginkgo

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("modify", func() {

	var (
		text = `
			Entry("with RunStrategyOnce", v1.RunStrategyOnce, func(vm *v1.VirtualMachine) {
				By("Starting the VM")
				err = virtClient.VirtualMachine(vm.Namespace).Start(context.Background(), vm.Name, &v1.StartOptions{})
				Expect(err).To(MatchError(ContainSubstring("Once does not support manual start requests")))
			}),
			Entry("[test_id:2190] with RunStrategyManual", v1.RunStrategyManual, func(vm *v1.VirtualMachine) {
				// At this point, explicitly test that a start command will delete an existing
				// VMI in the Succeeded phase.
				By("Starting the VM")
				err = virtClient.VirtualMachine(vm.Namespace).Start(context.Background(), vm.Name, &v1.StartOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Waiting for StartRequest to be cleared")
				Eventually(ThisVM(vm), 240*time.Second, 1*time.Second).Should(Not(HaveStateChangeRequests()))

				By("Waiting for VM to be ready")
				Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(BeReady())
			}),
`
		expectedModification = `
			Entry("with RunStrategyOnce", v1.RunStrategyOnce, func(vm *v1.VirtualMachine) {
				By("Starting the VM")
				err = virtClient.VirtualMachine(vm.Namespace).Start(context.Background(), vm.Name, &v1.StartOptions{})
				Expect(err).To(MatchError(ContainSubstring("Once does not support manual start requests")))
			}),
			Entry("[QUARANTINE][test_id:2190] with RunStrategyManual", decorators.Quarantine, v1.RunStrategyManual, func(vm *v1.VirtualMachine) {
				// At this point, explicitly test that a start command will delete an existing
				// VMI in the Succeeded phase.
				By("Starting the VM")
				err = virtClient.VirtualMachine(vm.Namespace).Start(context.Background(), vm.Name, &v1.StartOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Waiting for StartRequest to be cleared")
				Eventually(ThisVM(vm), 240*time.Second, 1*time.Second).Should(Not(HaveStateChangeRequests()))

				By("Waiting for VM to be ready")
				Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(BeReady())
			}),
`
	)

	When("modify is used", func() {

		It("doesn't modify anything if input is empty", func() {
			_, err := modify(text, "", "")
			Expect(err).To(HaveOccurred())
		})

		It("modifies it as expected", func() {
			Expect(modify(text, `"[test_id:2190] with RunStrategyManual"`, `"[QUARANTINE][test_id:2190] with RunStrategyManual", decorators.Quarantine`)).To(BeEquivalentTo(expectedModification))
		})
	})

	When("quarantine is used", func() {
		It("does quarantine a test", func() {
			Expect(quarantine(text, "[test_id:2190] with RunStrategyManual")).To(BeEquivalentTo(expectedModification))
		})
	})

})
