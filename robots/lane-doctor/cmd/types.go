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

package cmd

// ScanReport is the output of the scan command, listing all stuck PRs for a lane.
type ScanReport struct {
	Lane      string      `json:"lane"`
	Repo      string      `json:"repo"`
	ScannedAt string      `json:"scannedAt"`
	Summary   ScanSummary `json:"summary"`
	StuckPRs  []StuckPR   `json:"stuckPRs"`
}

// ScanSummary counts PRs in each classification category.
type ScanSummary struct {
	Total   int `json:"total"`
	Stuck   int `json:"stuck"`
	Missing int `json:"missing"`
	Running int `json:"running"`
	Success int `json:"success"`
	Failed  int `json:"failed"`
}

// StuckPR describes a PR whose lane status is stuck or missing.
type StuckPR struct {
	Number          int      `json:"number"`
	Title           string   `json:"title"`
	Author          string   `json:"author"`
	HeadSHA         string   `json:"headSHA"`
	UpdatedAt       string   `json:"updatedAt"`
	Labels          []string `json:"labels"`
	IsDraft         bool     `json:"isDraft"`
	StatusState     string   `json:"statusState"`
	StatusUpdatedAt string   `json:"statusUpdatedAt,omitempty"`
	HasTargetURL    bool     `json:"hasTargetURL"`
}

// PriorityReport is the output of the prioritize command.
type PriorityReport struct {
	Lane          string          `json:"lane"`
	Repo          string          `json:"repo"`
	PrioritizedAt string          `json:"prioritizedAt"`
	Groups        []PriorityGroup `json:"groups"`
}

// PriorityGroup is a named tier (P1–P4) of PR numbers.
type PriorityGroup struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	PRNumbers   []int  `json:"prNumbers"`
}
