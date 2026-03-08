package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func TestParseSpreadCron(t *testing.T) {
	tests := []struct {
		name     string
		cron     string
		expected CronInfo
		wantErr  bool
	}{
		{
			name: "simple cron",
			cron: "30 7,15,23 * * *",
			expected: CronInfo{
				Minute: 30,
				Hours:  []int{7, 15, 23},
			},
		},
		{
			name: "four times per day",
			cron: "25 0,6,12,18 * * *",
			expected: CronInfo{
				Minute: 25,
				Hours:  []int{0, 6, 12, 18},
			},
		},
		{
			name: "twice per day",
			cron: "30 4,16 * * *",
			expected: CronInfo{
				Minute: 30,
				Hours:  []int{4, 16},
			},
		},
		{
			name:    "invalid cron",
			cron:    "invalid",
			wantErr: true,
		},
		{
			name:    "missing parts",
			cron:    "30 7",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSpreadCron(tt.cron)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSpreadCron() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Minute != tt.expected.Minute {
					t.Errorf("parseSpreadCron() minute = %v, want %v", got.Minute, tt.expected.Minute)
				}
				if len(got.Hours) != len(tt.expected.Hours) {
					t.Errorf("parseSpreadCron() hours length = %v, want %v", len(got.Hours), len(tt.expected.Hours))
					return
				}
				for i := range got.Hours {
					if got.Hours[i] != tt.expected.Hours[i] {
						t.Errorf("parseSpreadCron() hours[%d] = %v, want %v", i, got.Hours[i], tt.expected.Hours[i])
					}
				}
			}
		})
	}
}

func TestGroupByFrequency(t *testing.T) {
	jobs := []JobWithNode{
		{Name: "job1", Cron: CronInfo{Hours: []int{0, 6, 12, 18}}},
		{Name: "job2", Cron: CronInfo{Hours: []int{0, 6, 12, 18}}},
		{Name: "job3", Cron: CronInfo{Hours: []int{7, 15, 23}}},
		{Name: "job4", Cron: CronInfo{Hours: []int{4, 16}}},
		{Name: "job5", Cron: CronInfo{Hours: []int{5}}},
	}

	groups := groupByFrequency(jobs)

	if len(groups) != 4 {
		t.Errorf("groupByFrequency() groups = %v, want 4", len(groups))
	}

	expectedFreqs := []int{4, 3, 2, 1}
	for i, group := range groups {
		if group.Frequency != expectedFreqs[i] {
			t.Errorf("groupByFrequency() group[%d].Frequency = %v, want %v", i, group.Frequency, expectedFreqs[i])
		}
	}

	expectedCounts := map[int]int{
		4: 2,
		3: 1,
		2: 1,
		1: 1,
	}

	for _, group := range groups {
		if len(group.Jobs) != expectedCounts[group.Frequency] {
			t.Errorf("groupByFrequency() frequency %d has %d jobs, want %d",
				group.Frequency, len(group.Jobs), expectedCounts[group.Frequency])
		}
	}
}

