# Coverage External Plugin - Implementation Design

## Overview

This document outlines the design for a Prow external plugin that automatically runs Go unit test coverage on pull requests containing Go code changes.

**Issue**: https://github.com/kubevirt/project-infra/issues/4064

**Goal**: Create an external plugin that:
- Detects when Go files are changed in a PR
- Automatically generates and submits a coverage ProwJob
- Makes coverage artifacts browsable via Prow's Spyglass
- Posts GitHub status with link to coverage report

## Architecture

Based on the existing `rehearse` plugin pattern.

### Directory Structure

```
external-plugins/coverage/
├── plugin/
│   ├── main.go                    # Entry point, server setup
│   ├── server/
│   │   └── eventsserver.go        # HTTP webhook handler
│   ├── handler/
│   │   └── handler.go             # Event processing & job generation
│   ├── coverage_suite_test.go     # Ginkgo test suite
│   └── coverage_test.go           # Unit tests
├── Containerfile                  # Docker image build
└── README.md                      # Plugin documentation
```

## Component Design

### 1. Main Entry Point (`main.go`)

**Responsibilities:**
- Parse command-line flags
- Initialize Kubernetes client (for ProwJob API)
- Initialize GitHub client
- Start HTTP server for webhooks
- Handle graceful shutdown

**Command-line Flags:**
```go
--port=9901                                    # HTTP server port
--endpoint=/                                   # Webhook path
--hmac-secret-file=/etc/webhook/hmac          # GitHub webhook secret
--github-token-path=/etc/github/oauth         # GitHub API token
--github-endpoint=http://ghproxy              # GitHub API endpoint
--jobs-namespace=kubevirt-prow-jobs           # K8s namespace for ProwJobs
--kubeconfig=                                 # Optional kubeconfig path
--dry-run=false                               # If true, log job spec without submitting
--coverage-dirs=./external-plugins/...,./releng/...,./robots/...  # Directories to include in coverage
```

**Pseudocode:**
```go
func main() {
    // 1. Parse flags
    // 2. Create K8s REST client for prowjobs
    // 3. Create GitHub client
    // 4. Create event channel
    // 5. Create handler with clients
    // 6. Create HTTP server with handler
    // 7. Start server
    // 8. Wait for shutdown signal
}
```

### 2. HTTP Server (`server/eventsserver.go`)

**Responsibilities:**
- Receive GitHub webhook events
- Validate HMAC signatures
- Route events to handler asynchronously
- Return quick HTTP response

**Event Types to Handle:**
- `pull_request` with actions: `opened`, `synchronize`

**Pseudocode:**
```go
func (s *GitHubEventsServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // 1. Validate HMAC signature
    // 2. Parse webhook payload
    // 3. Extract event type and GUID
    // 4. Send to event channel (non-blocking)
    // 5. Return 200 OK immediately
}
```

### 3. Event Handler (`handler/handler.go`)

**Responsibilities:**
- Process pull_request events
- Detect Go file changes
- Generate coverage ProwJob
- Submit job to Kubernetes
- Post GitHub status

**Core Functions:**

#### `Handle(event GitHubEvent)`
Main event processing loop.

```go
func (h *Handler) Handle(event GitHubEvent) {
    switch event.Type {
    case "pull_request":
        h.handlePullRequestEvent(event)
    }
}
```

#### `handlePullRequestEvent(event)`
Processes PR events.

```go
func (h *Handler) handlePullRequestEvent(event) {
    // 1. Parse pull_request payload
    pr := event.Payload.PullRequest

    // 2. Only process "opened" or "synchronize" actions
    if action != "opened" && action != "synchronize" {
        return
    }

    // 3. Get list of changed files from GitHub API
    files := h.githubClient.GetPullRequestChanges(pr.Org, pr.Repo, pr.Number)

    // 4. Check if Go files changed
    if !hasGoFileChanges(files) {
        log.Info("No Go files changed, skipping coverage")
        return
    }

    // 5. Generate coverage job
    job := h.generateCoverageJob(pr)

    // 6. Submit job to Kubernetes
    if !h.dryRun {
        h.prowJobClient.Create(job)
        log.Infof("Created coverage job: %s", job.Name)
    } else {
        log.Infof("Dry run - would create job: %s", job.Name)
    }

    // 7. Post GitHub status (pending)
    h.postGitHubStatus(pr, "pending", job)
}
```

