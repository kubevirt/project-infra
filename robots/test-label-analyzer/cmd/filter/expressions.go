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

package filter

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"

	ginkgotypes "github.com/onsi/ginkgo/v2/types"
	"github.com/spf13/cobra"
)

type filterExpressionsOptions struct {
	inputFilePath string
	mode          string
}

var filterExpressionsOpts = &filterExpressionsOptions{}

// expressionsCmd generates ready-to-use Ginkgo filter expressions from a matching-tests JSON file.
//
// It is intended to be used after `test-label-analyzer filter stats matching-tests`, taking its
// JSON output as input and emitting filter expressions for both Ginkgo v1 text filters and
// Ginkgo v2 label filters.
var expressionsCmd = &cobra.Command{
	Use:   "expressions <matching-tests.json>",
	Short: "Generates Ginkgo filter expressions from matching tests",
	Long: `Generates expressions that can be used with Ginkgo's --filter/--skip (v1) and --label-filter (v2) flags.

The command expects as input the JSON produced by 'test-label-analyzer filter stats matching-tests'.
By default it prints a simple text representation with the expressions.
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filterExpressionsOpts.inputFilePath = args[0]
		return runExpressions(filterExpressionsOpts, cmd.OutOrStdout())
	},
}

// expressionsOutput represents the combined filter expressions for Ginkgo v1 and v2.
type expressionsOutput struct {
	// Filter is suitable for use with Ginkgo v1 --filter
	Filter string `json:"filter"`

	// Skip is suitable for use with Ginkgo v1 --skip
	Skip string `json:"skip"`

	// LabelFilter is suitable for use with Ginkgo v2 --label-filter when reasons are valid labels
	LabelFilter string `json:"label_filter"`
}

func runExpressions(opts *filterExpressionsOptions, outWriter io.Writer) error {
	fileContent, err := os.ReadFile(opts.inputFilePath)
	if err != nil {
		return fmt.Errorf("failed to read input file %q: %w", opts.inputFilePath, err)
	}

	var matches matchingTests
	if err := json.Unmarshal(fileContent, &matches); err != nil {
		return fmt.Errorf("failed to unmarshal input file %q: %w", opts.inputFilePath, err)
	}

	if len(matches) == 0 {
		return fmt.Errorf("no matching tests found in input")
	}

	out, err := buildExpressions(matches)
	if err != nil {
		return err
	}

	switch strings.ToLower(opts.mode) {
	case "", "text":
		fmt.Fprintf(outWriter, "# Ginkgo v1\n")
		if out.Filter != "" {
			fmt.Fprintf(outWriter, "--filter=%s\n", out.Filter)
		}
		if out.Skip != "" {
			fmt.Fprintf(outWriter, "--skip=%s\n", out.Skip)
		}
		fmt.Fprintf(outWriter, "\n# Ginkgo v2\n")
		if out.LabelFilter != "" {
			fmt.Fprintf(outWriter, "--label-filter=%s\n", out.LabelFilter)
		}
	case "json":
		enc := json.NewEncoder(outWriter)
		enc.SetIndent("", "  ")
		if err := enc.Encode(out); err != nil {
			return fmt.Errorf("failed to marshal expressions output: %w", err)
		}
	default:
		return fmt.Errorf("unsupported output mode %q, supported modes: text, json", opts.mode)
	}

	return nil
}

// buildExpressions deduplicates the matching tests and constructs simple OR-based expressions.
func buildExpressions(matches matchingTests) (expressionsOutput, error) {
	// For now we keep things deliberately simple:
	// - v1 filter: OR of all test names as literal substrings
	// - v1 skip: left empty (can be extended later)
	// - v2 label-filter: OR of all unique reasons, if they are already valid Ginkgo labels

	testNameSet := map[string]struct{}{}
	reasonSet := map[string]struct{}{}

	for _, m := range matches {
		if m.TestName != "" {
			testNameSet[m.TestName] = struct{}{}
		}
		if m.Reason != "" {
			reasonSet[m.Reason] = struct{}{}
		}
	}

	testNames := make([]string, 0, len(testNameSet))
	for name := range testNameSet {
		testNames = append(testNames, name)
	}
	sort.Strings(testNames)

	reasons := make([]string, 0, len(reasonSet))
	for r := range reasonSet {
		reasons = append(reasons, r)
	}
	sort.Strings(reasons)

	var filterExpr string
	if len(testNames) > 0 {
		parts := make([]string, 0, len(testNames))
		for _, name := range testNames {
			parts = append(parts, regexp.QuoteMeta(name))
		}
		filterExpr = strings.Join(parts, "|")
	}

	var labelExpr string
	if len(reasons) > 0 {
		parts := make([]string, 0, len(reasons))
		for _, r := range reasons {
			cleaned, err := ginkgotypes.ValidateAndCleanupLabel(r, ginkgotypes.CodeLocation{})
			if err != nil {
				return expressionsOutput{}, fmt.Errorf("cannot generate --label-filter from reason %q: %w", r, err)
			}
			parts = append(parts, cleaned)
		}
		labelExpr = strings.Join(parts, "||")
	}

	return expressionsOutput{
		Filter:      filterExpr,
		Skip:        "",
		LabelFilter: labelExpr,
	}, nil
}

func init() {
	rootCmd.AddCommand(expressionsCmd)
	expressionsCmd.PersistentFlags().StringVar(&filterExpressionsOpts.mode, "output-mode", "text", "output mode: text or json")
}
