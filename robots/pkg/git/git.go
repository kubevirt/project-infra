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

var gitBlameRegex = regexp.MustCompile(`^([0-9a-f]+)(\s+\S+)?\s+\(([\S ]+)\s([0-9]{4}-[0-9]{2}-[0-9]{2}\s[0-9]{2}:[0-9]{2}:[0-9]{2}\s[-+][0-9]{4})\s+([0-9]+)\)\s(.*)$`)

// BlameLine holds the record of a line of data that the git blame command provides
type BlameLine struct {
	CommitID string    `json:"commit_id"`
	Author   string    `json:"author"`
	Date     time.Time `json:"date"`
	LineNo   int       `json:"line_no"`
	Line     string    `json:"line"`
}

// GetBlameLinesForFile returns the git blame information per line for the given file. If no line numbers are given via `lineNos` then the blame for the full file is returned, otherwise the blame is reduced to the given line numbers of the file
func GetBlameLinesForFile(testFilePath string, lineNos ...int) ([]*BlameLine, error) {
	blameLines, err := getBlameForFile(testFilePath, lineNos...)
	if err != nil {
		return nil, err
	}
	gitBlameInfo := extractBlameInfo(blameLines)
	return gitBlameInfo, nil
}

func extractBlameInfo(lines []string) []*BlameLine {
	var info []*BlameLine
	for _, line := range lines {
		if !gitBlameRegex.MatchString(line) {
			continue
		}
		submatches := gitBlameRegex.FindAllStringSubmatch(line, -1)
		date, err := time.Parse(BlameDateLayout, submatches[0][4])
		if err != nil {
			panic(err)
		}
		lineNo, err := strconv.Atoi(submatches[0][5])
		if err != nil {
			panic(err)
		}
		info = append(info, &BlameLine{
			CommitID: submatches[0][1],
			Author:   strings.TrimSpace(submatches[0][3]),
			Date:     date,
			LineNo:   lineNo,
			Line:     submatches[0][6],
		})
	}
	return info
}

// getBlameForFile returns the git blame information for the given file. If no line numbers are given via `lineNos` then the blame for the full file is returned, otherwise the blame is reduced to the given line numbers of the file
func getBlameForFile(testFilePath string, lineNos ...int) ([]string, error) {
	blameArgs := []string{"blame", filepath.Base(testFilePath)}
	for _, blameLineNo := range lineNos {
		blameArgs = append(blameArgs, fmt.Sprintf("-L %d,%d", blameLineNo, blameLineNo))
	}
	command := exec.Command("git", blameArgs...)
	command.Dir = filepath.Dir(testFilePath)
	output, err := command.Output()
	if err != nil {
		switch err.(type) {
		case *exec.ExitError:
			e := err.(*exec.ExitError)
			return nil, fmt.Errorf("exec %v failed: %v", command, e.Stderr)
		case *exec.Error:
			e := err.(*exec.Error)
			return nil, fmt.Errorf("exec %v failed: %v", command, e)
		default:
			return nil, fmt.Errorf("exec %v failed: %v", command, err)
		}
	}
	blameLines := strings.Split(string(output), "\n")
	return blameLines, nil
}
