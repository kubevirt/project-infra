module kubevirt.io/project-infra

require (
	cloud.google.com/go v0.66.0
	cloud.google.com/go/storage v1.12.0
	github.com/Masterminds/semver v1.5.0
	github.com/avast/retry-go v3.0.0+incompatible
	github.com/bazelbuild/buildtools v0.0.0-20200922170545-10384511ce98
	github.com/bndr/gojenkins v1.1.0
	github.com/bndr/gotabulate v1.1.2
	github.com/go-git/go-git/v5 v5.3.0
	github.com/go-test/deep v1.0.7
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/go-github/v28 v28.1.1
	github.com/google/go-github/v32 v32.0.0
	github.com/jetstack/cert-manager v1.1.0
	github.com/joshdk/go-junit v0.0.0-20190428045703-ad7e11aa49ff
	github.com/lnquy/cron v1.1.1
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.13.0
	github.com/opencontainers/runc v0.1.1
	github.com/prometheus/client_golang v1.11.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.1.3
	golang.org/x/mod v0.4.0 // indirect
	golang.org/x/oauth2 v0.0.0-20200902213428-5d25da1a8d43
	google.golang.org/api v0.32.0
	k8s.io/api v0.21.1
	k8s.io/apiextensions-apiserver v0.21.1
	k8s.io/apimachinery v0.21.1
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	k8s.io/test-infra v0.0.0-20210903101950-5c7809e9c5e9
	sigs.k8s.io/boskos v0.0.0-20210730172138-093b54882439
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v14.2.0+incompatible
	github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.4.1
	golang.org/x/lint => golang.org/x/lint v0.0.0-20190409202823-959b441ac422
	k8s.io/client-go => k8s.io/client-go v0.21.1
)

go 1.16
