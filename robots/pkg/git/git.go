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

package git

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// BlameDateLayout is the layout that is used to parse git blame dates
const BlameDateLayout = "2006-01-02 15:04:05 -0700"

var gitBlameRegex = regexp.MustCompile(`^(\^?[0-9a-f]+)(\s+\S+)?\s+\(([\S ]+)\s([0-9]{4}-[0-9]{2}-[0-9]{2}\s[0-9]{2}:[0-9]{2}:[0-9]{2}\s[-+][0-9]{4})\s+([0-9]+)\)\s(.*)$`)

// string submatch indexes for the regex
const (
	commitID = iota + 1
	_
	author
	date
	lineNo
	line
)

// BlameLine holds the record of a line of data that the git blame command provides
type BlameLine struct {
	CommitID string    `json:"commit_id"`
	Author   string    `json:"author"`
	Date     time.Time `json:"date"`
	LineNo   int       `json:"line_no"`
	Line     string    `json:"line"`
}

// GetBlameLinesForFile returns the git blame information per line for the given file. If no line numbers are given via `lineNos` then the blame for the full file is returned, otherwise the blame is reduced to the given line numbers of the file
func GetBlameLinesForFile(sourceFilepath string, lineNos ...int) ([]*BlameLine, error) {
	blameLines, err := getBlameForFile(sourceFilepath, lineNos...)
	if err != nil {
		return nil, err
	}
	gitBlameInfo := extractBlameInfo(blameLines)
	return gitBlameInfo, nil
}

func extractBlameInfo(gitBlameLines []string) []*BlameLine {
	var info []*BlameLine
	for _, blameLine := range gitBlameLines {
		if !gitBlameRegex.MatchString(blameLine) {
			continue
		}
		submatch := gitBlameRegex.FindStringSubmatch(blameLine)
		commitDate, err := time.Parse(BlameDateLayout, submatch[date])
		if err != nil {
			panic(err)
		}
		fileLineNo, err := strconv.Atoi(submatch[lineNo])
		if err != nil {
			panic(err)
		}
		info = append(info, &BlameLine{
			CommitID: submatch[commitID],
			Author:   strings.TrimSpace(submatch[author]),
			Date:     commitDate,
			LineNo:   fileLineNo,
			Line:     submatch[line],
		})
	}
	return info
}

// getBlameForFile returns the git blame information for the given file. If no line numbers are given via `lineNos` then the blame for the full file is returned, otherwise the blame is reduced to the given line numbers of the file
func getBlameForFile(sourceFilepath string, lineNos ...int) ([]string, error) {
	blameArgs := []string{"blame", filepath.Base(sourceFilepath)}
	for _, blameLineNo := range lineNos {
		blameArgs = append(blameArgs, fmt.Sprintf("-L %d,%d", blameLineNo, blameLineNo))
	}
	command := exec.Command("git", blameArgs...)
	command.Dir = filepath.Dir(sourceFilepath)
	output, err := command.Output()
	if err != nil {
		switch e := err.(type) {
		case *exec.ExitError:
			return nil, fmt.Errorf("exec %s failed: %s", command, e.Stderr)
		case *exec.Error:
			return nil, fmt.Errorf("exec %s failed: %s", command, e)
		default:
			return nil, fmt.Errorf("exec %s failed: %s", command, err)
		}
	}
	blameLines := strings.Split(string(output), "\n")
	return blameLines, nil
}
