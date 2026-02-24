package main

import (
	_ "embed"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/nao1215/markdown/mermaid/gantt"
	"gopkg.in/yaml.v3"
)

//go:embed default-runtimes.yaml
var defaultRuntimesYAML []byte

var (
	inputFile    = flag.String("input", "github/ci/prow-deploy/files/jobs/kubevirt/kubevirt/kubevirt-periodics.yaml", "Input YAML file path")
	pattern      = flag.String("pattern", "periodic-kubevirt-e2e-k8s-", "Job name prefix to match")
	runtimesFile = flag.String("runtimes", "", "Custom runtimes YAML file (optional, uses embedded defaults if not specified)")
)

// RuntimeConfig holds the runtime estimates configuration
type RuntimeConfig struct {
	Runtimes map[string]float64 `yaml:"runtimes"`
	Default  float64            `yaml:"default"`
}

var runtimeConfig RuntimeConfig

// loadRuntimes loads runtime estimates from either custom file or embedded defaults
func loadRuntimes() error {
	var data []byte
	var err error

	if *runtimesFile != "" {
		data, err = os.ReadFile(*runtimesFile)
		if err != nil {
			return fmt.Errorf("failed to read runtimes file: %w", err)
		}
	} else {
		data = defaultRuntimesYAML
	}

	if err := yaml.Unmarshal(data, &runtimeConfig); err != nil {
		return fmt.Errorf("failed to parse runtimes YAML: %w", err)
	}

	// Set default if not specified
	if runtimeConfig.Default == 0 {
		runtimeConfig.Default = 2.0
	}

	return nil
}

// runtimeFor returns the estimated runtime for a sig section.
// Falls back to longest prefix match, then configured default.
func runtimeFor(sec string) float64 {
	if rt, ok := runtimeConfig.Runtimes[sec]; ok {
		return rt
	}
	// prefix match (e.g. "sig-performance-kwok-100" → "sig-performance")
	best, bestLen := runtimeConfig.Default, 0
	for key, rt := range runtimeConfig.Runtimes {
		if strings.HasPrefix(sec, key) && len(key) > bestLen {
			best, bestLen = rt, len(key)
		}
	}
	return best
}

// extractParts splits "1.35-ipv6-sig-network" into version="1.35-ipv6", section="sig-network".
// Input is the job name with jobPrefix already stripped.
func extractParts(trimmed string) (version, section string) {
	idx := strings.Index(trimmed, "-sig-")
	if idx < 0 {
		return trimmed, "other"
	}
	return trimmed[:idx], "sig-" + trimmed[idx+5:]
}

// parseCron returns (hour, minute) pairs from "M H[,H...] * * *".
// Returns nil for wildcard-hour or unparseable crons.
func parseCron(cron string) [][2]int {
	parts := strings.Fields(cron)
	if len(parts) < 2 {
		return nil
	}
	minute, err := strconv.Atoi(parts[0])
	if err != nil || parts[1] == "*" {
		return nil
	}
	var starts [][2]int
	for _, h := range strings.Split(parts[1], ",") {
		hour, err := strconv.Atoi(strings.TrimSpace(h))
		if err != nil {
			continue
		}
		starts = append(starts, [2]int{hour, minute})
	}
	return starts
}

func timeStr(hour, minute int) string {
	return fmt.Sprintf("%02d:%02d", hour, minute)
}

// durationStr converts fractional hours to Mermaid duration format (e.g. "3h30m").
func durationStr(h float64) string {
	total := int(h * 60)
	hh, mm := total/60, total%60
	if mm == 0 {
		return fmt.Sprintf("%dh", hh)
	}
	return fmt.Sprintf("%dh%dm", hh, mm)
}

type prowConfig struct {
	Periodics []struct {
		Name string `yaml:"name"`
		Cron string `yaml:"cron"`
	} `yaml:"periodics"`
}

type jobRun struct {
	version string
	hour    int
	minute  int
	runtime float64
}

func main() {
	flag.Parse()

	// Load runtime estimates
	if err := loadRuntimes(); err != nil {
		fmt.Fprintf(os.Stderr, "error loading runtimes: %v\n", err)
		os.Exit(1)
	}

	data, err := os.ReadFile(*inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading file: %v\n", err)
		os.Exit(1)
	}
	var cfg prowConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "error parsing yaml: %v\n", err)
		os.Exit(1)
	}

	// Collect runs per section, preserving first-seen section order.
	sections := map[string][]jobRun{}
	var sectionOrder []string

	for _, p := range cfg.Periodics {
		if !strings.HasPrefix(p.Name, *pattern) {
			continue
		}
		trimmed := strings.TrimPrefix(p.Name, *pattern)
		version, sec := extractParts(trimmed)
		rt := runtimeFor(sec)

		if _, seen := sections[sec]; !seen {
			sectionOrder = append(sectionOrder, sec)
		}
		for _, start := range parseCron(p.Cron) {
			sections[sec] = append(sections[sec], jobRun{
				version: version,
				hour:    start[0],
				minute:  start[1],
				runtime: rt,
			})
		}
	}

	// Sort runs within each section by start time.
	for sec := range sections {
		runs := sections[sec]
		sort.Slice(runs, func(i, j int) bool {
			return runs[i].hour*60+runs[i].minute < runs[j].hour*60+runs[j].minute
		})
		sections[sec] = runs
	}

	// Sort sections alphabetically for consistent output.
	sort.Slice(sectionOrder, func(i, j int) bool {
		return sectionOrder[i] < sectionOrder[j]
	})

	fmt.Println("```mermaid")
	title := fmt.Sprintf("%s* Schedule (24h)", *pattern)
	chart := gantt.NewChart(
		os.Stdout,
		gantt.WithTitle(title),
		gantt.WithDateFormat("HH:mm"),
		gantt.WithAxisFormat("%H:%M"),
		gantt.WithTickInterval("3h"),
	)

	for _, sec := range sectionOrder {
		runs := sections[sec]
		chart.Section(sec)

		// Count per version so we can number runs when there are multiple.
		vTotal := map[string]int{}
		for _, r := range runs {
			vTotal[r.version]++
		}
		vSeen := map[string]int{}
		for _, r := range runs {
			vSeen[r.version]++
			label := r.version
			if vTotal[r.version] > 1 {
				label = fmt.Sprintf("%s #%d", r.version, vSeen[r.version])
			}
			chart.Task(label, timeStr(r.hour, r.minute), durationStr(r.runtime))
		}
	}

	if err := chart.Build(); err != nil {
		fmt.Fprintf(os.Stderr, "build error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("\n```")
}
