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
	"github.com/onsi/ginkgo/v2/ginkgo/command"
	"github.com/onsi/ginkgo/v2/ginkgo/run"
	"github.com/onsi/ginkgo/v2/types"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"regexp"
	"strings"
)

func DryRun(path string) (reports []types.Report, output []byte, err error) {

	var tempfile *os.File
	tempfile, err = os.CreateTemp("", "ginkgo-report-*.json")
	if err != nil {
		return reports, output, err
	}
	defer os.Remove(tempfile.Name())

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

func GetMatchingSpecReport(reports []types.Report, testName string) *types.SpecReport {
	for _, report := range reports {
	specReportLoop:
		for _, specReport := range report.SpecReports {
			if specReport.LeafNodeText == "" {
				continue specReportLoop
			}
			if !containsMultiSpaceNormalized(testName, specReport.LeafNodeText) {
				continue specReportLoop
			}
			for _, text := range specReport.ContainerHierarchyTexts {
				if !containsMultiSpaceNormalized(testName, text) {
					continue specReportLoop
				}
			}
			return &specReport
		}
	}
	return nil
}

func containsMultiSpaceNormalized(fullText, substring string) bool {
	if strings.Contains(fullText, substring) {
		return true
	}
	if multipleSpacesMatcher.MatchString(substring) {
		// retry with spaces normalized
		normalized := multipleSpacesMatcher.ReplaceAllString(substring, " ")
		if strings.Contains(fullText, normalized) {
			return true
		}
	}
	return false
}
