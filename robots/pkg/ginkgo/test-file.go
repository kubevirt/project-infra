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

func init() {
	command := osexec.Command("which", "rg")
	output, err := command.CombinedOutput()
	if err != nil {
		log.Warnf("ripgrep binary check failed: %s", output)
	}
}

func FindTestFileByName(name string, directoryPath string) (string, error) {
	command := osexec.Command("rg", "--multiline", "--multiline-dotall", byTestName(name))
	command.Dir = directoryPath
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("couldn't find test file with %v: %v", command.String(), err)
	}
	fileNames := make(map[string]struct{})
	ripgrepOutputMatcher := regexp.MustCompile("(?m)^([^:]+):[^:]+$")
	allStringSubmatches := ripgrepOutputMatcher.FindAllStringSubmatch(string(output), -1)
	for _, submatch := range allStringSubmatches {
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
	regex := fmt.Sprintf(`"%s"`, strings.ReplaceAll(name, " ", ".*"))
	regexp.MustCompile(regex)
	return regex
}
