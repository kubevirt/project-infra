module kubevirt.io/project-infra

require (
	cloud.google.com/go v0.66.0
	cloud.google.com/go/storage v1.12.0
	github.com/Masterminds/semver v1.5.0
	github.com/bazelbuild/buildtools v0.0.0-20190917191645-69366ca98f89
	github.com/google/go-cmp v0.5.4 // indirect
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/go-github/v28 v28.1.1
	github.com/google/go-github/v32 v32.0.0
	github.com/jetstack/cert-manager v1.1.0
	github.com/joshdk/go-junit v0.0.0-20190428045703-ad7e11aa49ff
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/sirupsen/logrus v1.6.0
	golang.org/x/mod v0.4.0 // indirect
	golang.org/x/oauth2 v0.0.0-20200902213428-5d25da1a8d43
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a // indirect
	golang.org/x/tools v0.0.0-20201230224404-63754364767c // indirect
	google.golang.org/api v0.32.0
	honnef.co/go/tools v0.1.0 // indirect
	k8s.io/api v0.19.3
	k8s.io/apiextensions-apiserver v0.19.2
	k8s.io/apimachinery v0.19.3
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	k8s.io/test-infra v0.0.0-20201214190528-57362ae63e51
	sigs.k8s.io/boskos v0.0.0-20201218211225-4c6cb9eeb307
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v14.2.0+incompatible
	golang.org/x/lint => golang.org/x/lint v0.0.0-20190409202823-959b441ac422
	k8s.io/api => k8s.io/api v0.19.3
	k8s.io/apimachinery => k8s.io/apimachinery v0.19.3
	k8s.io/client-go => k8s.io/client-go v0.19.3
	k8s.io/code-generator => k8s.io/code-generator v0.19.3
)

go 1.13
