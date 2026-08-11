package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGetMondayOfWeekDate(t *testing.T) {
	tests := []struct {
		year, week int
		want       string
	}{
		{2026, 1, "2025-12-29"},
		{2026, 27, "2026-06-29"},
		{2026, 28, "2026-07-06"},
		{2026, 29, "2026-07-13"},
		{2024, 1, "2024-01-01"},
		{2025, 1, "2024-12-30"},
	}

	for _, tc := range tests {
		got := getMondayOfWeekDate(tc.year, tc.week)
		if got != tc.want {
			t.Fatalf("getMondayOfWeekDate(%d, %d)=%q, want %q", tc.year, tc.week, got, tc.want)
		}

		parsed, err := time.Parse("2006-01-02", got)
		if err != nil {
			t.Fatalf("parse monday: %v", err)
		}
		if parsed.Weekday() != time.Monday {
			t.Fatalf("expected Monday for %q, got %s", got, parsed.Weekday())
		}
		year, week := parsed.ISOWeek()
		if year != tc.year || week != tc.week {
			t.Fatalf("ISOWeek(%q)=(%d,%d), want (%d,%d)", got, year, week, tc.year, tc.week)
		}
	}
}

func TestMergeDataPoints(t *testing.T) {
	t1 := time.Date(2026, 7, 6, 1, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 7, 7, 1, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 7, 8, 1, 0, 0, 0, time.UTC)

	existing := []RecordDataPoint{
		{Value: 10, Date: &t1},
		{Value: 11, Date: &t2},
	}
	incoming := []RecordDataPoint{
		{Value: 21, Date: &t2},
		{Value: 12, Date: &t3},
	}

	merged := mergeDataPoints(existing, incoming)
	if len(merged) != 3 {
		t.Fatalf("expected 3 points, got %d", len(merged))
	}
	if merged[0].Value != 10 || !merged[0].Date.Equal(t1) {
		t.Fatalf("unexpected first point: %+v", merged[0])
	}
	if merged[1].Value != 21 || !merged[1].Date.Equal(t2) {
		t.Fatalf("expected incoming to replace same-timestamp point: %+v", merged[1])
	}
	if merged[2].Value != 12 || !merged[2].Date.Equal(t3) {
		t.Fatalf("unexpected third point: %+v", merged[2])
	}
}

func TestCalculateAVGAndWriteOutputMergesExisting(t *testing.T) {
	tmp := t.TempDir()
	monday := getMondayOfWeekDate(2026, 28)
	outputPath := filepath.Join(tmp, "vmi", "vmiCreationToRunningSecondsP50", monday, "data", "results.json")
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		t.Fatal(err)
	}

	existingDate := time.Date(2026, 7, 6, 8, 0, 0, 0, time.UTC)
	existing := Record{
		StartDate:    monday,
		NumberOfDays: 7,
		Data: NewRecordDateWithAverage([]RecordDataPoint{
			{Value: 10, Date: &existingDate},
		}),
	}
	f, err := os.Create(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.NewEncoder(f).Encode(&existing); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	incomingDate := time.Date(2026, 7, 10, 8, 0, 0, 0, time.UTC)
	results := map[YearWeek][]ResultWithDate{
		{Year: 2026, Week: 28}: {
			{
				Date: &incomingDate,
				Values: map[ResultType]ResultValue{
					"vmiCreationToRunningSecondsP50": {Value: 20},
				},
			},
		},
	}

	if err := calculateAVGAndWriteOutput(results, "vmi", tmp, "vmiCreationToRunningSecondsP50"); err != nil {
		t.Fatal(err)
	}

	merged, err := readRecord(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if merged == nil {
		t.Fatal("expected merged record")
	}
	if len(merged.Data.DataPoints) != 2 {
		t.Fatalf("expected 2 merged points, got %d", len(merged.Data.DataPoints))
	}
	if merged.Data.Average != 15 {
		t.Fatalf("expected average 15, got %v", merged.Data.Average)
	}
}
