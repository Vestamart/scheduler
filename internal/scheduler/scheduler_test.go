package scheduler

import "testing"

func TestAnnealingGeneratesConflictFreeSample(t *testing.T) {
	result, err := Generate(SampleDataset(), AlgorithmAnnealing, ScheduleOptions{Seed: 42, Iterations: 3000})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if result.Stats.ScheduledCount == 0 {
		t.Fatal("expected scheduled entries")
	}
	if result.Stats.ConflictCount != 0 {
		t.Fatalf("expected conflict-free sample, got %d conflicts: %+v", result.Stats.ConflictCount, result.Conflicts)
	}
	if result.Stats.UnscheduledCount != 0 {
		t.Fatalf("expected all sample lessons to be scheduled, got %d unscheduled", result.Stats.UnscheduledCount)
	}
}

func TestCompareReturnsAllAlgorithms(t *testing.T) {
	response, err := Compare(SampleDataset(), ScheduleOptions{Seed: 7, Iterations: 1200})
	if err != nil {
		t.Fatalf("Compare returned error: %v", err)
	}
	if len(response.Comparisons) != 4 {
		t.Fatalf("expected 4 comparison rows, got %d", len(response.Comparisons))
	}
	if response.Annealing.Algorithm != AlgorithmAnnealing {
		t.Fatalf("expected annealing result, got %s", response.Annealing.Algorithm)
	}
}

func TestValidateDatasetRejectsUnknownTeacher(t *testing.T) {
	dataset := SampleDataset()
	dataset.Lessons[0].TeacherID = "missing"
	if err := ValidateDataset(dataset); err == nil {
		t.Fatal("expected validation error")
	}
}
