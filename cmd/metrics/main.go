package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"sheduler_ozhig/internal/scheduler"
)

type row struct {
	Dataset            string                  `json:"dataset"`
	Seed               int64                   `json:"seed"`
	TimeSlots          int                     `json:"timeslots"`
	Rooms              int                     `json:"rooms"`
	Teachers           int                     `json:"teachers"`
	Groups             int                     `json:"groups"`
	Lessons            int                     `json:"lessons"`
	Tasks              int                     `json:"tasks"`
	Algorithm          scheduler.AlgorithmName `json:"algorithm"`
	Title              string                  `json:"title"`
	ScheduledCount     int                     `json:"scheduled_count"`
	UnscheduledCount   int                     `json:"unscheduled_count"`
	ConflictCount      int                     `json:"conflict_count"`
	Score              int                     `json:"score"`
	UtilizationPercent float64                 `json:"utilization_percent"`
	ElapsedMS          float64                 `json:"elapsed_ms"`
}

func main() {
	datasetsDir := flag.String("datasets", "data/datasets", "directory with dataset JSON files")
	runs := flag.Int("runs", 5, "number of seeds per dataset")
	seedStart := flag.Int64("seed-start", 1, "first seed value")
	iterations := flag.Int("iterations", 8000, "annealing iterations")
	initialTemp := flag.Float64("initial-temp", 150, "annealing initial temperature")
	coolingRate := flag.Float64("cooling-rate", 0.997, "annealing cooling rate")
	csvPath := flag.String("csv", "metrics/metrics.csv", "CSV output path")
	summaryPath := flag.String("summary", "metrics/summary.csv", "summary CSV output path, empty to skip")
	jsonPath := flag.String("json", "metrics/metrics.json", "JSON output path, empty to skip")
	flag.Parse()

	files, err := datasetFiles(*datasetsDir)
	if err != nil {
		log.Fatal(err)
	}
	if len(files) == 0 {
		log.Fatalf("no dataset JSON files found in %s", *datasetsDir)
	}

	var rows []row
	for _, file := range files {
		dataset, err := loadDataset(file)
		if err != nil {
			log.Fatalf("load %s: %v", file, err)
		}
		if err := scheduler.ValidateDataset(dataset); err != nil {
			log.Fatalf("validate %s: %v", file, err)
		}

		name := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		for i := 0; i < *runs; i++ {
			seed := *seedStart + int64(i)
			options := scheduler.ScheduleOptions{
				Seed:              seed,
				IncludeComparison: true,
				Iterations:        *iterations,
				InitialTemp:       *initialTemp,
				CoolingRate:       *coolingRate,
			}
			response, err := scheduler.Compare(dataset, options)
			if err != nil {
				log.Fatalf("compare %s seed %d: %v", name, seed, err)
			}
			for _, stats := range response.Comparisons {
				rows = append(rows, rowFromStats(name, seed, dataset, stats))
			}
		}
	}

	if err := writeCSV(*csvPath, rows); err != nil {
		log.Fatal(err)
	}
	if *summaryPath != "" {
		if err := writeSummaryCSV(*summaryPath, rows); err != nil {
			log.Fatal(err)
		}
	}
	if *jsonPath != "" {
		if err := writeJSON(*jsonPath, rows); err != nil {
			log.Fatal(err)
		}
	}
	fmt.Printf("wrote %d metric rows to %s\n", len(rows), *csvPath)
	if *summaryPath != "" {
		fmt.Printf("wrote summary metrics to %s\n", *summaryPath)
	}
	if *jsonPath != "" {
		fmt.Printf("wrote JSON metrics to %s\n", *jsonPath)
	}
}

func datasetFiles(dir string) ([]string, error) {
	pattern := filepath.Join(dir, "*.json")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func loadDataset(path string) (scheduler.Dataset, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return scheduler.Dataset{}, err
	}
	var dataset scheduler.Dataset
	if err := json.Unmarshal(data, &dataset); err != nil {
		return scheduler.Dataset{}, err
	}
	return dataset, nil
}

func rowFromStats(datasetName string, seed int64, dataset scheduler.Dataset, stats scheduler.AlgorithmStats) row {
	return row{
		Dataset:            datasetName,
		Seed:               seed,
		TimeSlots:          len(dataset.TimeSlots),
		Rooms:              len(dataset.Rooms),
		Teachers:           len(dataset.Teachers),
		Groups:             len(dataset.Groups),
		Lessons:            len(dataset.Lessons),
		Tasks:              countTasks(dataset),
		Algorithm:          stats.Algorithm,
		Title:              stats.Title,
		ScheduledCount:     stats.ScheduledCount,
		UnscheduledCount:   stats.UnscheduledCount,
		ConflictCount:      stats.ConflictCount,
		Score:              stats.Score,
		UtilizationPercent: stats.UtilizationPercent,
		ElapsedMS:          stats.ElapsedMS,
	}
}

