package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/robfig/cron/v3"
	"gopkg.in/yaml.v3"
)

// Periodic represents a periodic job with its cron expression
type Periodic struct {
	Name string `yaml:"name"`
	Cron string `yaml:"cron"`
}

// PeriodicConfig represents the root structure
type PeriodicConfig struct {
	Periodics []yaml.Node `yaml:"periodics"`
}

// CronInfo contains parsed cron information
type CronInfo struct {
	Minute int
	Hours  []int
}

// JobGroup represents jobs grouped by frequency
type JobGroup struct {
	Frequency int // times per day
	Jobs      []JobWithNode
}

// JobWithNode tracks a job and its YAML node for updating
type JobWithNode struct {
	Name     string
	Cron     CronInfo
	NodeIdx  int
	CronNode *yaml.Node
}

var (
	inputFile  = flag.String("input", "", "Input YAML file path")
	outputFile = flag.String("output", "", "Output YAML file path (defaults to input file)")
	pattern    = flag.String("pattern", "periodic-kubevirt-e2e-k8s-", "Job name pattern to match")
	dryRun     = flag.Bool("dry-run", false, "Print changes without modifying files")
	verbose    = flag.Bool("verbose", false, "Enable verbose output")
)

func main() {
	flag.Parse()

	if *inputFile == "" {
		log.Fatal("--input flag is required")
	}

	if *outputFile == "" {
		*outputFile = *inputFile
	}

	if err := run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func run() error {
	// Read the YAML file
	data, err := os.ReadFile(*inputFile)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Parse the YAML while preserving structure
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Find the periodics array
	periodicsNode, err := findPeriodicsNode(&root)
	if err != nil {
		return err
	}

	// Find matching jobs
	matchedJobs := findMatchingJobs(periodicsNode, *pattern)
	if len(matchedJobs) == 0 {
		log.Printf("No jobs found matching pattern: %s", *pattern)
		return nil
	}

	log.Printf("Found %d jobs matching pattern '%s'", len(matchedJobs), *pattern)

	// Group jobs by frequency
	groups := groupByFrequency(matchedJobs)

	if *verbose {
		printGroups(groups)
	}

	// Apply spreading algorithm to each group
	// This modifies the jobs in place
	for i := range groups {
		spreadJobs(&groups[i])
	}

	// Gather all updated jobs from groups back into matchedJobs
	// We need to update the original matchedJobs with the spread times
	jobMap := make(map[string]CronInfo)
	for _, group := range groups {
		for _, job := range group.Jobs {
			jobMap[job.Name] = job.Cron
		}
	}

	// Update the matched jobs with the new cron info
	for i := range matchedJobs {
		if newCron, ok := jobMap[matchedJobs[i].Name]; ok {
			matchedJobs[i].Cron = newCron
		}
	}

	// Update the YAML nodes with new cron expressions
	updateCronNodes(matchedJobs)

	if *dryRun {
		log.Println("\nDry run mode - changes not written to file")
		printChanges(matchedJobs)
		return nil
	}

	// Write back the YAML
	output, err := yaml.Marshal(&root)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	if err := os.WriteFile(*outputFile, output, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	log.Printf("\nSuccessfully updated %s", *outputFile)
	printChanges(matchedJobs)

	return nil
}

func findPeriodicsNode(root *yaml.Node) (*yaml.Node, error) {
	// Root is a document node, get its content (mapping node)
	if root.Kind != yaml.DocumentNode || len(root.Content) == 0 {
		return nil, fmt.Errorf("invalid YAML structure")
	}

	mappingNode := root.Content[0]
	if mappingNode.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("expected mapping node")
	}

	// Find "periodics" key
	for i := 0; i < len(mappingNode.Content); i += 2 {
		keyNode := mappingNode.Content[i]
		valueNode := mappingNode.Content[i+1]

		if keyNode.Value == "periodics" && valueNode.Kind == yaml.SequenceNode {
			return valueNode, nil
		}
	}

	return nil, fmt.Errorf("periodics array not found")
}

func findMatchingJobs(periodicsNode *yaml.Node, pattern string) []JobWithNode {
	var jobs []JobWithNode

	for idx, jobNode := range periodicsNode.Content {
		if jobNode.Kind != yaml.MappingNode {
			continue
		}

		var name, cron string
		var cronNode *yaml.Node

		// Extract name and cron from this job
		for i := 0; i < len(jobNode.Content); i += 2 {
			key := jobNode.Content[i].Value
			value := jobNode.Content[i+1]

			if key == "name" {
				name = value.Value
			} else if key == "cron" {
				cron = value.Value
				cronNode = value
			}
		}

		// Check if job matches pattern
		if strings.Contains(name, pattern) && cron != "" {
			cronInfo, err := parseCron(cron)
			if err != nil {
				log.Printf("Warning: failed to parse cron for %s: %v", name, err)
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

func parseCron(cronExpr string) (CronInfo, error) {
	// Validate the cron expression using robfig/cron library
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	_, err := parser.Parse(cronExpr)
	if err != nil {
		return CronInfo{}, fmt.Errorf("invalid cron expression: %w", err)
	}

	// Extract minute and hours by simple string parsing
	parts := strings.Fields(cronExpr)
	if len(parts) < 5 {
		return CronInfo{}, fmt.Errorf("invalid cron format: expected at least 5 fields, got %d", len(parts))
	}

	// Parse minute (first field)
	minute, err := strconv.Atoi(parts[0])
	if err != nil {
		return CronInfo{}, fmt.Errorf("invalid minute field: %w", err)
	}

	// Validate minute range
	if minute < 0 || minute > 59 {
		return CronInfo{}, fmt.Errorf("minute must be 0-59, got %d", minute)
	}

	// Parse hours (second field, can be comma-separated like "1,7,13,19")
	hourParts := strings.Split(parts[1], ",")
	hours := make([]int, 0, len(hourParts))
	for _, hp := range hourParts {
		h, err := strconv.Atoi(strings.TrimSpace(hp))
		if err != nil {
			return CronInfo{}, fmt.Errorf("invalid hour field: %w", err)
		}
		// Validate hour range
		if h < 0 || h > 23 {
			return CronInfo{}, fmt.Errorf("hour must be 0-23, got %d", h)
		}
		hours = append(hours, h)
	}

	// Sort hours for consistent output
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

	// Sort jobs within each frequency group by name for deterministic output
	for freq := range frequencyMap {
		sort.Slice(frequencyMap[freq], func(i, j int) bool {
			return frequencyMap[freq][i].Name < frequencyMap[freq][j].Name
		})
	}

	// Convert to slice and sort by frequency
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

func spreadJobs(group *JobGroup) {
	freq := group.Frequency
	numJobs := len(group.Jobs)

	if numJobs == 0 {
		return
	}

	// Calculate the period (hours between each run)
	periodHours := 24 / freq

	// Calculate stagger interval in minutes
	// We want to spread jobs evenly across one period
	staggerMinutes := (periodHours * 60) / numJobs

	log.Printf("\nSpreading %d jobs at %dx/day (every %dh):", numJobs, freq, periodHours)
	log.Printf("  Stagger interval: %d minutes", staggerMinutes)

	// Assign new cron times
	for i := range group.Jobs {
		// Calculate the offset in minutes from the start of each period
		offsetMinutes := i * staggerMinutes

		// Convert to hour and minute
		startHour := offsetMinutes / 60
		startMinute := offsetMinutes % 60

		// Generate the hours list (repeating every period)
		var hours []int
		for h := startHour; h < 24; h += periodHours {
			hours = append(hours, h)
		}

		// Update the job's cron info
		group.Jobs[i].Cron.Minute = startMinute
		group.Jobs[i].Cron.Hours = hours

		if *verbose {
			log.Printf("  %s: %02d %s",
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

func printGroups(groups []JobGroup) {
	log.Println("\nJob groups by frequency:")
	for _, group := range groups {
		log.Printf("  %dx/day: %d jobs", group.Frequency, len(group.Jobs))
		for _, job := range group.Jobs {
			log.Printf("    - %s (cron: %d %s * * *)",
				job.Name,
				job.Cron.Minute,
				formatHours(job.Cron.Hours))
		}
	}
}

func printChanges(jobs []JobWithNode) {
	log.Println("\nCron expression changes:")
	for _, job := range jobs {
		newCron := fmt.Sprintf("%d %s * * *", job.Cron.Minute, formatHours(job.Cron.Hours))
		log.Printf("  %s: %s", job.Name, newCron)
	}
}
