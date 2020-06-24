module kubevirt.io/project-infra

require (
	cloud.google.com/go v0.47.0
	github.com/bazelbuild/buildtools v0.0.0-20190917191645-69366ca98f89
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/go-github/v28 v28.0.0
	github.com/joshdk/go-junit v0.0.0-20190428045703-ad7e11aa49ff
	github.com/onsi/ginkgo v1.11.0
	github.com/onsi/gomega v1.8.1
	github.com/sirupsen/logrus v1.6.0
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	google.golang.org/api v0.15.0
	k8s.io/apimachinery v0.17.3
	k8s.io/test-infra v0.0.0-20200519204219-34a27f5e6d4e
	sigs.k8s.io/yaml v1.2.0
)

replace (
	cloud.google.com/go => cloud.google.com/go v0.44.3
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v12.2.0+incompatible
	golang.org/x/lint => golang.org/x/lint v0.0.0-20190409202823-959b441ac422
	k8s.io/api => k8s.io/api v0.17.3
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.3
	k8s.io/client-go => k8s.io/client-go v0.17.3
	k8s.io/code-generator => k8s.io/code-generator v0.17.3
)

go 1.13
