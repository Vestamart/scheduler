package scheduler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestBundledDatasetsAreValidAndSchedulable(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "..", "data", "datasets", "*.json"))
	if err != nil {
		t.Fatalf("glob datasets: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("expected bundled datasets")
	}

	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			data, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("read dataset: %v", err)
			}
			var dataset Dataset
			if err := json.Unmarshal(data, &dataset); err != nil {
				t.Fatalf("decode dataset: %v", err)
			}
			if err := ValidateDataset(dataset); err != nil {
				t.Fatalf("validate dataset: %v", err)
			}

			response, err := Compare(dataset, ScheduleOptions{Seed: 11, Iterations: 1500})
			if err != nil {
				t.Fatalf("compare algorithms: %v", err)
			}
			if len(response.Comparisons) != 4 {
				t.Fatalf("expected 4 algorithms, got %d", len(response.Comparisons))
			}
			for _, stats := range response.Comparisons {
				if stats.ConflictCount != 0 {
					t.Fatalf("%s produced conflicts: %+v", stats.Algorithm, stats)
				}
				if stats.ScheduledCount+stats.UnscheduledCount != countDatasetTasks(dataset) {
					t.Fatalf("%s has inconsistent task counts: %+v", stats.Algorithm, stats)
				}
			}
		})
	}
}

func countDatasetTasks(dataset Dataset) int {
	total := 0
	for _, lesson := range dataset.Lessons {
		if lesson.Sessions <= 0 {
			total++
			continue
		}
		total += lesson.Sessions
	}
	return total
}
