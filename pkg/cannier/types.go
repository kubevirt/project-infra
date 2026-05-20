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

package cannier

const (
	ReadCount = iota
	WriteCount
	RunTime
	WaitTime
	ContextSwitches
	CoveredLines
	SourceCoveredLines
	CoveredChanges
	MaxThreads
	MaxChildren
	MaxMemory
	ASTDepth
	Assertions
	ExternalModules
	HalsteadVolume
	CyclomaticComplexity
	TestLinesOfCode
	Maintainability
)

var featureNames = [18]string{
	"ReadCount",
	"WriteCount",
	"RunTime",
	"WaitTime",
	"ContextSwitches",
	"CoveredLines",
	"SourceCoveredLines",
	"CoveredChanges",
	"MaxThreads",
	"MaxChildren",
	"MaxMemory",
	"ASTDepth",
	"Assertions",
	"ExternalModules",
	"HalsteadVolume",
	"CyclomaticComplexity",
	"TestLinesOfCode",
	"Maintainability",
}

func FeatureNames() [18]string {
	result := featureNames
	return result
}

// FeatureSet holds the features of the CANNIER framework.
//
// See the CANNIER paper at https://www.gregorykapfhammer.com/research/papers/parry2023/
type FeatureSet struct {

	/* static analysis */

	// ASTDepth holds the maximum depth of nested program statements in the test case code.
	ASTDepth int `json:"ast_depth"`

	// Assertions holds the number of assertion statements in the test case code.
	Assertions int `json:"assertions"`

	// CyclomaticComplexity holds the number of branches in the test case code
	CyclomaticComplexity int `json:"cyclomatic_complexity"`

	// TestLinesOfCode holds the number of lines in the test case code
	TestLinesOfCode int `json:"test_lines_of_code"`

	// ExternalModules holds the number of non-standard modules (i.e., libraries) used by the
	//test case
	ExternalModules int `json:"external_modules"`

	// HalsteadVolume holds the measure of the size of the test case implementation
	HalsteadVolume float64 `json:"halstead_volume"`

	/* metrics */

	// Maintainability holds the measure of how easy the test case code is to support and modify
	Maintainability float64 `json:"maintainability"`

	/* dynamic analysis recorded during test execution */

	// ReadCount holds the number of times the filesystem had to perform input
	ReadCount int `json:"read_count"`

	// WriteCount holds the number of times the filesystem had to perform output
	WriteCount int `json:"write_count"`

	// MaxMemory holds the peak memory usage
	MaxMemory int `json:"max_memory"`

	// ContextSwitches holds the number of voluntary context switches
	ContextSwitches int `json:"context_switches"`

	// MaxThreads holds the peak number of concurrently running threads
	MaxThreads int `json:"max_threads"`

	// MaxChildren holds the peak number of concurrently running child processes
	MaxChildren int `json:"max_children"`

	/* elapsed times recorded during test execution */

	// RunTime holds the elapsed wall-clock time of the whole test case execution
	RunTime float64 `json:"run_time"`

	// WaitTime holds the elapsed wall-clock time spent waiting for input/output operations to complete
	WaitTime float64 `json:"wait_time"`

	/* test coverage */

	// CoveredLines holds the number of lines covered
	CoveredLines int `json:"covered_lines"`

	// SourceCoveredLines holds the number of lines covered that are not part of test cases
	SourceCoveredLines int `json:"source_covered_lines"`

	/* VCS calculations */

	// CoveredChanges holds the total number of times each covered line has been modified in the last 75 commits
	CoveredChanges int `json:"covered_changes"`
}

// AsFloatVector returns the feature vector with values in the order as described in the
// paper. See ReadCount et al for the ordering references
func (receiver FeatureSet) AsFloatVector() []float64 {
	return []float64{
		float64(receiver.ReadCount),
		float64(receiver.WriteCount),
		float64(receiver.RunTime),
		float64(receiver.WaitTime),
		float64(receiver.ContextSwitches),
		float64(receiver.CoveredLines),
		float64(receiver.SourceCoveredLines),
		float64(receiver.CoveredChanges),
		float64(receiver.MaxThreads),
		float64(receiver.MaxChildren),
		float64(receiver.MaxMemory),
		float64(receiver.ASTDepth),
		float64(receiver.Assertions),
		float64(receiver.ExternalModules),
		float64(receiver.HalsteadVolume),
		float64(receiver.CyclomaticComplexity),
		float64(receiver.TestLinesOfCode),
		float64(receiver.Maintainability),
	}
}

// FromFloatVector creates a feature set extracting the values in the order as described by CANNIER.
// See ReadCount et al for the ordering references
func FromFloatVector(features []float64) FeatureSet {
	return FeatureSet{
		ReadCount:            int(features[ReadCount]),
		WriteCount:           int(features[WriteCount]),
		RunTime:              features[RunTime],
		WaitTime:             features[WaitTime],
		ContextSwitches:      int(features[ContextSwitches]),
		CoveredLines:         int(features[CoveredLines]),
		SourceCoveredLines:   int(features[SourceCoveredLines]),
		CoveredChanges:       int(features[CoveredChanges]),
		MaxThreads:           int(features[MaxThreads]),
		MaxChildren:          int(features[MaxChildren]),
		MaxMemory:            int(features[MaxMemory]),
		ASTDepth:             int(features[ASTDepth]),
		Assertions:           int(features[Assertions]),
		ExternalModules:      int(features[ExternalModules]),
		HalsteadVolume:       features[HalsteadVolume],
		CyclomaticComplexity: int(features[CyclomaticComplexity]),
		TestLinesOfCode:      int(features[TestLinesOfCode]),
		Maintainability:      features[Maintainability],
	}
}

// TestLabel is a label that determines which class a test belongs to
type TestLabel int

const (
	// TestLabelStable describes that the test is stable
	TestLabelStable TestLabel = iota
	// TestLabelFlaky describes that the test has a nondeterministic outcome
	TestLabelFlaky = iota
	// TestLabelUnstable describes that the test is failing
	TestLabelUnstable = iota
)

var testLabels = [3]string{
	"Stable",
	"Flaky",
	"Unstable",
}

func TestLabels() [3]string {
	result := testLabels
	return result
}
