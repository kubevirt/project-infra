package test_label_analyzer

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

var gitBlameRegex = regexp.MustCompile(`^([0-9a-f]+)(\s+\S+)?\s+\(([\S ]+)\s([0-9]{4}-[0-9]{2}-[0-9]{2}\s[0-9]{2}:[0-9]{2}:[0-9]{2}\s[-+][0-9]{4})\s+([0-9]+)\)\s(.*)$`)

type GitBlameInfo struct {
	CommitID string    `json:"commit_id"`
	Author   string    `json:"author"`
	Date     time.Time `json:"date"`
	LineNo   int       `json:"line_no"`
	Line     string    `json:"line"`
}

func ExtractGitBlameInfo(lines []string) []*GitBlameInfo {
	var info []*GitBlameInfo
	for _, line := range lines {
		if !gitBlameRegex.MatchString(line) {
			continue
		}
		submatches := gitBlameRegex.FindAllStringSubmatch(line, -1)
		date, err := time.Parse(gitDateLayout, submatches[0][4])
		if err != nil {
			panic(err)
		}
		lineNo, err := strconv.Atoi(submatches[0][5])
		if err != nil {
			panic(err)
		}
		info = append(info, &GitBlameInfo{
			CommitID: submatches[0][1],
			Author:   strings.TrimSpace(submatches[0][3]),
			Date:     date,
			LineNo:   lineNo,
			Line:     submatches[0][6],
		})
	}
	return info
}
