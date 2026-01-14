/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2023 Red Hat, Inc.
 */

package get

import (
	_ "embed"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	"kubevirt.io/project-infra/pkg/querier"
	"sigs.k8s.io/prow/pkg/config"
)

const k8sScheduleYAML = "https://raw.githubusercontent.com/kubernetes/website/main/data/releases/schedule.yaml"
const k8sEOLYAML = "https://raw.githubusercontent.com/kubernetes/website/main/data/releases/eol.yaml"

type schedulesDoc struct {
	Schedules []*Schedule `yaml:"schedules"`
}

var getSupportMatrixCommand = &cobra.Command{
	Use:   "support-matrix",
	Short: "kubevirt get support-matrix generates a support matrix document from k8s release schedule and project-infra job data for kubevirt/kubevirt",
	Long: getPeriodicsShortDescription + `

It reads the job configurations for kubevirt/kubevirt e2e presubmit jobs, extracts information and creates a table in 
html format, matching supported k8s versions towards k6t versions.
`,
	RunE: GenerateMarkdownForSupportMatrix,
}

var jobNameRegex = regexp.MustCompile(`^pull-kubevirt-e2e-k8s-([0-9]+\.[0-9]+)-sig-compute$`)
var k6tVersionRegex = regexp.MustCompile(`^.*kubevirt-presubmits(-([0-9]+\.[0-9]+))?\.yaml$`)

//go:embed support-matrix.gomd
var getSupportMatrixTemplate string

type getSupportMatrixOptions struct {
	OutputFile             string
	OverwriteOutputFile    bool
	JobConfigDirectoryPath string
	KubeVirtVersion        string
}

func (o *getSupportMatrixOptions) validateOptions() error {
	if o.OutputFile != "" {
		_, err := os.Stat(o.OutputFile)
		if !o.OverwriteOutputFile && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("output file %s might exist already: %v", o.OutputFile, err)
		}
	}
	stat, err := os.Stat(o.JobConfigDirectoryPath)
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("job config directory %s does not exist", o.JobConfigDirectoryPath)
	}
	if !stat.IsDir() {
		return fmt.Errorf("job config directory %s is not a directory", o.JobConfigDirectoryPath)
	}
	return nil
}

var getSupportMatrixOpts = getSupportMatrixOptions{}

func init() {
	getSupportMatrixCommand.PersistentFlags().StringVar(&getSupportMatrixOpts.OutputFile, "output-file", "", "output file to write to, otherwise standard out will be used")
	getSupportMatrixCommand.PersistentFlags().BoolVar(&getSupportMatrixOpts.OverwriteOutputFile, "overwrite-output-file", false, "output file should be overwritten if it exists")
	getSupportMatrixCommand.PersistentFlags().StringVar(&getSupportMatrixOpts.JobConfigDirectoryPath, "job-config-path", "", "path to kubevirt job configuration files")
	getSupportMatrixCommand.PersistentFlags().StringVar(&getSupportMatrixOpts.KubeVirtVersion, "kubevirt-version", "", "version of kubevirt to generate matrix for, default is empty string, which means all recent versions")
}

type SupportMatrixTemplateData struct {
	K6tVersions          []string
	K8sVersions          []string
	SupportedK8sVersions map[string]bool
	MapK6tToK8sVersions  map[string]map[string]bool
}

