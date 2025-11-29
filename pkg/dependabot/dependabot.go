package dependabot

import (
	"golang.org/x/mod/semver"
	"kubevirt.io/project-infra/pkg/dependabot/api"
)

type GroupedAlert struct {
	Name          string
	LatestVersion string
}

// FilterAlerts looks at all project errors, and filters out package duplicates,
// keeping the newest versions of a package which solve all CVEs
func FilterAlerts(alerts []api.Alert) map[string]*GroupedAlert {
	latest := map[string]*GroupedAlert{}

	for _, cve := range api.GetOpenGolangCVEs(alerts) {
		if _, exists := latest[cve.PackageName]; !exists {
			latest[cve.PackageName] = &GroupedAlert{
				Name:          cve.PackageName,
				LatestVersion: cve.FixedPackageVersion,
			}
		} else {
			v := latest[cve.PackageName]
			if semver.Compare(cve.FixedPackageVersion, v.LatestVersion) > 0 {
				v.LatestVersion = cve.FixedPackageVersion
			}
		}
	}
	return latest
}