func countTasks(dataset scheduler.Dataset) int {
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

func writeCSV(path string, rows []row) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{
		"dataset", "seed", "timeslots", "rooms", "teachers", "groups", "lessons", "tasks",
		"algorithm", "title", "scheduled_count", "unscheduled_count", "conflict_count",
		"score", "utilization_percent", "elapsed_ms",
	}
	if err := writer.Write(header); err != nil {
		return err
	}
	for _, item := range rows {
		if err := writer.Write([]string{
			item.Dataset,
			strconv.FormatInt(item.Seed, 10),
			strconv.Itoa(item.TimeSlots),
			strconv.Itoa(item.Rooms),
			strconv.Itoa(item.Teachers),
			strconv.Itoa(item.Groups),
			strconv.Itoa(item.Lessons),
			strconv.Itoa(item.Tasks),
			string(item.Algorithm),
			item.Title,
			strconv.Itoa(item.ScheduledCount),
			strconv.Itoa(item.UnscheduledCount),
			strconv.Itoa(item.ConflictCount),
			strconv.Itoa(item.Score),
			strconv.FormatFloat(item.UtilizationPercent, 'f', 1, 64),
			strconv.FormatFloat(item.ElapsedMS, 'f', 3, 64),
		}); err != nil {
			return err
		}
	}
	return writer.Error()
}

func writeJSON(path string, rows []row) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

type summary struct {
	Dataset              string
	Algorithm            scheduler.AlgorithmName
	Title                string
	Runs                 int
	Tasks                int
	AvgScheduled         float64
	AvgUnscheduled       float64
	AvgConflicts         float64
	AvgScore             float64
	BestScore            int
	WorstScore           int
	AvgUtilization       float64
	AvgElapsedMS         float64
	ConflictFreeRunShare float64
}

func writeSummaryCSV(path string, rows []row) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	items := summarize(rows)
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{
		"dataset", "algorithm", "title", "runs", "tasks", "avg_scheduled",
		"avg_unscheduled", "avg_conflicts", "avg_score", "best_score",
		"worst_score", "avg_utilization_percent", "avg_elapsed_ms",
		"conflict_free_run_share",
	}
	if err := writer.Write(header); err != nil {
		return err
	}
	for _, item := range items {
		if err := writer.Write([]string{
			item.Dataset,
			string(item.Algorithm),
			item.Title,
			strconv.Itoa(item.Runs),
			strconv.Itoa(item.Tasks),
			formatFloat(item.AvgScheduled),
			formatFloat(item.AvgUnscheduled),
			formatFloat(item.AvgConflicts),
			formatFloat(item.AvgScore),
			strconv.Itoa(item.BestScore),
			strconv.Itoa(item.WorstScore),
			formatFloat(item.AvgUtilization),
			formatFloat(item.AvgElapsedMS),
			formatFloat(item.ConflictFreeRunShare),
		}); err != nil {
			return err
		}
	}
	return writer.Error()
}

func summarize(rows []row) []summary {
	type acc struct {
		summary
		scheduledSum     int
		unscheduledSum   int
		conflictSum      int
		scoreSum         int
		utilizationSum   float64
		elapsedSum       float64
		conflictFreeRuns int
	}

	groups := map[string]*acc{}
	for _, item := range rows {
		key := item.Dataset + "|" + string(item.Algorithm)
		if groups[key] == nil {
			groups[key] = &acc{
				summary: summary{
					Dataset:    item.Dataset,
					Algorithm:  item.Algorithm,
					Title:      item.Title,
					Tasks:      item.Tasks,
					BestScore:  item.Score,
					WorstScore: item.Score,
				},
			}
		}
		group := groups[key]
		group.Runs++
		group.scheduledSum += item.ScheduledCount
		group.unscheduledSum += item.UnscheduledCount
		group.conflictSum += item.ConflictCount
		group.scoreSum += item.Score
		group.utilizationSum += item.UtilizationPercent
		group.elapsedSum += item.ElapsedMS
		if item.ConflictCount == 0 {
			group.conflictFreeRuns++
		}
		if item.Score < group.BestScore {
			group.BestScore = item.Score
		}
		if item.Score > group.WorstScore {
			group.WorstScore = item.Score
		}
	}

	items := make([]summary, 0, len(groups))
	for _, group := range groups {
		runs := float64(group.Runs)
		group.AvgScheduled = float64(group.scheduledSum) / runs
		group.AvgUnscheduled = float64(group.unscheduledSum) / runs
		group.AvgConflicts = float64(group.conflictSum) / runs
		group.AvgScore = float64(group.scoreSum) / runs
		group.AvgUtilization = group.utilizationSum / runs
		group.AvgElapsedMS = group.elapsedSum / runs
		group.ConflictFreeRunShare = float64(group.conflictFreeRuns) / runs
		items = append(items, group.summary)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Dataset != items[j].Dataset {
			return items[i].Dataset < items[j].Dataset
		}
		return items[i].Algorithm < items[j].Algorithm
	})
	return items
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', 3, 64)
}