func GenerateMarkdownForSupportMatrix(_ *cobra.Command, _ []string) error { //nolint:gocyclo
	if err := getSupportMatrixOpts.validateOptions(); err != nil {
		return fmt.Errorf("failed to validate options: %v", err)
	}

	fileNames, err := getJobConfigFileNames(getSupportMatrixOpts.JobConfigDirectoryPath)
	if err != nil {
		return err
	}

	// each k8t version supports the latest three k8s versions at time of release
	// but to make sure that we actually print the ones that versions are tested against we read the kubevirt presubmit job files and check against
	// sig-compute required ones, aka always_run: true, optional: false
	k8sVersionsSet := make(map[string]struct{})
	k6tVersionsSet := make(map[string]struct{})
	mapK6tToK8sVersions := make(map[string]map[string]bool)
	for _, file := range fileNames {
		k6tVersion := ""
		if !k6tVersionRegex.MatchString(file) {
			return fmt.Errorf("no k6t version available: %s", file)
		}
		submatch := k6tVersionRegex.FindStringSubmatch(file)
		if submatch != nil {
			k6tVersion = submatch[2]
		}
		if k6tVersion == "" {
			continue
		} else {
			if getSupportMatrixOpts.KubeVirtVersion != "" && k6tVersion != getSupportMatrixOpts.KubeVirtVersion {
				continue
			}
			k6tVersionsSet[k6tVersion] = struct{}{}
		}
		jobConfig, readConfigErr := config.ReadJobConfig(file)
		if readConfigErr != nil {
			return fmt.Errorf("failed to read job config: %v", readConfigErr)
		}
		for _, presubmit := range jobConfig.PresubmitsStatic["kubevirt/kubevirt"] {
			if !presubmit.AlwaysRun || presubmit.Optional {
				continue
			}
			k8sVersion := ""
			if !jobNameRegex.MatchString(presubmit.Name) && !jobNameRegex.MatchString(presubmit.Context) {
				continue
			}
			if jobNameRegex.MatchString(presubmit.Context) {
				k8sVersion = jobNameRegex.FindStringSubmatch(presubmit.Context)[1]
			}
			if jobNameRegex.MatchString(presubmit.Name) {
				k8sVersion = jobNameRegex.FindStringSubmatch(presubmit.Name)[1]
			}
			if k8sVersion == "" {
				continue
			}
			k8sVersionsSet[k8sVersion] = struct{}{}
			if _, exists := mapK6tToK8sVersions[k6tVersion]; !exists {
				mapK6tToK8sVersions[k6tVersion] = make(map[string]bool)
			}
			mapK6tToK8sVersions[k6tVersion][k8sVersion] = true
		}
	}

	k8sVersions := make([]string, 0)
	for k8sVersion := range k8sVersionsSet {
		k8sVersions = append(k8sVersions, k8sVersion)
	}

	latestK8sEOLRelease, err := fetchLatestK8sEOLRelease()
	if err != nil {
		return err
	}
	isK8sVersionEOL := func(k8sVersion string) bool {
		semVer := mustParseMajorMinor(k8sVersion)
		return latestK8sEOLRelease.Compare(&semVer) >= 0
	}
	supportedK8sVersions := make(map[string]bool, 0)
	for _, k8sVersion := range k8sVersions {
		if isK8sVersionEOL(k8sVersion) {
			continue
		}
		supportedK8sVersions[k8sVersion] = true
	}

	generateStringSemVerComparison := func(versions []string) func(i, j int) bool {
		return func(i, j int) bool {
			jSemVer, iSemVer := mustParseMajorMinor(versions[j]), mustParseMajorMinor(versions[i])
			return jSemVer.Compare(&iSemVer) < 0
		}
	}

	sort.SliceStable(k8sVersions, generateStringSemVerComparison(k8sVersions))

	// no more than three EOL K8S versions
	noMoreThanXEOLK8sVersions := 3
	numberOfEOLK8sVersionsSoFar := 0
	k8sVersionsWithLessEOLVersions := make([]string, 0)
	for _, k8sVersion := range k8sVersions {
		if isK8sVersionEOL(k8sVersion) {
			if numberOfEOLK8sVersionsSoFar >= noMoreThanXEOLK8sVersions {
				break
			}
			numberOfEOLK8sVersionsSoFar++
		}
		k8sVersionsWithLessEOLVersions = append(k8sVersionsWithLessEOLVersions, k8sVersion)
	}
	k8sVersions = k8sVersionsWithLessEOLVersions

	k8sVersionsSet = make(map[string]struct{})
	for _, k8sVersion := range k8sVersions {
		k8sVersionsSet[k8sVersion] = struct{}{}
	}

	k6tVersionsWithK8sInformation := make([]string, 0)
	for k6tVersion, supportedK8sVersionsForK6tVersion := range mapK6tToK8sVersions {
		// if no support for any version then continue
		if len(supportedK8sVersionsForK6tVersion) == 0 {
			continue
		}

		// search for any supported version inside the stripped down versions
		contains := false
	containsLoop:
		for supportedK8sVersion := range supportedK8sVersionsForK6tVersion {
			if _, k8sVersionExists := k8sVersionsSet[supportedK8sVersion]; k8sVersionExists {
				contains = true
				break containsLoop
			}
		}
		if !contains {
			continue
		}
		k6tVersionsWithK8sInformation = append(k6tVersionsWithK8sInformation, k6tVersion)
	}
	k6tVersions := k6tVersionsWithK8sInformation
	sort.SliceStable(k6tVersions, generateStringSemVerComparison(k6tVersions))

	var writer io.Writer
	if getSupportMatrixOpts.OutputFile != "" {
		writer, err = os.OpenFile(getSupportMatrixOpts.OutputFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			return fmt.Errorf("failed to open file %q: %v", getSupportMatrixOpts.OutputFile, err)
		}
	} else {
		writer = os.Stdout
	}

	supportMatrixTemplate, err := template.New("support-matrix").Parse(getSupportMatrixTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse go template: %v", err)
	}

	err = supportMatrixTemplate.Execute(writer, SupportMatrixTemplateData{
		K6tVersions:          k6tVersions,
		K8sVersions:          k8sVersions,
		SupportedK8sVersions: supportedK8sVersions,
		MapK6tToK8sVersions:  mapK6tToK8sVersions,
	})
	if err != nil {
		return fmt.Errorf("failed to execute go template: %v", err)
	}

	return nil
}

