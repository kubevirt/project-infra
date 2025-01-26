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
	log "github.com/sirupsen/logrus"
	"os"
	osexec "os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	ripgrepOutputMatcher = regexp.MustCompile("^([^:]+):.*$")
	testIdMatcher        = regexp.MustCompile("\\[test_id:[0-9]+]")
	testIdExtractor      = regexp.MustCompile("(\\[test_id:[0-9]+])")
)

func init() {
	command := osexec.Command("which", "rg")
	output, err := command.CombinedOutput()
	if err != nil {
		log.Warnf("ripgrep binary check failed: %s", output)
	}
}

func FindTestFileByName(name string, directoryPath string) (string, error) {
	var command *osexec.Cmd
	if testIdMatcher.MatchString(name) {
		submatch := testIdExtractor.FindStringSubmatch(name)
		command = osexec.Command("rg", regexp.QuoteMeta(submatch[1]))
		command.Dir = directoryPath
	} else {
		command = osexec.Command("rg", "--multiline", "--multiline-dotall", byTestName(name))
		command.Dir = directoryPath
	}
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("couldn't find test file with %v in directory %q: %v", command.String(), directoryPath, err)
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
	for fileName, _ := range fileNames {
		return filepath.Join(directoryPath, fileName), nil
	}
	return "", nil
}

func byTestName(name string) string {
	regex := fmt.Sprintf(`"%s"`, strings.ReplaceAll(regexp.QuoteMeta(name), " ", ".*"))
	regexp.MustCompile(regex)
	return regex
}
