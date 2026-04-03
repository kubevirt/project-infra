package cmd

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/robfig/cron/v3"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// Periodic represents a periodic job with its cron expression.
type Periodic struct {
	Name string `yaml:"name"`
	Cron string `yaml:"cron"`
}

// PeriodicConfig represents the root structure.
type PeriodicConfig struct {
	Periodics []yaml.Node `yaml:"periodics"`
}

// CronInfo contains parsed cron information.
type CronInfo struct {
	Minute int
	Hours  []int
}

// JobGroup represents jobs grouped by frequency.
type JobGroup struct {
	Frequency int
	Jobs      []JobWithNode
}

// JobWithNode tracks a job and its YAML node for updating.
type JobWithNode struct {
	Name     string
	Cron     CronInfo
	NodeIdx  int
	CronNode *yaml.Node
}

type spreadOptions struct {
	outputFile string
	dryRun     bool
	verbose    bool
}

var spreadOpts spreadOptions

// SpreadCommand returns the cobra command for the spread subcommand.
func SpreadCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "spread",
		Short: "Spread periodic jobs evenly across time slots",
		RunE:  runSpread,
	}
	cmd.Flags().StringVar(&spreadOpts.outputFile, "output", "",
		"Output YAML file path (defaults to input file)")
	cmd.Flags().BoolVar(&spreadOpts.dryRun, "dry-run", false,
		"Print changes without modifying files")
	cmd.Flags().BoolVar(&spreadOpts.verbose, "verbose", false,
		"Enable verbose output")
	return cmd
}

func runSpread(cmd *cobra.Command, args []string) error {
	out := cmd.OutOrStdout()

	output := spreadOpts.outputFile
	if output == "" {
		output = inputFile
	}

	data, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	periodicsNode, err := findPeriodicsNode(&root)
	if err != nil {
		return err
	}

	matchedJobs := findMatchingJobs(periodicsNode, pattern)
	if len(matchedJobs) == 0 {
		fmt.Fprintf(out, "No jobs found matching pattern: %s\n", pattern)
		return nil
	}

	fmt.Fprintf(out, "Found %d jobs matching pattern '%s'\n", len(matchedJobs), pattern)

	groups := groupByFrequency(matchedJobs)

	if spreadOpts.verbose {
		printGroups(out, groups)
	}

	for i := range groups {
		spreadJobs(out, &groups[i], spreadOpts.verbose)
	}

	jobMap := make(map[string]CronInfo)
	for _, group := range groups {
		for _, job := range group.Jobs {
			jobMap[job.Name] = job.Cron
		}
	}

	for i := range matchedJobs {
		if newCron, ok := jobMap[matchedJobs[i].Name]; ok {
			matchedJobs[i].Cron = newCron
		}
	}

	updateCronNodes(matchedJobs)

	if spreadOpts.dryRun {
		fmt.Fprintf(out, "\nDry run mode - changes not written to file\n")
		printChanges(out, matchedJobs)
		return nil
	}

	marshaledOutput, err := yaml.Marshal(&root)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	if err := os.WriteFile(output, marshaledOutput, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Fprintf(out, "\nSuccessfully updated %s\n", output)
	printChanges(out, matchedJobs)

	return nil
}

func findPeriodicsNode(root *yaml.Node) (*yaml.Node, error) {
	if root.Kind != yaml.DocumentNode || len(root.Content) == 0 {
		return nil, fmt.Errorf("invalid YAML structure")
	}

	mappingNode := root.Content[0]
	if mappingNode.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("expected mapping node")
	}

	for i := 0; i < len(mappingNode.Content); i += 2 {
		keyNode := mappingNode.Content[i]
		valueNode := mappingNode.Content[i+1]

		if keyNode.Value == "periodics" && valueNode.Kind == yaml.SequenceNode {
			return valueNode, nil
		}
	}

	return nil, fmt.Errorf("periodics array not found")
}

func findMatchingJobs(periodicsNode *yaml.Node, jobPattern string) []JobWithNode {
	var jobs []JobWithNode

	for idx, jobNode := range periodicsNode.Content {
		if jobNode.Kind != yaml.MappingNode {
			continue
		}

		var name, cronExpr string
		var cronNode *yaml.Node

		for i := 0; i < len(jobNode.Content); i += 2 {
			key := jobNode.Content[i].Value
			value := jobNode.Content[i+1]

			if key == "name" {
				name = value.Value
			} else if key == "cron" {
				cronExpr = value.Value
				cronNode = value
			}
		}

		if strings.Contains(name, jobPattern) && cronExpr != "" {
			cronInfo, err := parseSpreadCron(cronExpr)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to parse cron for %s: %v\n", name, err)
				continue
			}

			jobs = append(jobs, JobWithNode{
				Name:     name,
				Cron:     cronInfo,
				NodeIdx:  idx,
				CronNode: cronNode,
			})
		}
	}

	return jobs
}

