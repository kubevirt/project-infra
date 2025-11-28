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
	"regexp"
	"strings"
)

type FileChange struct {
	ChangeType  DiffFilterModifier
	Filename    string
	OldFilename string
}
type LogCommit struct {
	Hash        string
	FileChanges []*FileChange
}

var commitHashRegex = regexp.MustCompile(`^[a-z0-9]+$`)
var fileChangeRegex = regexp.MustCompile(`^([ACDMRTUXB])([0-9]{3})?\s+(\S*)(\s+(.*))?$`)

type DiffFilterModifier string

const (
	Added         = DiffFilterModifier("A")
	Copied        = DiffFilterModifier("C")
	Deleted       = DiffFilterModifier("D")
	Modified      = DiffFilterModifier("M")
	Renamed       = DiffFilterModifier("R")
	TypeChanged   = DiffFilterModifier("T")
	Unmerged      = DiffFilterModifier("U")
	Unknown       = DiffFilterModifier("X")
	BrokenPairing = DiffFilterModifier("B")
)

func LogCommits(revisionRange string, path string, subDirectory string) ([]*LogCommit, error) {
	var logCommits []*LogCommit
	output, err := execGit(path, []string{"log", "--format=%H", "--name-status", revisionRange, "--", subDirectory})
	if err != nil {
		return nil, err
	}
	for _, outputLine := range strings.Split(string(output), "\n") {
		switch {
		case commitHashRegex.MatchString(outputLine):
			logCommits = append(logCommits, &LogCommit{Hash: outputLine})
		case fileChangeRegex.MatchString(outputLine):
			logCommit := logCommits[len(logCommits)-1]
			submatch := fileChangeRegex.FindStringSubmatch(outputLine)
			diffFilterModifier := DiffFilterModifier(submatch[1])
			fileName := submatch[3]
			oldFileName := ""
			if submatch[5] != "" {
				fileName = submatch[5]
				oldFileName = submatch[3]
			}
			fileChange := FileChange{
				ChangeType:  diffFilterModifier,
				Filename:    fileName,
				OldFilename: oldFileName,
			}
			logCommit.FileChanges = append(logCommit.FileChanges, &fileChange)
		}
	}
	return logCommits, nil
}

func GetLatestMergeCommit(path string, subDirectory string) (string, error) {
	output, err := execGit(path, []string{"log", "--max-count=1", "--format=%H", "--merges", "--", subDirectory})
	if err != nil {
		return "", err
	}
	mergeCommit := string(output)
	mergeCommit = strings.TrimSuffix(mergeCommit, "\n")
	return mergeCommit, nil
}
