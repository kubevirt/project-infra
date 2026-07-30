package handler

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/prow/pkg/config"
	"sigs.k8s.io/prow/pkg/github"
)

func presubmit(name string, phase string) config.Presubmit {
	ps := config.Presubmit{
		JobBase: config.JobBase{Name: name},
	}
	if phase != "" {
		ps.JobBase.Annotations = map[string]string{PhaseAnnotationKey: phase}
	}
	return ps
}

func presubmitWithContext(name, ctx, phase string) config.Presubmit {
	ps := presubmit(name, phase)
	ps.Context = ctx
	return ps
}

var _ = Describe("jobContext", func() {
	It("returns Name when Context is empty", func() {
		job := presubmit("pull-kubevirt-build", "0")
		Expect(jobContext(job)).To(Equal("pull-kubevirt-build"))
	})

	It("returns Context when set", func() {
		job := presubmitWithContext("pull-kubevirt-build", "custom-context", "0")
		Expect(jobContext(job)).To(Equal("custom-context"))
	})
})

var _ = Describe("groupJobsByPhase", func() {
	It("groups jobs by their phase annotation", func() {
		presubmits := []config.Presubmit{
			presubmit("build", "0"),
			presubmit("generate", "0"),
			presubmit("e2e", "1"),
		}
		phases, sorted := groupJobsByPhase(presubmits)
		Expect(sorted).To(Equal([]int{0, 1}))
		Expect(phases[0]).To(HaveLen(2))
		Expect(phases[1]).To(HaveLen(1))
		Expect(phases[1][0].Name).To(Equal("e2e"))
	})

	It("skips jobs without phase annotation", func() {
		presubmits := []config.Presubmit{
			presubmit("build", "0"),
			presubmit("merge-job", ""),
		}
		phases, sorted := groupJobsByPhase(presubmits)
		Expect(sorted).To(Equal([]int{0}))
		Expect(phases[0]).To(HaveLen(1))
	})

	It("skips jobs with non-integer phase annotation", func() {
		presubmits := []config.Presubmit{
			presubmit("build", "0"),
			{
				JobBase: config.JobBase{
					Name:        "bad-phase",
					Annotations: map[string]string{PhaseAnnotationKey: "notanumber"},
				},
			},
		}
		phases, sorted := groupJobsByPhase(presubmits)
		Expect(sorted).To(Equal([]int{0}))
		Expect(phases[0]).To(HaveLen(1))
	})

	It("returns empty maps when no jobs have phase annotations", func() {
		presubmits := []config.Presubmit{
			presubmit("merge-a", ""),
			presubmit("merge-b", ""),
		}
		phases, sorted := groupJobsByPhase(presubmits)
		Expect(sorted).To(BeEmpty())
		Expect(phases).To(BeEmpty())
	})

	It("handles three phases", func() {
		presubmits := []config.Presubmit{
			presubmit("build", "0"),
			presubmit("unit", "1"),
			presubmit("e2e", "2"),
		}
		phases, sorted := groupJobsByPhase(presubmits)
		Expect(sorted).To(Equal([]int{0, 1, 2}))
		Expect(phases[0]).To(HaveLen(1))
		Expect(phases[1]).To(HaveLen(1))
		Expect(phases[2]).To(HaveLen(1))
	})
})