func TestSpreadJobs(t *testing.T) {
	tests := []struct {
		name      string
		frequency int
		numJobs   int
		checkFunc func(*testing.T, *JobGroup)
	}{
		{
			name:      "4x per day - 14 jobs",
			frequency: 4,
			numJobs:   14,
			checkFunc: func(t *testing.T, group *JobGroup) {
				expectedStagger := 25

				for i := 0; i < len(group.Jobs)-1; i++ {
					time1 := group.Jobs[i].Cron.Hours[0]*60 + group.Jobs[i].Cron.Minute
					time2 := group.Jobs[i+1].Cron.Hours[0]*60 + group.Jobs[i+1].Cron.Minute
					diff := time2 - time1

					if diff != expectedStagger {
						t.Errorf("Stagger between job %d and %d = %d, want %d",
							i, i+1, diff, expectedStagger)
					}
				}

				for i, job := range group.Jobs {
					if len(job.Cron.Hours) != 4 {
						t.Errorf("Job %d has %d runs per day, want 4", i, len(job.Cron.Hours))
					}

					for j := 0; j < len(job.Cron.Hours)-1; j++ {
						if job.Cron.Hours[j+1]-job.Cron.Hours[j] != 6 {
							t.Errorf("Job %d hour gap between run %d and %d = %d, want 6",
								i, j, j+1, job.Cron.Hours[j+1]-job.Cron.Hours[j])
						}
					}
				}
			},
		},
		{
			name:      "3x per day - 3 jobs",
			frequency: 3,
			numJobs:   3,
			checkFunc: func(t *testing.T, group *JobGroup) {
				expectedStagger := 160

				for i := 0; i < len(group.Jobs)-1; i++ {
					time1 := group.Jobs[i].Cron.Hours[0]*60 + group.Jobs[i].Cron.Minute
					time2 := group.Jobs[i+1].Cron.Hours[0]*60 + group.Jobs[i+1].Cron.Minute
					diff := time2 - time1

					if diff != expectedStagger {
						t.Errorf("Stagger between job %d and %d = %d, want %d",
							i, i+1, diff, expectedStagger)
					}
				}

				for i, job := range group.Jobs {
					if len(job.Cron.Hours) != 3 {
						t.Errorf("Job %d has %d runs per day, want 3", i, len(job.Cron.Hours))
					}
				}
			},
		},
		{
			name:      "2x per day - 4 jobs",
			frequency: 2,
			numJobs:   4,
			checkFunc: func(t *testing.T, group *JobGroup) {
				expectedStagger := 180

				for i := 0; i < len(group.Jobs)-1; i++ {
					time1 := group.Jobs[i].Cron.Hours[0]*60 + group.Jobs[i].Cron.Minute
					time2 := group.Jobs[i+1].Cron.Hours[0]*60 + group.Jobs[i+1].Cron.Minute
					diff := time2 - time1

					if diff != expectedStagger {
						t.Errorf("Stagger between job %d and %d = %d, want %d",
							i, i+1, diff, expectedStagger)
					}
				}

				for i, job := range group.Jobs {
					if len(job.Cron.Hours) != 2 {
						t.Errorf("Job %d has %d runs per day, want 2", i, len(job.Cron.Hours))
					}

					if job.Cron.Hours[1]-job.Cron.Hours[0] != 12 {
						t.Errorf("Job %d hour gap = %d, want 12",
							i, job.Cron.Hours[1]-job.Cron.Hours[0])
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jobs := make([]JobWithNode, tt.numJobs)
			hours := make([]int, tt.frequency)
			period := 24 / tt.frequency
			for i := 0; i < tt.frequency; i++ {
				hours[i] = i * period
			}

			for i := 0; i < tt.numJobs; i++ {
				jobs[i] = JobWithNode{
					Name: "test-job-" + string(rune('a'+i)),
					Cron: CronInfo{
						Minute: 0,
						Hours:  append([]int{}, hours...),
					},
				}
			}

			group := JobGroup{
				Frequency: tt.frequency,
				Jobs:      jobs,
			}

			spreadJobs(os.Stdout, &group, false)

			tt.checkFunc(t, &group)
		})
	}
}

func TestFormatHours(t *testing.T) {
	tests := []struct {
		name     string
		hours    []int
		expected string
	}{
		{
			name:     "single hour",
			hours:    []int{7},
			expected: "7",
		},
		{
			name:     "multiple hours",
			hours:    []int{0, 6, 12, 18},
			expected: "0,6,12,18",
		},
		{
			name:     "three hours",
			hours:    []int{7, 15, 23},
			expected: "7,15,23",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatHours(tt.hours)
			if got != tt.expected {
				t.Errorf("formatHours() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestEndToEnd(t *testing.T) {
	testYAML := `periodics:
- name: periodic-kubevirt-e2e-k8s-1.35-sig-compute-migrations
  cron: 10 3,7,15,23 * * *
- name: periodic-kubevirt-e2e-k8s-1.35-sig-network
  cron: 20 1,7,13,19 * * *
- name: periodic-kubevirt-e2e-k8s-1.35-sig-storage
  cron: 50 3,9,15,21 * * *
- name: periodic-kubevirt-e2e-k8s-1.35-sig-operator
  cron: 10 4,10,16,22 * * *
- name: some-other-job
  cron: 0 0 * * *
`

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-periodics.yaml")

	if err := os.WriteFile(testFile, []byte(testYAML), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Save and restore package-level vars
	origInput := inputFile
	origPattern := pattern
	origOpts := spreadOpts
	defer func() {
		inputFile = origInput
		pattern = origPattern
		spreadOpts = origOpts
	}()

	inputFile = testFile
	pattern = "periodic-kubevirt-e2e-k8s-"
	spreadOpts = spreadOptions{
		outputFile: testFile,
		dryRun:     false,
		verbose:    false,
	}

	testCmd := &cobra.Command{}
	testCmd.SetOut(os.Stdout)
	if err := runSpread(testCmd, nil); err != nil {
		t.Fatalf("runSpread() failed: %v", err)
	}

	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read result: %v", err)
	}

	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	periodicsNode, err := findPeriodicsNode(&root)
	if err != nil {
		t.Fatalf("Failed to find periodics: %v", err)
	}

	matchedJobs := findMatchingJobs(periodicsNode, pattern)
	if len(matchedJobs) != 4 {
		t.Errorf("Expected 4 matched jobs, got %d", len(matchedJobs))
	}

	var foundOtherJob bool
	for _, jobNode := range periodicsNode.Content {
		if jobNode.Kind != yaml.MappingNode {
			continue
		}

		for i := 0; i < len(jobNode.Content); i += 2 {
			if jobNode.Content[i].Value == "name" && jobNode.Content[i+1].Value == "some-other-job" {
				foundOtherJob = true
				for j := 0; j < len(jobNode.Content); j += 2 {
					if jobNode.Content[j].Value == "cron" {
						if jobNode.Content[j+1].Value != "0 0 * * *" {
							t.Errorf("Non-matching job cron changed to: %s", jobNode.Content[j+1].Value)
						}
					}
				}
			}
		}
	}

	if !foundOtherJob {
		t.Error("Non-matching job not found in output")
	}

	startTimes := make([]int, len(matchedJobs))
	for i, job := range matchedJobs {
		startTimes[i] = job.Cron.Hours[0]*60 + job.Cron.Minute
	}

	var sortedTimes []int
	sortedTimes = append(sortedTimes, startTimes...)
	for i := 0; i < len(sortedTimes); i++ {
		for j := i + 1; j < len(sortedTimes); j++ {
			if sortedTimes[i] > sortedTimes[j] {
				sortedTimes[i], sortedTimes[j] = sortedTimes[j], sortedTimes[i]
			}
		}
	}

	// For 4 jobs at 4x/day, stagger should be (6*60)/4 = 90 minutes
	expectedStagger := 90
	for i := 0; i < len(sortedTimes)-1; i++ {
		diff := sortedTimes[i+1] - sortedTimes[i]
		if diff != expectedStagger {
			t.Errorf("Stagger between sorted time %d (%d) and %d (%d) = %d, want %d",
				i, sortedTimes[i], i+1, sortedTimes[i+1], diff, expectedStagger)
		}
	}
}
