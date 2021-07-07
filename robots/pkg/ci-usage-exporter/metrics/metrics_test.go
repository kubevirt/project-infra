package metrics_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"regexp"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	prowConfig "k8s.io/test-infra/prow/config"

	"kubevirt.io/project-infra/robots/pkg/ci-usage-exporter/metrics"
)

const (
	host           = "127.0.0.1"
	port           = "9798"
	metricsPath    = "/metrics"
	quantityValue1 = 29000000000
	quantityValue2 = 19000000000
	quantityValue3 = 13000000000
)

var (
	subject   *metrics.Handler
	srv       http.Server
	err       error
	quantity1 = *resource.NewQuantity(quantityValue1, resource.BinarySI)
	quantity2 = *resource.NewQuantity(quantityValue2, resource.BinarySI)
	quantity3 = *resource.NewQuantity(quantityValue3, resource.BinarySI)
)

type metricExpectations struct {
	name   string
	value  float64
	labels map[string]string
}

func TestMetrics(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "metrics suite")
}

var _ = Describe("resources", func() {
	BeforeEach(func() {
		sm := http.NewServeMux()
		sm.Handle(metricsPath, promhttp.Handler())
		srv = http.Server{
			Handler: sm,
			Addr:    fmt.Sprintf("%s:%s", host, port),
		}
		l, err := net.Listen("tcp", ":"+port)
		Expect(err).NotTo(HaveOccurred())
		go func() {
			defer GinkgoRecover()
			err := srv.Serve(l)
			Expect(err).To(Equal(http.ErrServerClosed))
		}()

		subject = metrics.NewHandler()
	})

	AfterEach(func() {
		err := srv.Shutdown(context.Background())
		Expect(err).NotTo(HaveOccurred())

		err = subject.Stop()
		Expect(err).NotTo(HaveOccurred())
	})

	DescribeTable("Exposed resource metrics", func(c metrics.Config, expectations []metricExpectations) {
		err := subject.Start(c)
		Expect(err).NotTo(HaveOccurred())

		resp, err := http.Get(fmt.Sprintf("http://%s:%s%s", host, port, metricsPath))
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())

		bodyStr := string(body)

		for _, expectation := range expectations {
			for labelName, labelValue := range expectation.labels {
				valueStr := fmt.Sprintf("%g", expectation.value)
				valueStr = strings.Replace(valueStr, "+", `\+`, 1)
				r, err := regexp.Compile(fmt.Sprintf(`%s{.*%s=\"%s\".*} %s`, expectation.name, labelName, labelValue, valueStr))
				Expect(err).NotTo(HaveOccurred())

				if !r.MatchString(bodyStr) {
					Fail(fmt.Sprintf("Label %q with value %q for metric %q with value %g not found in exposed metrics %q", labelName, labelValue, expectation.name, expectation.value, bodyStr))
				}
			}
		}
	},
		Entry("basic presubmit",
			metrics.Config{
				ProwConfig: &prowConfig.Config{
					JobConfig: prowConfig.JobConfig{
						PresubmitsStatic: map[string][]prowConfig.Presubmit{
							"test-org/test-repo": {
								{
									JobBase: prowConfig.JobBase{
										Name:    "test-job",
										Cluster: "test-cluster",
										Spec: &v1.PodSpec{
											Containers: []v1.Container{
												{
													Resources: v1.ResourceRequirements{
														Requests: v1.ResourceList{
															v1.ResourceMemory: quantity1,
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			[]metricExpectations{
				{
					name:  "kubevirt_ci_job_memory_bytes",
					value: quantityValue1,
					labels: map[string]string{
						"job_name":    "test-job",
						"org":         "test-org",
						"repo":        "test-repo",
						"type":        "presubmit",
						"job_cluster": "test-cluster",
					},
				},
			},
		),
		Entry("two presubmits, same orgrepo",
			metrics.Config{
				ProwConfig: &prowConfig.Config{
					JobConfig: prowConfig.JobConfig{
						PresubmitsStatic: map[string][]prowConfig.Presubmit{
							"test-org/test-repo": {
								{
									JobBase: prowConfig.JobBase{
										Name:    "test-job1",
										Cluster: "test-cluster",
										Spec: &v1.PodSpec{
											Containers: []v1.Container{
												{
													Resources: v1.ResourceRequirements{
														Requests: v1.ResourceList{
															v1.ResourceMemory: quantity1,
														},
													},
												},
											},
										},
									},
								},
								{
									JobBase: prowConfig.JobBase{
										Name:    "test-job2",
										Cluster: "test-cluster",
										Spec: &v1.PodSpec{
											Containers: []v1.Container{
												{
													Resources: v1.ResourceRequirements{
														Requests: v1.ResourceList{
															v1.ResourceMemory: quantity2,
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			[]metricExpectations{
				{
					name:  "kubevirt_ci_job_memory_bytes",
					value: quantityValue1,
					labels: map[string]string{
						"job_name":    "test-job1",
						"org":         "test-org",
						"repo":        "test-repo",
						"type":        "presubmit",
						"job_cluster": "test-cluster",
					},
				},
				{
					name:  "kubevirt_ci_job_memory_bytes",
					value: quantityValue2,
					labels: map[string]string{
						"job_name":    "test-job2",
						"org":         "test-org",
						"repo":        "test-repo",
						"type":        "presubmit",
						"job_cluster": "test-cluster",
					},
				},
			},
		),
		Entry("two presubmits, different orgrepo",
			metrics.Config{
				ProwConfig: &prowConfig.Config{
					JobConfig: prowConfig.JobConfig{
						PresubmitsStatic: map[string][]prowConfig.Presubmit{
							"test-org1/test-repo1": {
								{
									JobBase: prowConfig.JobBase{
										Name:    "test-job1",
										Cluster: "test-cluster",
										Spec: &v1.PodSpec{
											Containers: []v1.Container{
												{
													Resources: v1.ResourceRequirements{
														Requests: v1.ResourceList{
															v1.ResourceMemory: quantity1,
														},
													},
												},
											},
										},
									},
								},
							},
							"test-org2/test-repo2": {
								{
									JobBase: prowConfig.JobBase{
										Name:    "test-job2",
										Cluster: "test-cluster",
										Spec: &v1.PodSpec{
											Containers: []v1.Container{
												{
													Resources: v1.ResourceRequirements{
														Requests: v1.ResourceList{
															v1.ResourceMemory: quantity2,
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			[]metricExpectations{
				{
					name:  "kubevirt_ci_job_memory_bytes",
					value: quantityValue1,
					labels: map[string]string{
						"job_name":    "test-job1",
						"org":         "test-org1",
						"repo":        "test-repo1",
						"type":        "presubmit",
						"job_cluster": "test-cluster",
					},
				},
				{
					name:  "kubevirt_ci_job_memory_bytes",
					value: quantityValue2,
					labels: map[string]string{
						"job_name":    "test-job2",
						"org":         "test-org2",
						"repo":        "test-repo2",
						"type":        "presubmit",
						"job_cluster": "test-cluster",
					},
				},
			},
		),
		Entry("basic postsubmit",
			metrics.Config{
				ProwConfig: &prowConfig.Config{
					JobConfig: prowConfig.JobConfig{
						PostsubmitsStatic: map[string][]prowConfig.Postsubmit{
							"test-org/test-repo": {
								{
									JobBase: prowConfig.JobBase{
										Name:    "test-job",
										Cluster: "test-cluster",
										Spec: &v1.PodSpec{
											Containers: []v1.Container{
												{
													Resources: v1.ResourceRequirements{
														Requests: v1.ResourceList{
															v1.ResourceMemory: quantity1,
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			[]metricExpectations{
				{
					name:  "kubevirt_ci_job_memory_bytes",
					value: quantityValue1,
					labels: map[string]string{
						"job_name":    "test-job",
						"org":         "test-org",
						"repo":        "test-repo",
						"type":        "postsubmit",
						"job_cluster": "test-cluster",
					},
				},
			},
		),
		Entry("two postsubmits, same orgrepo",
			metrics.Config{
				ProwConfig: &prowConfig.Config{
					JobConfig: prowConfig.JobConfig{
						PostsubmitsStatic: map[string][]prowConfig.Postsubmit{
							"test-org/test-repo": {
								{
									JobBase: prowConfig.JobBase{
										Name:    "test-job1",
										Cluster: "test-cluster",
										Spec: &v1.PodSpec{
											Containers: []v1.Container{
												{
													Resources: v1.ResourceRequirements{
														Requests: v1.ResourceList{
															v1.ResourceMemory: quantity1,
														},
													},
												},
											},
										},
									},
								},
								{
									JobBase: prowConfig.JobBase{
										Name:    "test-job2",
										Cluster: "test-cluster",
										Spec: &v1.PodSpec{
											Containers: []v1.Container{
												{
													Resources: v1.ResourceRequirements{
														Requests: v1.ResourceList{
															v1.ResourceMemory: quantity2,
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			[]metricExpectations{
				{
					name:  "kubevirt_ci_job_memory_bytes",
					value: quantityValue1,
					labels: map[string]string{
						"job_name":    "test-job1",
						"org":         "test-org",
						"repo":        "test-repo",
						"type":        "postsubmit",
						"job_cluster": "test-cluster",
					},
				},
				{
					name:  "kubevirt_ci_job_memory_bytes",
					value: quantityValue2,
					labels: map[string]string{
						"job_name":    "test-job2",
						"org":         "test-org",
						"repo":        "test-repo",
						"type":        "postsubmit",
						"job_cluster": "test-cluster",
					},
				},
			},
		),
		Entry("two postsubmits, different orgrepo",
			metrics.Config{
				ProwConfig: &prowConfig.Config{
					JobConfig: prowConfig.JobConfig{
						PostsubmitsStatic: map[string][]prowConfig.Postsubmit{
							"test-org1/test-repo1": {
								{
									JobBase: prowConfig.JobBase{
										Name:    "test-job1",
										Cluster: "test-cluster",
										Spec: &v1.PodSpec{
											Containers: []v1.Container{
												{
													Resources: v1.ResourceRequirements{
														Requests: v1.ResourceList{
															v1.ResourceMemory: quantity1,
														},
													},
												},
											},
										},
									},
								},
							},
							"test-org2/test-repo2": {
								{
									JobBase: prowConfig.JobBase{
										Name:    "test-job2",
										Cluster: "test-cluster",
										Spec: &v1.PodSpec{
											Containers: []v1.Container{
												{
													Resources: v1.ResourceRequirements{
														Requests: v1.ResourceList{
															v1.ResourceMemory: quantity2,
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			[]metricExpectations{
				{
					name:  "kubevirt_ci_job_memory_bytes",
					value: quantityValue1,
					labels: map[string]string{
						"job_name":    "test-job1",
						"org":         "test-org1",
						"repo":        "test-repo1",
						"type":        "postsubmit",
						"job_cluster": "test-cluster",
					},
				},
				{
					name:  "kubevirt_ci_job_memory_bytes",
					value: quantityValue2,
					labels: map[string]string{
						"job_name":    "test-job2",
						"org":         "test-org2",
						"repo":        "test-repo2",
						"type":        "postsubmit",
						"job_cluster": "test-cluster",
					},
				},
			},
		),
		Entry("basic periodic",
			metrics.Config{
				ProwConfig: &prowConfig.Config{
					JobConfig: prowConfig.JobConfig{
						Periodics: []prowConfig.Periodic{
							{
								JobBase: prowConfig.JobBase{
									Name:    "test-job",
									Cluster: "test-cluster",
									Spec: &v1.PodSpec{
										Containers: []v1.Container{
											{
												Resources: v1.ResourceRequirements{
													Requests: v1.ResourceList{
														v1.ResourceMemory: quantity1,
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			[]metricExpectations{
				{
					name:  "kubevirt_ci_job_memory_bytes",
					value: quantityValue1,
					labels: map[string]string{
						"job_name":    "test-job",
						"type":        "periodic",
						"job_cluster": "test-cluster",
					},
				},
			},
		),
		Entry("two periodics",
			metrics.Config{
				ProwConfig: &prowConfig.Config{
					JobConfig: prowConfig.JobConfig{
						Periodics: []prowConfig.Periodic{
							{
								JobBase: prowConfig.JobBase{
									Name:    "test-job1",
									Cluster: "test-cluster",
									Spec: &v1.PodSpec{
										Containers: []v1.Container{
											{
												Resources: v1.ResourceRequirements{
													Requests: v1.ResourceList{
														v1.ResourceMemory: quantity1,
													},
												},
											},
										},
									},
								},
							},
							{
								JobBase: prowConfig.JobBase{
									Name:    "test-job2",
									Cluster: "test-cluster",
									Spec: &v1.PodSpec{
										Containers: []v1.Container{
											{
												Resources: v1.ResourceRequirements{
													Requests: v1.ResourceList{
														v1.ResourceMemory: quantity2,
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			[]metricExpectations{
				{
					name:  "kubevirt_ci_job_memory_bytes",
					value: quantityValue1,
					labels: map[string]string{
						"job_name":    "test-job1",
						"type":        "periodic",
						"job_cluster": "test-cluster",
					},
				},
				{
					name:  "kubevirt_ci_job_memory_bytes",
					value: quantityValue2,
					labels: map[string]string{
						"job_name":    "test-job2",
						"type":        "periodic",
						"job_cluster": "test-cluster",
					},
				},
			},
		),
		Entry("presubmit, postsubmit, and periodic",
			metrics.Config{
				ProwConfig: &prowConfig.Config{
					JobConfig: prowConfig.JobConfig{
						PresubmitsStatic: map[string][]prowConfig.Presubmit{
							"test-org1/test-repo1": {
								{
									JobBase: prowConfig.JobBase{
										Name:    "test-job1",
										Cluster: "test-cluster1",
										Spec: &v1.PodSpec{
											Containers: []v1.Container{
												{
													Resources: v1.ResourceRequirements{
														Requests: v1.ResourceList{
															v1.ResourceMemory: quantity1,
														},
													},
												},
											},
										},
									},
								},
							},
						},
						PostsubmitsStatic: map[string][]prowConfig.Postsubmit{
							"test-org2/test-repo2": {
								{
									JobBase: prowConfig.JobBase{
										Name:    "test-job2",
										Cluster: "test-cluster2",
										Spec: &v1.PodSpec{
											Containers: []v1.Container{
												{
													Resources: v1.ResourceRequirements{
														Requests: v1.ResourceList{
															v1.ResourceMemory: quantity2,
														},
													},
												},
											},
										},
									},
								},
							},
						},
						Periodics: []prowConfig.Periodic{
							{
								JobBase: prowConfig.JobBase{
									Name:    "test-job3",
									Cluster: "test-cluster3",
									Spec: &v1.PodSpec{
										Containers: []v1.Container{
											{
												Resources: v1.ResourceRequirements{
													Requests: v1.ResourceList{
														v1.ResourceMemory: quantity3,
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			[]metricExpectations{
				{
					name:  "kubevirt_ci_job_memory_bytes",
					value: quantityValue1,
					labels: map[string]string{
						"job_name":    "test-job1",
						"org":         "test-org1",
						"repo":        "test-repo1",
						"type":        "postsubmit",
						"job_cluster": "test-cluster1",
					},
				},
				{
					name:  "kubevirt_ci_job_memory_bytes",
					value: quantityValue2,
					labels: map[string]string{
						"job_name":    "test-job2",
						"org":         "test-org2",
						"repo":        "test-repo2",
						"type":        "postsubmit",
						"job_cluster": "test-cluster2",
					},
				},
				{
					name:  "kubevirt_ci_job_memory_bytes",
					value: quantityValue3,
					labels: map[string]string{
						"job_name":    "test-job3",
						"type":        "periodic",
						"job_cluster": "test-cluster3",
					},
				},
			},
		),
	)
})
