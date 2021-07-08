package metrics

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	prowConfig "k8s.io/test-infra/prow/config"
)

var memoryResources = prometheus.NewGaugeVec(prometheus.GaugeOpts{
	Name: "kubevirt_ci_job_memory_bytes",
	Help: "Memory used by CI job",
}, []string{"job_name", "org", "repo", "type", "job_cluster"})

type resourcesExporter struct{}

func (r *resourcesExporter) Start(c Config) error {
	if c.ProwConfig.PresubmitsStatic != nil {
		for orgrepo, presubmits := range c.ProwConfig.PresubmitsStatic {
			for _, presubmit := range presubmits {
				if err := r.registerJobBase(orgrepo, "presubmit", presubmit.JobBase); err != nil {
					return err
				}
			}
		}
	}
	if c.ProwConfig.PostsubmitsStatic != nil {
		for orgrepo, postsubmits := range c.ProwConfig.PostsubmitsStatic {
			for _, postsubmit := range postsubmits {
				if err := r.registerJobBase(orgrepo, "postsubmit", postsubmit.JobBase); err != nil {
					return err
				}
			}
		}
	}
	if c.ProwConfig.Periodics != nil {
		for _, periodic := range c.ProwConfig.Periodics {
			if err := r.registerJobBase("", "periodic", periodic.JobBase); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *resourcesExporter) Stop() error {
	return nil
}

func init() {
	r := &resourcesExporter{}

	exporters = append(exporters, r)

	prometheus.Register(memoryResources)
}

func (r *resourcesExporter) registerJobBase(orgrepo, kind string, jobBase prowConfig.JobBase) error {
	org, repo := extractOrgRepo(orgrepo)
	for _, container := range jobBase.Spec.Containers {
		requestedMemory := container.Resources.Requests.Memory()
		if requestedMemory != nil {
			memoryResources.WithLabelValues(jobBase.Name, org, repo, kind, jobBase.Cluster).Set(requestedMemory.ToDec().AsApproximateFloat64())
		}
	}
	return nil
}

func extractOrgRepo(orgrepo string) (string, string) {
	const sep = "/"

	if !strings.Contains(orgrepo, sep) {
		return "", ""
	}

	i := strings.Split(orgrepo, sep)
	return i[0], i[1]
}
