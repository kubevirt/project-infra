package server

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"k8s.io/client-go/testing"
	"sigs.k8s.io/prow/pkg/client/clientset/versioned/typed/prowjobs/v1/fake"
	"sigs.k8s.io/prow/pkg/github"
	"sigs.k8s.io/prow/pkg/github/fakegithub"

	"kubevirt.io/project-infra/external-plugins/coverage/plugin/handler"
)

var _ = Describe("GitHubEventsServer", func() {
	var (
		eventsServer *GitHubEventsServer
		hmacSecret   []byte
		recorder     *httptest.ResponseRecorder
	)

	BeforeEach(func() {
		hmacSecret = []byte("test-hmac-secret")
		logger := logrus.New()
		logger.SetOutput(io.Discard)

		fakeProwClient := &fake.FakeProwV1{
			Fake: &testing.Fake{},
		}
		fakeGithubClient := fakegithub.NewFakeClient()

		cfg := &handler.Config{
			Defaults: handler.JobConfig{
				Namespace: "test-namespace",
				Image:     "test-image:latest",
				Cluster:   "test-cluster",
				GCS:       handler.GCSConfig{Bucket: "test-bucket", CredentialsSecret: "gcs"},
				UtilityImages: handler.UtilityImagesConfig{
					CloneRefs:  "clonerefs:latest",
					InitUpload: "initupload:latest",
					Entrypoint: "entrypoint:latest",
					Sidecar:    "sidecar:latest",
				},
			},
			Repos: map[string]handler.JobConfig{
				"kubevirt/project-infra": {
					TestPackages: "./...",
				},
			},
		}
		eventsHandler := handler.NewGitHubEventsHandler(
			logger, fakeProwClient.ProwJobs(cfg.Defaults.Namespace), fakeGithubClient, cfg, true,
		)
		eventsServer = NewGitHubEventsServer(
			func() []byte { return hmacSecret },
			eventsHandler,
		)
		recorder = httptest.NewRecorder()
	})

	Context("When a valid webhook is received", func() {
		It("Should return 200 with success message", func() {
			payload := []byte(`{"action": "opened"}`)
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(payload))
			req.Header.Set("X-GitHub-Event", "pull_request")
			req.Header.Set("X-GitHub-Delivery", "test-guid")
			req.Header.Set("X-Hub-Signature", github.PayloadSignature(payload, hmacSecret))
			req.Header.Set("Content-Type", "application/json")

			eventsServer.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(http.StatusOK))
			Expect(recorder.Body.String()).To(Equal("Event received. Have a nice day."))
		})
	})

	DescribeTable("Should reject invalid webhook requests",
		func(method, signature string, setSignature bool) {
			payload := []byte(`{"action": "opened"}`)
			req := httptest.NewRequest(method, "/", bytes.NewBuffer(payload))
			req.Header.Set("X-GitHub-Event", "pull_request")
			req.Header.Set("X-GitHub-Delivery", "test-guid-456")
			req.Header.Set("Content-Type", "application/json")
			if setSignature {
				req.Header.Set("X-Hub-Signature", signature)
			}

			eventsServer.ServeHTTP(recorder, req)

			Expect(recorder.Code).NotTo(Equal(http.StatusOK))
		},
		Entry("wrong HMAC key", http.MethodPost, github.PayloadSignature([]byte(`{"action": "opened"}`), []byte("wrong-secret")), true),
		Entry("malformed signature", http.MethodPost, "sha1=invalidsignature", true),
		Entry("missing signature header", http.MethodPost, "", false),
		Entry("non-POST HTTP method", http.MethodGet, github.PayloadSignature([]byte(`{"action": "opened"}`), []byte("test-hmac-secret")), true),
	)

})
