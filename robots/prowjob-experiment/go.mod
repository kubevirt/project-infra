module kubevirt.io/project-infra/robots/prowjob-experiment

go 1.13

replace k8s.io/client-go => k8s.io/client-go v0.0.0-20190918200256-06eb1244587a

require (
	github.com/sirupsen/logrus v1.4.2
	k8s.io/test-infra v0.0.0-20200113075537-d6212fa62fb0
	sigs.k8s.io/yaml v1.1.0
)