func mustParseMajorMinor(k8sVersion string) querier.SemVer {
	semVer, parseErr := parseMajorMinor(k8sVersion)
	if parseErr != nil {
		log.Fatalf("failed to parse %q: %v", k8sVersion, parseErr)
	}
	return semVer
}

func fetchLatestK8sEOLRelease() (querier.SemVer, error) {
	schedules, err := fetchK8sSchedules()
	if err != nil {
		return querier.SemVer{}, fmt.Errorf("failed to fetch k8s eols: %v", err)
	}
	eols, err := fetchK8sEOLs()
	if err != nil {
		return querier.SemVer{}, fmt.Errorf("failed to fetch k8s eols: %v", err)
	}
	k8sReleasesToEOLDates, err := getK8sReleasesToEOLDates(schedules.Schedules, eols.Branches)
	if err != nil {
		return querier.SemVer{}, fmt.Errorf("failed to get k8s releases to eol dates: %v", err)
	}
	var latestEOLRelease querier.SemVer
	var latestEOLDate time.Time
	for k8sRelease, eolDate := range k8sReleasesToEOLDates {
		if !eolDate.Before(time.Now()) {
			continue
		}
		if !eolDate.After(latestEOLDate) {
			continue
		}
		eolRelease, parseMajorMinorErr := parseMajorMinor(k8sRelease)
		if parseMajorMinorErr != nil {
			return querier.SemVer{}, fmt.Errorf("failed to parse k8s version %q: %v", k8sRelease, parseMajorMinorErr)
		}
		latestEOLRelease, latestEOLDate = eolRelease, eolDate
	}
	return latestEOLRelease, nil
}

var semVerWithOptionalReleaseRegex = regexp.MustCompile(`^([0-9]+)\.([0-9]+)(\.([0-9]+))?$`)

func parseMajorMinor(version string) (querier.SemVer, error) {
	if !semVerWithOptionalReleaseRegex.MatchString(version) {
		return querier.SemVer{}, fmt.Errorf("invalid major minor version string: %s", version)
	}
	subMatches := semVerWithOptionalReleaseRegex.FindAllStringSubmatch(version, -1)
	return querier.SemVer{
		Major: subMatches[0][1],
		Minor: subMatches[0][2],
	}, nil
}

func getJobConfigFileNames(jobConfigPath string) ([]string, error) {
	var fileNames []string
	err := filepath.WalkDir(jobConfigPath, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() || !k6tVersionRegex.MatchString(d.Name()) {
			return nil
		}
		fileNames = append(fileNames, filepath.Join(jobConfigPath, d.Name()))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk dir %q: %v", jobConfigPath, err)
	}
	return fileNames, nil
}

func fetchK8sSchedules() (*schedulesDoc, error) {
	resp, err := http.Get(k8sScheduleYAML)
	if err != nil {
		return nil, fmt.Errorf("error when fetching k8s schedule yaml: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error when reading k8s schedule yaml: %v", err)
	}
	var k8sSchedules *schedulesDoc
	err = yaml.Unmarshal(body, &k8sSchedules)
	if err != nil {
		return nil, fmt.Errorf("error on deserializing k8s schedule yaml: %v", err)
	}
	return k8sSchedules, nil
}

func fetchK8sEOLs() (*EOLDoc, error) {
	resp, err := http.Get(k8sEOLYAML)
	if err != nil {
		return nil, fmt.Errorf("error when fetching %q: %v", k8sEOLYAML, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error when reading %q: %v", k8sEOLYAML, err)
	}
	var eolDoc *EOLDoc
	err = yaml.Unmarshal(body, &eolDoc)
	if err != nil {
		return nil, fmt.Errorf("error on deserializing %q: %v", k8sEOLYAML, err)
	}
	return eolDoc, nil
}

func getK8sReleasesToEOLDates(k8sSchedules []*Schedule, branches []*Branch) (map[string]time.Time, error) {
	supportedK8sReleases := make(map[string]time.Time)
	for _, k8sSchedule := range k8sSchedules {
		date := k8sSchedule.EndOfLifeDate
		eolDate, err := parseDate(date)
		if err != nil {
			return nil, err
		}
		supportedK8sReleases[k8sSchedule.Release] = eolDate
	}
	for _, branch := range branches {
		date := branch.EndOfLifeDate
		eolDate, err := parseDate(date)
		if err != nil {
			return nil, err
		}
		supportedK8sReleases[branch.Release] = eolDate
	}
	return supportedK8sReleases, nil
}

func parseDate(date string) (time.Time, error) {
	eolDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return time.Time{}, fmt.Errorf("error on parsing eol date: %v", err)
	}
	return eolDate, nil
}