#### `hasGoFileChanges(files []github.PullRequestChange) bool`
Determines if coverage is applicable.

```go
func hasGoFileChanges(files []github.PullRequestChange) bool {
    for _, file := range files {
        // Check if file is in coverage directories
        if !isInCoverageDir(file.Filename) {
            continue
        }

        // Check if file is a Go file
        if strings.HasSuffix(file.Filename, ".go") ||
           file.Filename == "go.mod" ||
           file.Filename == "go.sum" {
            return true
        }
    }
    return false
}

func isInCoverageDir(filename string) bool {
    // Check if file is under external-plugins/, releng/, or robots/
    return strings.HasPrefix(filename, "external-plugins/") ||
           strings.HasPrefix(filename, "releng/") ||
           strings.HasPrefix(filename, "robots/")
}
```

#### `generateCoverageJob(pr *github.PullRequest) *prowapi.ProwJob`
Creates the ProwJob spec.

```go
func (h *Handler) generateCoverageJob(pr *github.PullRequest) *prowapi.ProwJob {
    // Based on existing pull-project-infra-coverage job
    job := &prowapi.ProwJob{
        TypeMeta: metav1.TypeMeta{
            APIVersion: "prow.k8s.io/v1",
            Kind:       "ProwJob",
        },
        ObjectMeta: metav1.ObjectMeta{
            Name:      fmt.Sprintf("coverage-%s-%d", pr.Base.Repo.Name, pr.Number),
            Namespace: h.jobsNamespace,
            Labels: map[string]string{
                "coverage-plugin": "true",
                "prow.k8s.io/type": "presubmit",
                "prow.k8s.io/refs.org": pr.Base.Repo.Owner.Login,
                "prow.k8s.io/refs.repo": pr.Base.Repo.Name,
                "prow.k8s.io/refs.pull": strconv.Itoa(pr.Number),
            },
        },
        Spec: prowapi.ProwJobSpec{
            Type: prowapi.PresubmitJob,
            Cluster: "kubevirt-prow-control-plane",
            Job: "coverage-auto",
            Refs: &prowapi.Refs{
                Org:      pr.Base.Repo.Owner.Login,
                Repo:     pr.Base.Repo.Name,
                BaseRef:  pr.Base.Ref,
                BaseSHA:  pr.Base.SHA,
                Pulls: []prowapi.Pull{
                    {
                        Number: pr.Number,
                        Author: pr.User.Login,
                        SHA:    pr.Head.SHA,
                    },
                },
            },
            PodSpec: &v1.PodSpec{
                Containers: []v1.Container{
                    {
                        Image: "quay.io/kubevirtci/golang:v20251218-e7a7fc9",
                        Command: []string{
                            "/usr/local/bin/runner.sh",
                            "/bin/sh",
                            "-ce",
                        },
                        Args: []string{
                            "make coverage",
                        },
                        Env: []v1.EnvVar{
                            {Name: "GIMME_GO_VERSION", Value: "1.25.1"},
                        },
                        Resources: v1.ResourceRequirements{
                            Requests: v1.ResourceList{
                                v1.ResourceMemory: resource.MustParse("4Gi"),
                            },
                            Limits: v1.ResourceList{
                                v1.ResourceMemory: resource.MustParse("4Gi"),
                            },
                        },
                    },
                },
            },
        },
    }

    return job
}
```

#### `postGitHubStatus(pr, state, job)`
Posts status check to PR.

