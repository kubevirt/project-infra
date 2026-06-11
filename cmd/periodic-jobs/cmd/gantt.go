package cmd

import (
	_ "embed"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/nao1215/markdown/mermaid/gantt"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

//go:embed default-runtimes.yaml
var defaultRuntimesYAML []byte

// runtimeConfig holds the runtime estimates configuration.
type runtimeConfig struct {
	Runtimes map[string]float64 `yaml:"runtimes"`
	Default  float64            `yaml:"default"`
}

type ganttOptions struct {
	runtimesFile string
}

var ganttOpts ganttOptions

// GanttCommand returns the cobra command for the gantt subcommand.
func GanttCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gantt",
		Short: "Generate a Mermaid Gantt chart of periodic job schedules",
		RunE:  runGantt,
	}
	cmd.Flags().StringVar(&ganttOpts.runtimesFile, "runtimes", "",
		"Custom runtimes YAML file (optional, uses embedded defaults)")
	return cmd
}

func loadRuntimes(runtimesFile string) (runtimeConfig, error) {
	var data []byte
	var err error

	if runtimesFile != "" {
		data, err = os.ReadFile(runtimesFile)
		if err != nil {
			return runtimeConfig{}, fmt.Errorf("failed to read runtimes file: %w", err)
		}
	} else {
		data = defaultRuntimesYAML
	}

	var cfg runtimeConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return runtimeConfig{}, fmt.Errorf("failed to parse runtimes YAML: %w", err)
	}

	if cfg.Default == 0 {
		cfg.Default = 2.0
	}

	return cfg, nil
}

// runtimeFor returns the estimated runtime for a sig section.
// Falls back to longest prefix match, then configured default.
func runtimeFor(cfg runtimeConfig, sec string) float64 {
	if rt, ok := cfg.Runtimes[sec]; ok {
		return rt
	}
	best, bestLen := cfg.Default, 0
	for key, rt := range cfg.Runtimes {
		if strings.HasPrefix(sec, key) && len(key) > bestLen {
			best, bestLen = rt, len(key)
		}
	}
	return best
}

// extractParts splits "1.35-ipv6-sig-network" into version="1.35-ipv6", section="sig-network".
func extractParts(trimmed string) (version, section string) {
	idx := strings.Index(trimmed, "-sig-")
	if idx < 0 {
		return trimmed, "other"
	}
	return trimmed[:idx], "sig-" + trimmed[idx+5:]
}

// parseGanttCron returns (hour, minute) pairs from "M H[,H...] * * *".
// Returns nil for wildcard-hour or unparseable crons.
func parseGanttCron(cron string) [][2]int {
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

func runGantt(cmd *cobra.Command, args []string) error {
	rtCfg, err := loadRuntimes(ganttOpts.runtimesFile)
	if err != nil {
		return fmt.Errorf("error loading runtimes: %w", err)
	}

	data, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}
	var cfg prowConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("error parsing yaml: %w", err)
	}

	// Collect runs per section, preserving first-seen section order.
	sections := map[string][]jobRun{}
	var sectionOrder []string

	for _, p := range cfg.Periodics {
		if !strings.HasPrefix(p.Name, pattern) {
			continue
		}
		trimmed := strings.TrimPrefix(p.Name, pattern)
		version, sec := extractParts(trimmed)
		rt := runtimeFor(rtCfg, sec)

		if _, seen := sections[sec]; !seen {
			sectionOrder = append(sectionOrder, sec)
		}
		for _, start := range parseGanttCron(p.Cron) {
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

	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "```mermaid")
	title := fmt.Sprintf("%s* Schedule (24h)", pattern)
	chart := gantt.NewChart(
		out,
		gantt.WithTitle(title),
		gantt.WithDateFormat("HH:mm"),
		gantt.WithAxisFormat("%H:%M"),
		gantt.WithTickInterval("3h"),
	)

	for _, sec := range sectionOrder {
		runs := sections[sec]
		chart.Section(sec)

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
		return fmt.Errorf("build error: %w", err)
	}
	fmt.Fprintln(out, "\n```")
	return nil
}
