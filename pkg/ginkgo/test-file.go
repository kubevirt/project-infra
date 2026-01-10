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
	"fmt"
	"os"
	osexec "os/exec"
	"path/filepath"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

var (
	ripgrepOutputMatcher = regexp.MustCompile("^([^:]+):.*$")
)

func init() {
	command := osexec.Command("which", "rg")
	output, err := command.CombinedOutput()
	if err != nil {
		log.Warnf("ripgrep binary check failed: %s", output)
	}
}

func FindTestFileByName(name string, directoryPath string) (string, error) {
	reports, _, err := DryRun(directoryPath)
	if err != nil {
		return "", fmt.Errorf("could not find test file with name %q: %w", name, err)
	}
	if reports == nil {
		return "", fmt.Errorf("could not find test file with name %q", name)
	}
	matchingSpecReport := GetSpecReportByTestName(reports, name)
	if matchingSpecReport == nil {
		return "", fmt.Errorf("could not find test file with name %q", name)
	}
	return matchingSpecReport.FileName(), nil
}

func FindTestFileById(name string, directoryPath string) (string, error) {
	testId, err := GetTestId(name)
	if err != nil {
		return "", err
	}
	command := osexec.Command("rg", regexp.QuoteMeta(testId))
	command.Dir = directoryPath
	return findTestFile(command)
}

func findTestFile(command *osexec.Cmd) (string, error) {
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("couldn't find test file with %v in directory %q: %v", command.String(), command.Dir, err)
	}
	fileNames := make(map[string]struct{})
	for _, line := range strings.Split(string(output), "\n") {
		if line == "" {
			continue
		}
		if !ripgrepOutputMatcher.MatchString(line) {
			log.Warnf("%q did not match regex %q", line, ripgrepOutputMatcher.String())
			continue
		}
		submatch := ripgrepOutputMatcher.FindStringSubmatch(line)
		fileNames[submatch[1]] = struct{}{}
	}
	if len(fileNames) == 0 {
		return "", os.ErrNotExist
	}
	if len(fileNames) > 1 {
		return "", fmt.Errorf("multiple matching files found: %v", fileNames)
	}
	for fileName := range fileNames {
		return filepath.Join(command.Dir, fileName), nil
	}
	return "", nil
}