var _ = Describe("allJobsSucceeded", func() {
	It("returns true when all jobs have success status", func() {
		jobs := []config.Presubmit{
			presubmit("build", "0"),
			presubmit("generate", "0"),
		}
		statusMap := map[string]string{
			"build":    github.StatusSuccess,
			"generate": github.StatusSuccess,
		}
		Expect(allJobsSucceeded(jobs, statusMap)).To(BeTrue())
	})

	It("returns false when a job is pending", func() {
		jobs := []config.Presubmit{
			presubmit("build", "0"),
			presubmit("generate", "0"),
		}
		statusMap := map[string]string{
			"build":    github.StatusSuccess,
			"generate": github.StatusPending,
		}
		Expect(allJobsSucceeded(jobs, statusMap)).To(BeFalse())
	})

	It("returns false when a job is failed", func() {
		jobs := []config.Presubmit{
			presubmit("build", "0"),
		}
		statusMap := map[string]string{
			"build": github.StatusFailure,
		}
		Expect(allJobsSucceeded(jobs, statusMap)).To(BeFalse())
	})

	It("returns false when a job has no status entry", func() {
		jobs := []config.Presubmit{
			presubmit("build", "0"),
			presubmit("generate", "0"),
		}
		statusMap := map[string]string{
			"build": github.StatusSuccess,
		}
		Expect(allJobsSucceeded(jobs, statusMap)).To(BeFalse())
	})

	It("uses Context field when set", func() {
		jobs := []config.Presubmit{
			presubmitWithContext("build", "ci/build", "0"),
		}
		statusMap := map[string]string{
			"ci/build": github.StatusSuccess,
		}
		Expect(allJobsSucceeded(jobs, statusMap)).To(BeTrue())
	})

	It("returns true for empty job list", func() {
		Expect(allJobsSucceeded(nil, map[string]string{})).To(BeTrue())
	})
})

var _ = Describe("hasUntriggeredJobs", func() {
	It("returns true when a job has no status entry", func() {
		jobs := []config.Presubmit{
			presubmit("e2e", "1"),
		}
		statusMap := map[string]string{}
		Expect(hasUntriggeredJobs(jobs, statusMap)).To(BeTrue())
	})

	It("returns false when all jobs have a status entry", func() {
		jobs := []config.Presubmit{
			presubmit("e2e", "1"),
		}
		statusMap := map[string]string{
			"e2e": github.StatusPending,
		}
		Expect(hasUntriggeredJobs(jobs, statusMap)).To(BeFalse())
	})

	It("returns true when some jobs are triggered and some are not", func() {
		jobs := []config.Presubmit{
			presubmit("e2e-a", "1"),
			presubmit("e2e-b", "1"),
		}
		statusMap := map[string]string{
			"e2e-a": github.StatusPending,
		}
		Expect(hasUntriggeredJobs(jobs, statusMap)).To(BeTrue())
	})

	It("returns false for empty job list", func() {
		Expect(hasUntriggeredJobs(nil, map[string]string{})).To(BeFalse())
	})
})

