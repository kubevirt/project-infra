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
 * Copyright The KubeVirt Authors.
 */

package git

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
)

func TestGit(t *testing.T) {
	RegisterFailHandler(Fail)

	stat, err := os.Stat("testdata/repo")
	if os.IsNotExist(err) || !stat.IsDir() {
		execGitPanickingOnError("testdata", "clone", "repo.gitbundle", "repo")
		execGitPanickingOnError("testdata/repo", "checkout", "main")
		execGitPanickingOnError("testdata/repo", "checkout", "change-test")
	}

	RunSpecs(t, "Git Main Suite")
}

func execGitPanickingOnError(sourceFilepath string, args ...string) {
	output, err := execGit(sourceFilepath, args)
	if err != nil {
		panic(err)
	}
	log.Infoln(string(output))
}
