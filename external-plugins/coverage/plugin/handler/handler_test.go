// TODO: Add unit tests using Ginkgo

package handler

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/prow/pkg/github"
)

var _ = Describe("DetectGoFileChanges", func() {


	Context("A file is in the coverage directory", func() {
		It("Return true for external-plugins directory", func() {
			files := []string{"external-plugins/coverage/plugin/handler/handler.go"}
			result := DetectGoFileChanges(files)
			Expect(result).To(BeTrue())
		})

		It("Return true for releng path", func() {
			files := []string{"releng/release-tool/release-tool.go"}
			result := DetectGoFileChanges(files)
			Expect(result).To(BeTrue())
		})

		It("Return true for robots path", func() {
			files := []string{"robots/flakefinder/flakefinder.go"}
			result := DetectGoFileChanges(files)
			Expect(result).To(BeTrue())
		})
		It("Return true for rehearse path", func() {
			files := []string{"external-plugins/rehearse/plugin/handler/handler.go"}
			result := DetectGoFileChanges(files)
			Expect(result).To(BeTrue())
		})
		It("Return true for coverage path", func() {
			files := []string{"external-plugins/coverage/plugin/handler/handler.go"}
			result := DetectGoFileChanges(files)
			Expect(result).To(BeTrue())
		})
	})

	Context("A file is not in coverage directiry", func(){
		It("Returns false for  github path", func() {
			files := []string{"github/ci/prow-deplooy/config.yaml"}
			result := DetectGoFileChanges(files)
			Expect(result).To(BeFalse())
		})
		It("Return false for root level file", func () {
			files := []string{"README.md"}
			result := DetectGoFileChanges(files)
			Expect(result).To(BeFalse())
		})
		It("Retun false on an empty string", func() {
			files := []string{}
			result := DetectGoFileChanges(files)
			Expect(result).To(BeFalse())
		})
	})	
})

var _ = Describe("ActOnPrEvent", func() {
	Context("When an action should trigger coverage plugin", func() {
		It("Return true for open and synchronize actions", func() {
			event := &github.PullRequestEvent {
				Action: github.PullRequestActionOpened,
			}
			result := ActOnPrEvent(event)
			Expect(result).To(BeTrue())
	})
        It("Return true for synchronize actions", func() {
			event := &github.PullRequestEvent {
				Action : github.PullRequestActionSynchronize,
			}
			result := ActOnPrEvent(event)
			Expect(result).To(BeTrue())
		})
	})

	Context("When an action should not trigger coverage plugin", func() {
		It("Return false for closed action", func() {
			event := &github.PullRequestEvent {
				Action : github.PullRequestActionClosed,
			}
			result := ActOnPrEvent(event)
			Expect(result).To(BeFalse())
		})
		It("Return false for labled action", func() {
			event := &github.PullRequestEvent {
				Action : github.PullRequestActionLabeled,
			}
			result := ActOnPrEvent(event)
			Expect(result).To(BeFalse())
		})
		It("Return false for edited action", func() {
			event := &github.PullRequestEvent {
				Action : github.PullRequestActionEdited,
			}
			result := ActOnPrEvent(event)
			Expect(result).To(BeFalse())
		})
	})
})