```go
func (h *Handler) postGitHubStatus(pr *github.PullRequest, state string, job *prowapi.ProwJob) {
    status := github.Status{
        State:       state,  // "pending", "success", "failure"
        Context:     "coverage-auto",
        Description: "Automated Go unit test coverage",
        TargetURL:   buildSpyglassURL(job),
    }

    h.githubClient.CreateStatus(pr.Org, pr.Repo, pr.Head.SHA, status)
}

func buildSpyglassURL(job *prowapi.ProwJob) string {
    // https://prow.ci.kubevirt.io/view/gs/kubevirt-prow/pr-logs/pull/{org}_{repo}/{pr}/coverage-auto/{buildID}/
    return fmt.Sprintf("https://prow.ci.kubevirt.io/view/gs/kubevirt-prow/pr-logs/pull/%s_%s/%d/%s/%s/",
        job.Spec.Refs.Org,
        job.Spec.Refs.Repo,
        job.Spec.Refs.Pulls[0].Number,
        job.Spec.Job,
        job.Status.BuildID,
    )
}
```

## Changes to Existing Files

### 1. Makefile

Update the `coverage` target to produce Spyglass-compatible artifacts:

```makefile
coverage:
	if ! command -V covreport; then go install github.com/cancue/covreport@latest; fi
	go test ${WHAT_COVERAGE} -coverprofile=/tmp/coverage.out
	mkdir -p ${ARTIFACTS}
	cp /tmp/coverage.out ${ARTIFACTS}/filtered.cov          # ← ADD: For Spyglass coverage lens
	covreport -i /tmp/coverage.out -o ${ARTIFACTS}/filtered.html  # ← CHANGE: Rename output
```

**Why this change?**
- Prow's Spyglass coverage lens looks for `artifacts/filtered.cov` and `artifacts/filtered.html`
- Currently produces `coverage.html` which doesn't match

### 2. Prow Configuration

#### `github/ci/prow-deploy/kustom/base/configs/current/plugins/plugins.yaml`

Add coverage plugin to kubevirt/project-infra:

```yaml
kubevirt/project-infra:
  - name: coverage
    endpoint: http://prow-coverage:9901
    events:
      - pull_request
```

#### Create Kubernetes Manifests

**`github/ci/prow-deploy/kustom/base/manifests/local/prow-coverage-rbac.yaml`**
```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: prow-coverage
  namespace: kubevirt-prow-jobs
---
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: prow-coverage
rules:
  - apiGroups:
      - prow.k8s.io
    resources:
      - prowjobs
    verbs:
      - create
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: prow-coverage
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: prow-coverage
subjects:
  - kind: ServiceAccount
    name: prow-coverage
    namespace: kubevirt-prow-jobs
```

**`github/ci/prow-deploy/kustom/base/manifests/local/prow-coverage-service.yaml`**
```yaml
apiVersion: v1
kind: Service
metadata:
  name: prow-coverage
  namespace: kubevirt-prow-jobs
spec:
  selector:
    app: prow-coverage
  ports:
    - name: http
      port: 9901
      targetPort: 9901
  type: ClusterIP
```

**`github/ci/prow-deploy/kustom/base/manifests/local/prow-coverage-deployment.yaml`**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: prow-coverage
  namespace: kubevirt-prow-jobs
  labels:
    app: prow-coverage
