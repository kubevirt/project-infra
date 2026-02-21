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
 * Copyright the KubeVirt Authors.
 *
 */

package ginkgo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/onsi/ginkgo/v2/ginkgo/command"
	"github.com/onsi/ginkgo/v2/ginkgo/run"
	"github.com/onsi/ginkgo/v2/types"
	log "github.com/sirupsen/logrus"
)

func DryRun(path string) (reports []types.Report, output []byte, err error) {

	var tempfile *os.File
	tempfile, err = os.CreateTemp("", "ginkgo-report-*.json")
	if err != nil {
		return reports, output, err
	}
	defer func() {
		err := tempfile.Close()
		if err != nil {
			log.Errorf("failed to close %q: %v", tempfile.Name(), err)
		}
		err = os.Remove(tempfile.Name())
		if err != nil {
			log.Errorf("failed to remove %q: %v", tempfile.Name(), err)
		}
	}()

	// since there's no output catchable from the command, we need to use pipe
	// and redirect the output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	buildRunCommand := run.BuildRunCommand()

	outC := make(chan string)

	// since we are using the outline command version that panics on any error
	// we need to handle the panic, returning an error only if the command.AbortDetails
	// indicate that case
	defer func() {
		if r := recover(); r != nil {
			errClose := w.Close()
			if errClose != nil {
				log.Warnf("err on close: %v", errClose)
			}
			os.Stdout = old
			out := <-outC
			output = []byte(out)
			switch x := r.(type) {
			case error:
				err = x
			case command.AbortDetails:
				d := r.(command.AbortDetails)
				if d.Error != nil {
					if strings.Contains(d.Error.Error(), "file does not import \"github.com/onsi/ginkgo/v2\"") {
						err = nil
						return
					}
					err = d.Error
				}

				reportContent, err := io.ReadAll(tempfile)
				if err != nil {
					log.WithError(err).Errorf("failed to read %s", tempfile.Name())
					return
				}

				err = json.Unmarshal(reportContent, &reports)
				if err != nil {
					log.WithError(err).Errorf("failed to read %s", tempfile.Name())
				}

			default:
				err = fmt.Errorf("unknown panic: %v", r)
			}
		}
	}()

	go func() {
		var buf bytes.Buffer
		_, err := io.Copy(&buf, r)
		if err != nil {
			panic(err)
		}
		outC <- buf.String()
	}()

	buildRunCommand.Run([]string{"-v", "--dry-run", "--json-report", tempfile.Name(), path + "/..."}, nil)

	defer func() {
		// restore the output to normal
		err = w.Close()
		os.Stdout = old
		out := <-outC
		output = []byte(out)
	}()

	return []types.Report{}, nil, nil
}

var multipleSpacesMatcher = regexp.MustCompile(`\s{2,}`)

func GetSpecReportByTestName(reports []types.Report, testName string) *types.SpecReport {
	matchingSpecReports := FilterSpecReports(reports, ByName(testName), 1)
	if len(matchingSpecReports) != 1 {
		return nil
	}
	return &matchingSpecReports[0]
}

// ByName checks whether all the node texts of types.SpecReport are contained in the test name
func ByName(testName string) func(r types.SpecReport) bool {
	return func(r types.SpecReport) bool {
		if r.LeafNodeText == "" {
			return false
		}
		if !containsMultiSpaceNormalized(testName, r.LeafNodeText) {
			return false
		}
		for _, text := range r.ContainerHierarchyTexts {
			if !containsMultiSpaceNormalized(testName, text) {
				return false
			}
		}
		return true
	}
}

func FilterSpecReports(reports []types.Report, f SpecReportFilter, maxResults int) []types.SpecReport {
	if maxResults == 0 || maxResults < -1 {
		return nil
	}
	var result []types.SpecReport
	for _, report := range reports {
		for _, specReport := range report.SpecReports {
			if !f(specReport) {
				continue
			}
			result = append(result, specReport)
			if maxResults > 0 && len(result) == maxResults {
				return result
			}
		}
	}
	return result
}

type LabelMatcher func(l string) bool

func NewRegexLabelMatcher(regex string) LabelMatcher {
	pattern := regexp.MustCompile(regex)
	return func(l string) bool {
		return pattern.MatchString(l)
	}
}

func ExtractLabels(r types.SpecReport, matchers ...LabelMatcher) []string {
	var labels []string
	for _, containerLabels := range r.ContainerHierarchyLabels {
		labels = append(labels, filterLabels(containerLabels, matchers...)...)
	}
	labels = append(labels, filterLabels(r.LeafNodeLabels, matchers...)...)
	return labels
}

func filterLabels(labels []string, matchers ...LabelMatcher) []string {
	var filteredLabels []string
	for _, l := range labels {
		matchesAll := true
		for _, m := range matchers {
			if !m(l) {
				matchesAll = false
				break
			}
		}
		if matchesAll {
			filteredLabels = append(filteredLabels, l)
		}
	}
	return filteredLabels
}

func containsMultiSpaceNormalized(fullText, substring string) bool {
	if strings.Contains(fullText, substring) {
		return true
	}
	// special case: gingko "When" nodes add an extra "when " prefix to the node text
	if strings.HasPrefix(substring, "when ") && strings.Contains(fullText, strings.TrimPrefix(substring, "when ")) {
		return true
	}
	// special case: multiple spaces are being normalized in the node text
	if multipleSpacesMatcher.MatchString(substring) {
		// retry with spaces normalized
		normalized := multipleSpacesMatcher.ReplaceAllString(substring, " ")
		if strings.Contains(fullText, normalized) {
			return true
		}
	}
	return false
}
