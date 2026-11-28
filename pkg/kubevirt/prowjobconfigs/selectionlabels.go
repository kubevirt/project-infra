/*
 * Copyright The KubeVirt Authors.
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package prowjobconfigs

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

const (
	// LabelK8sVersion is the Kubernetes minor the lane targets, for example "1.37".
	// Lanes that do not name a cluster version use SelectionLabelNone.
	// prow-exporter publishes this as label_ci_kubevirt_io_k8s_version on prow_job_labels.
	LabelK8sVersion = "ci.kubevirt.io/k8s-version"
	// LabelProvider is the cluster provider named by the job, "k8s" or "kind".
	LabelProvider = "ci.kubevirt.io/provider"
	// LabelSIG is the SIG lane, for example "compute" or "network".
	LabelSIG = "ci.kubevirt.io/sig"

	// SelectionLabelNone is set when a kubevirt e2e lane has no value for that dimension.
	SelectionLabelNone = "none"
)

// SelectionLabelKeys are the job labels used to filter prow_job_labels in Prometheus.
var SelectionLabelKeys = []string{
	LabelK8sVersion,
	LabelProvider,
	LabelSIG,
}

var (
	kubevirtE2EJobName = regexp.MustCompile(`^(?:pull|periodic|postsubmit)-kubevirt-e2e-`)
	k8sVersionInName   = regexp.MustCompile(`(?:^|-)(?:k8s|kind)-(\d+\.\d+)(?:-|$)`)
	sigInName          = regexp.MustCompile(`(?:^|-)(compute|network|storage|operator|monitoring|performance)(?:-|$)`)
	jobNameLine        = regexp.MustCompile(`^(\s+)name: (\S+)\s*$`)
)

// DeriveSelectionLabels returns the Prometheus selection labels for a kubevirt e2e
// Prow job. Other job names return nil.
func DeriveSelectionLabels(jobName string) map[string]string {
	if !kubevirtE2EJobName.MatchString(jobName) {
		return nil
	}
	return map[string]string{
		LabelK8sVersion: k8sVersionOrNone(jobName),
		LabelProvider:   providerOrNone(jobName),
		LabelSIG:        sigOrNone(jobName),
	}
}

// ApplySelectionLabels sets the selection labels for jobName and drops them when
// the job is not a kubevirt e2e lane. The passed map is updated in place.
func ApplySelectionLabels(labels map[string]string, jobName string) map[string]string {
	derived := DeriveSelectionLabels(jobName)
	if labels == nil {
		if derived == nil {
			return nil
		}
		labels = map[string]string{}
	}
	for _, key := range SelectionLabelKeys {
		if derived == nil {
			delete(labels, key)
			continue
		}
		labels[key] = derived[key]
	}
	return labels
}

// SelectionLabelErrors reports selection labels that are missing or disagree with jobName.
func SelectionLabelErrors(jobName string, labels map[string]string) []string {
	derived := DeriveSelectionLabels(jobName)
	var errs []string
	if derived == nil {
		for _, key := range SelectionLabelKeys {
			if _, ok := labels[key]; ok {
				errs = append(errs, fmt.Sprintf("job %q has %s but is not a kubevirt e2e lane", jobName, key))
			}
		}
		return errs
	}
	for _, key := range SelectionLabelKeys {
		got, ok := labels[key]
		if !ok {
			errs = append(errs, fmt.Sprintf("job %q is missing label %s=%q", jobName, key, derived[key]))
			continue
		}
		if got != derived[key] {
			errs = append(errs, fmt.Sprintf("job %q has %s=%q, expected %q", jobName, key, got, derived[key]))
		}
	}
	return errs
}

func k8sVersionOrNone(jobName string) string {
	match := k8sVersionInName.FindStringSubmatch(jobName)
	if match == nil {
		return SelectionLabelNone
	}
	return match[1]
}

func sigOrNone(jobName string) string {
	match := sigInName.FindStringSubmatch(jobName)
	if match == nil {
		return SelectionLabelNone
	}
	return match[1]
}

func providerOrNone(jobName string) string {
	kindAt := strings.Index(jobName, "-kind-")
	k8sAt := strings.Index(jobName, "-k8s-")
	switch {
	case kindAt >= 0 && (k8sAt < 0 || kindAt < k8sAt):
		return "kind"
	case k8sAt >= 0:
		return "k8s"
	default:
		return SelectionLabelNone
	}
}

// InsertSelectionLabels adds or updates selection labels in a Prow job config
// document without rewriting the rest of the file.
func InsertSelectionLabels(content string) string {
	lines := strings.Split(content, "\n")
	type namedJob struct {
		idx    int
		name   string
		indent int
	}
	var jobs []namedJob
	for i, line := range lines {
		match := jobNameLine.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		name := strings.Trim(match[2], `"'`)
		if DeriveSelectionLabels(name) == nil {
			continue
		}
		jobs = append(jobs, namedJob{idx: i, name: name, indent: len(match[1])})
	}
	for i := len(jobs) - 1; i >= 0; i-- {
		lines = upsertJobSelectionLabels(lines, jobs[i].idx, jobs[i].name, jobs[i].indent)
	}
	return strings.Join(lines, "\n")
}

func upsertJobSelectionLabels(lines []string, nameIdx int, jobName string, nameIndent int) []string {
	derived := DeriveSelectionLabels(jobName)
	start, end := jobBounds(lines, nameIdx, nameIndent)
	labelsIdx := -1
	for i := start; i < end; i++ {
		if isLabelsKey(lines[i], nameIndent) {
			labelsIdx = i
			break
		}
	}
	labelIndent := strings.Repeat(" ", nameIndent+2)
	if labelsIdx == -1 {
		block := append([]string{strings.Repeat(" ", nameIndent) + "labels:"}, formatLabelLines(labelIndent, derived)...)
		return insertLines(lines, nameIdx, block)
	}

	entryStart := labelsIdx + 1
	entryEnd := entryStart
	for entryEnd < end {
		line := lines[entryEnd]
		if strings.TrimSpace(line) == "" || leadingSpaces(line) <= nameIndent {
			break
		}
		entryEnd++
	}
	entries := append([]string{}, lines[entryStart:entryEnd]...)
	entries = mergeLabelEntries(entries, labelIndent, derived)
	return replaceRange(lines, entryStart, entryEnd, entries)
}

func jobBounds(lines []string, nameIdx, nameIndent int) (start, end int) {
	start = 0
	for i := nameIdx - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) == "" {
			continue
		}
		if leadingSpaces(lines[i]) < nameIndent {
			start = i
			break
		}
	}
	end = len(lines)
	for i := nameIdx + 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "" {
			continue
		}
		if leadingSpaces(lines[i]) < nameIndent {
			end = i
			break
		}
	}
	return start, end
}

func mergeLabelEntries(entries []string, indent string, derived map[string]string) []string {
	keys := append([]string{}, SelectionLabelKeys...)
	sort.Strings(keys)
	for _, key := range keys {
		formatted := fmt.Sprintf("%s%s: %q", indent, key, derived[key])
		prefix := indent + key + ":"
		replaced := false
		for i, entry := range entries {
			if strings.HasPrefix(strings.TrimRight(entry, " \t"), prefix) {
				entries[i] = formatted
				replaced = true
				break
			}
		}
		if replaced {
			continue
		}
		insertAt := len(entries)
		for i, entry := range entries {
			existing := labelKey(entry)
			if existing == "" {
				continue
			}
			if existing > key {
				insertAt = i
				break
			}
		}
		entries = insertLines(entries, insertAt, []string{formatted})
	}
	return entries
}

func formatLabelLines(indent string, derived map[string]string) []string {
	keys := append([]string{}, SelectionLabelKeys...)
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, fmt.Sprintf("%s%s: %q", indent, key, derived[key]))
	}
	return lines
}

func labelKey(line string) string {
	trim := strings.TrimSpace(line)
	if trim == "" || strings.HasPrefix(trim, "#") || strings.HasPrefix(trim, "-") {
		return ""
	}
	colon := strings.Index(trim, ":")
	if colon <= 0 {
		return ""
	}
	return trim[:colon]
}

func isLabelsKey(line string, nameIndent int) bool {
	if leadingSpaces(line) == nameIndent && strings.TrimSpace(line) == "labels:" {
		return true
	}
	if nameIndent < 2 {
		return false
	}
	prefix := strings.Repeat(" ", nameIndent-2) + "- "
	rest, ok := strings.CutPrefix(line, prefix)
	return ok && strings.TrimSpace(rest) == "labels:"
}

func leadingSpaces(line string) int {
	return len(line) - len(strings.TrimLeft(line, " "))
}

func insertLines(lines []string, at int, extra []string) []string {
	out := make([]string, 0, len(lines)+len(extra))
	out = append(out, lines[:at]...)
	out = append(out, extra...)
	out = append(out, lines[at:]...)
	return out
}

func replaceRange(lines []string, start, end int, extra []string) []string {
	out := make([]string, 0, len(lines)-(end-start)+len(extra))
	out = append(out, lines[:start]...)
	out = append(out, extra...)
	out = append(out, lines[end:]...)
	return out
}