spec:
  replicas: 1
  selector:
    matchLabels:
      app: prow-coverage
  template:
    metadata:
      labels:
        app: prow-coverage
    spec:
      serviceAccountName: prow-coverage
      containers:
        - name: prow-coverage
          image: quay.io/kubevirtci/coverage:latest
          args:
            - --dry-run=false
            - --github-token-path=/etc/github/oauth
            - --github-endpoint=http://ghproxy
            - --github-endpoint=https://api.github.com
            - --hmac-secret-file=/etc/webhook/hmac
            - --jobs-namespace=kubevirt-prow-jobs
            - --port=9901
            - --coverage-dirs=./external-plugins/...,./releng/...,./robots/...
          ports:
            - name: http
              containerPort: 9901
          volumeMounts:
            - name: hmac
              mountPath: /etc/webhook
              readOnly: true
            - name: oauth
              mountPath: /etc/github
              readOnly: true
          resources:
            requests:
              memory: "256Mi"
              cpu: "100m"
      volumes:
        - name: hmac
          secret:
            secretName: hmac-token
        - name: oauth
          secret:
            secretName: oauth-token
```

### 3. Container Image

**`external-plugins/coverage/Containerfile`**
```dockerfile
FROM golang:1.25.1 as builder

WORKDIR /workspace
COPY . .

# Build the coverage plugin
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /coverage ./external-plugins/coverage/plugin

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /coverage .
USER 65532:65532

ENTRYPOINT ["/coverage"]
```

**Add to `github/ci/prow-deploy/files/jobs/kubevirt/project-infra/project-infra-presubmits.yaml`**
```yaml
- name: build-coverage-image
  always_run: false
  run_if_changed: "^(external-plugins|images)/coverage/.*|pkg/.*|go.mod|go.sum"
  decorate: true
  labels:
    preset-podman-in-container-enabled: "true"
    preset-docker-mirror-proxy: "true"
    preset-kubevirtci-quay-credential: "true"
    preset-project-infra-image-build-linux: "true"
  cluster: kubevirt-prow-control-plane
  spec:
    containers:
      - image: quay.io/kubevirtci/bootstrap:v20251218-e7a7fc9
        command:
          - "/usr/local/bin/runner.sh"
          - "/bin/bash"
          - "-ce"
          - "cd images && ./publish_image.sh -b -e coverage quay.io kubevirtci"
        securityContext:
          privileged: true
        resources:
          requests:
            memory: "1Gi"
          limits:
            memory: "1Gi"
```

## Testing Strategy

### Unit Tests
- Test `hasGoFileChanges()` with various file lists
- Test `generateCoverageJob()` produces valid ProwJob spec
- Mock GitHub and Kubernetes clients

### Integration Tests
1. **Local Testing**: Run plugin locally with dry-run mode
2. **PR Testing**: Create test PR with Go changes, verify job is created
3. **Artifact Testing**: Verify `filtered.cov` and `filtered.html` are produced
4. **Spyglass Testing**: Verify coverage is browsable in Prow UI

## Deployment Plan

1. **Phase 1**: Update Makefile (low risk, immediate benefit)
2. **Phase 2**: Build plugin code
3. **Phase 3**: Add container image build
4. **Phase 4**: Deploy to testing environment
5. **Phase 5**: Enable on kubevirt/project-infra
6. **Phase 6**: Monitor and iterate

## Success Criteria

- [ ] Plugin runs on every PR with Go file changes
- [ ] Coverage ProwJob is created automatically
- [ ] Artifacts appear in GCS with correct naming
- [ ] Spyglass coverage lens displays coverage
- [ ] GitHub status check appears on PR with link
- [ ] No coverage job runs when only non-Go files change
- [ ] Existing `pull-project-infra-coverage` job can be removed

## Open Questions

1. Should the plugin also respond to `/coverage` comment commands?
2. Should we add coverage percentage thresholds?
3. Should coverage be required for merge or just informational?
4. Should we notify users when coverage decreases?

## References

- Issue: https://github.com/kubevirt/project-infra/issues/4064
- Rehearse plugin: `/home/dywhite/project-infra/external-plugins/rehearse/`
- Existing coverage job: `/home/dywhite/project-infra/github/ci/prow-deploy/files/jobs/kubevirt/project-infra/project-infra-presubmits.yaml:146-168`
- Prow Spyglass docs: https://docs.prow.k8s.io/docs/spyglass/
