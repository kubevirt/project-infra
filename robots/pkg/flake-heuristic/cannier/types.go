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

func (receiver FeatureSet) AsFloatVector() []float64 {
	return []float64{
		float64(receiver.ReadCount),            // 1 Read Count
		float64(receiver.WriteCount),           // 2 Write Count
		float64(receiver.RunTime),              // 3 Run Time
		float64(receiver.WaitTime),             // 4 Wait Time
		float64(receiver.ContextSwitches),      // 5 Context Switches
		float64(receiver.CoveredLines),         // 6 Covered Lines
		float64(receiver.SourceCoveredLines),   // 7 Source Covered Lines
		float64(receiver.CoveredChanges),       // 8 Covered Changes
		float64(receiver.MaxThreads),           // 9 Max. Threads
		float64(receiver.MaxChildren),          // 10 Max. Children
		float64(receiver.ASTDepth),             // 11 Max. Memory
		float64(receiver.Assertions),           // 12 AST Depth
		float64(receiver.ExternalModules),      // 13 Assertions
		float64(receiver.HalsteadVolume),       // 14 External Modules
		float64(receiver.CyclomaticComplexity), // 15 Halstead Volume
		float64(receiver.TestLinesOfCode),      // 16 Cyclomatic Complexity
		float64(receiver.Maintainability),      // 17 Test Lines of Code
	}
}