var _ = Describe("findNextPhaseToTrigger", func() {
	It("returns phase 1 when phase 0 all succeeded and phase 1 untriggered", func() {
		phaseJobs := map[int][]config.Presubmit{
			0: {presubmit("build", "0"), presubmit("generate", "0")},
			1: {presubmit("e2e", "1")},
		}
		statusMap := map[string]string{
			"build":    github.StatusSuccess,
			"generate": github.StatusSuccess,
		}
		completed, next := findNextPhaseToTrigger(phaseJobs, []int{0, 1}, statusMap)
		Expect(completed).To(Equal(0))
		Expect(next).To(Equal(1))
	})

	It("returns -1,-1 when phase 0 is not fully succeeded", func() {
		phaseJobs := map[int][]config.Presubmit{
			0: {presubmit("build", "0"), presubmit("generate", "0")},
			1: {presubmit("e2e", "1")},
		}
		statusMap := map[string]string{
			"build":    github.StatusSuccess,
			"generate": github.StatusPending,
		}
		completed, next := findNextPhaseToTrigger(phaseJobs, []int{0, 1}, statusMap)
		Expect(completed).To(Equal(-1))
		Expect(next).To(Equal(-1))
	})

	It("returns -1,-1 when next phase is already triggered", func() {
		phaseJobs := map[int][]config.Presubmit{
			0: {presubmit("build", "0")},
			1: {presubmit("e2e", "1")},
		}
		statusMap := map[string]string{
			"build": github.StatusSuccess,
			"e2e":   github.StatusPending,
		}
		completed, next := findNextPhaseToTrigger(phaseJobs, []int{0, 1}, statusMap)
		Expect(completed).To(Equal(-1))
		Expect(next).To(Equal(-1))
	})

	It("returns -1,-1 when all phases completed", func() {
		phaseJobs := map[int][]config.Presubmit{
			0: {presubmit("build", "0")},
			1: {presubmit("e2e", "1")},
		}
		statusMap := map[string]string{
			"build": github.StatusSuccess,
			"e2e":   github.StatusSuccess,
		}
		completed, next := findNextPhaseToTrigger(phaseJobs, []int{0, 1}, statusMap)
		Expect(completed).To(Equal(-1))
		Expect(next).To(Equal(-1))
	})

	It("triggers phase 2 when phases 0 and 1 succeeded", func() {
		phaseJobs := map[int][]config.Presubmit{
			0: {presubmit("build", "0")},
			1: {presubmit("unit", "1")},
			2: {presubmit("e2e", "2")},
		}
		statusMap := map[string]string{
			"build": github.StatusSuccess,
			"unit":  github.StatusSuccess,
		}
		completed, next := findNextPhaseToTrigger(phaseJobs, []int{0, 1, 2}, statusMap)
		Expect(completed).To(Equal(1))
		Expect(next).To(Equal(2))
	})

	It("triggers phase 1 even when phase 2 exists untriggered", func() {
		phaseJobs := map[int][]config.Presubmit{
			0: {presubmit("build", "0")},
			1: {presubmit("unit", "1")},
			2: {presubmit("e2e", "2")},
		}
		statusMap := map[string]string{
			"build": github.StatusSuccess,
		}
		completed, next := findNextPhaseToTrigger(phaseJobs, []int{0, 1, 2}, statusMap)
		Expect(completed).To(Equal(0))
		Expect(next).To(Equal(1))
	})

	It("returns -1,-1 for empty phases", func() {
		completed, next := findNextPhaseToTrigger(map[int][]config.Presubmit{}, []int{}, map[string]string{})
		Expect(completed).To(Equal(-1))
		Expect(next).To(Equal(-1))
	})
})

var _ = Describe("presubmitCache", func() {
	It("returns miss on empty cache", func() {
		c := presubmitCache{ttl: 5}
		_, _, ok := c.get()
		Expect(ok).To(BeFalse())
	})

	It("returns hit after set", func() {
		c := presubmitCache{ttl: 5 * 60}
		presubmits := []config.Presubmit{
			presubmit("build", "0"),
			presubmit("merge-job", ""),
		}
		c.set(presubmits)
		ps, names, ok := c.get()
		Expect(ok).To(BeTrue())
		Expect(ps).To(HaveLen(2))
		Expect(names).To(HaveKey("build"))
		Expect(names).NotTo(HaveKey("merge-job"))
	})

	It("builds phasedNames only for annotated jobs", func() {
		c := presubmitCache{ttl: 5 * 60}
		c.set([]config.Presubmit{
			presubmit("phase-0-job", "0"),
			presubmit("phase-1-job", "1"),
			presubmit("unphased-job", ""),
		})
		_, names, ok := c.get()
		Expect(ok).To(BeTrue())
		Expect(names).To(HaveLen(2))
		Expect(names["phase-0-job"]).To(BeTrue())
		Expect(names["phase-1-job"]).To(BeTrue())
	})

	It("uses Context field for phasedNames key when set", func() {
		c := presubmitCache{ttl: 5 * 60}
		c.set([]config.Presubmit{
			presubmitWithContext("build", "ci/build", "0"),
		})
		_, names, ok := c.get()
		Expect(ok).To(BeTrue())
		Expect(names).To(HaveKey("ci/build"))
		Expect(names).NotTo(HaveKey("build"))
	})
})