func parseSpreadCron(cronExpr string) (CronInfo, error) {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	_, err := parser.Parse(cronExpr)
	if err != nil {
		return CronInfo{}, fmt.Errorf("invalid cron expression: %w", err)
	}

	parts := strings.Fields(cronExpr)
	if len(parts) < 5 {
		return CronInfo{}, fmt.Errorf("invalid cron format: expected at least 5 fields, got %d", len(parts))
	}

	minute, err := strconv.Atoi(parts[0])
	if err != nil {
		return CronInfo{}, fmt.Errorf("invalid minute field: %w", err)
	}

	if minute < 0 || minute > 59 {
		return CronInfo{}, fmt.Errorf("minute must be 0-59, got %d", minute)
	}

	hourParts := strings.Split(parts[1], ",")
	hours := make([]int, 0, len(hourParts))
	for _, hp := range hourParts {
		h, err := strconv.Atoi(strings.TrimSpace(hp))
		if err != nil {
			return CronInfo{}, fmt.Errorf("invalid hour field: %w", err)
		}
		if h < 0 || h > 23 {
			return CronInfo{}, fmt.Errorf("hour must be 0-23, got %d", h)
		}
		hours = append(hours, h)
	}

	sort.Ints(hours)

	return CronInfo{
		Minute: minute,
		Hours:  hours,
	}, nil
}

func groupByFrequency(jobs []JobWithNode) []JobGroup {
	frequencyMap := make(map[int][]JobWithNode)

	for _, job := range jobs {
		freq := len(job.Cron.Hours)
		frequencyMap[freq] = append(frequencyMap[freq], job)
	}

	for freq := range frequencyMap {
		sort.Slice(frequencyMap[freq], func(i, j int) bool {
			return frequencyMap[freq][i].Name < frequencyMap[freq][j].Name
		})
	}

	var groups []JobGroup
	for freq, jobs := range frequencyMap {
		groups = append(groups, JobGroup{
			Frequency: freq,
			Jobs:      jobs,
		})
	}

	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Frequency > groups[j].Frequency
	})

	return groups
}

func spreadJobs(out io.Writer, group *JobGroup, verbose bool) {
	freq := group.Frequency
	numJobs := len(group.Jobs)

	if numJobs == 0 {
		return
	}

	periodHours := 24 / freq
	staggerMinutes := (periodHours * 60) / numJobs

	fmt.Fprintf(out, "\nSpreading %d jobs at %dx/day (every %dh):\n", numJobs, freq, periodHours)
	fmt.Fprintf(out, "  Stagger interval: %d minutes\n", staggerMinutes)

	for i := range group.Jobs {
		offsetMinutes := i * staggerMinutes
		startHour := offsetMinutes / 60
		startMinute := offsetMinutes % 60

		var hours []int
		for h := startHour; h < 24; h += periodHours {
			hours = append(hours, h)
		}

		group.Jobs[i].Cron.Minute = startMinute
		group.Jobs[i].Cron.Hours = hours

		if verbose {
			fmt.Fprintf(out, "  %s: %02d %s\n",
				group.Jobs[i].Name,
				startMinute,
				formatHours(hours))
		}
	}
}

func formatHours(hours []int) string {
	parts := make([]string, len(hours))
	for i, h := range hours {
		parts[i] = strconv.Itoa(h)
	}
	return strings.Join(parts, ",")
}

func updateCronNodes(jobs []JobWithNode) {
	for _, job := range jobs {
		if job.CronNode != nil {
			newCron := fmt.Sprintf("%d %s * * *", job.Cron.Minute, formatHours(job.Cron.Hours))
			job.CronNode.Value = newCron
		}
	}
}

func printGroups(out io.Writer, groups []JobGroup) {
	fmt.Fprintf(out, "\nJob groups by frequency:\n")
	for _, group := range groups {
		fmt.Fprintf(out, "  %dx/day: %d jobs\n", group.Frequency, len(group.Jobs))
		for _, job := range group.Jobs {
			fmt.Fprintf(out, "    - %s (cron: %d %s * * *)\n",
				job.Name,
				job.Cron.Minute,
				formatHours(job.Cron.Hours))
		}
	}
}

func printChanges(out io.Writer, jobs []JobWithNode) {
	fmt.Fprintf(out, "\nCron expression changes:\n")
	for _, job := range jobs {
		newCron := fmt.Sprintf("%d %s * * *", job.Cron.Minute, formatHours(job.Cron.Hours))
		fmt.Fprintf(out, "  %s: %s\n", job.Name, newCron)
	}
}
