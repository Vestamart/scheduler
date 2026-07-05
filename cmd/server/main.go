package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"sheduler_ozhig/internal/scheduler"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", health)
	mux.HandleFunc("GET /api/sample", sample)
	mux.HandleFunc("GET /api/datasets", datasets)
	mux.HandleFunc("GET /api/datasets/", datasetByName)
	mux.HandleFunc("POST /api/schedule", schedule)

	staticDir := env("STATIC_DIR", defaultStaticDir())
	mux.Handle("/", http.FileServer(http.Dir(staticDir)))

	addr := ":" + env("PORT", "8080")
	log.Printf("schedule annealing server listens on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, logRequest(mux)); err != nil {
		log.Fatal(err)
	}
}

func health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func sample(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, scheduler.SampleDataset())
}

func datasets(w http.ResponseWriter, r *http.Request) {
	dir := env("DATASETS_DIR", defaultDatasetsDir())
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	items := make([]map[string]string, 0, len(files))
	for _, file := range files {
		name := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		items = append(items, map[string]string{
			"id":    name,
			"title": datasetTitle(name),
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i]["id"] < items[j]["id"] })
	writeJSON(w, http.StatusOK, items)
}

func datasetByName(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/api/datasets/")
	if name == "" || strings.Contains(name, "..") || strings.ContainsAny(name, `/\`) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "некорректное имя датасета"})
		return
	}

	data, err := os.ReadFile(filepath.Join(env("DATASETS_DIR", defaultDatasetsDir()), name+".json"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "датасет не найден"})
		return
	}

	var dataset scheduler.Dataset
	if err := json.Unmarshal(data, &dataset); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "датасет поврежден: " + err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, dataset)
}

func schedule(w http.ResponseWriter, r *http.Request) {
	var payload scheduler.ScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "некорректный JSON: " + err.Error()})
		return
	}
	if payload.Options.Seed == 0 {
		payload.Options.Seed = 42
	}
	response, err := scheduler.Compare(payload.Dataset, payload.Options)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("write json: %v", err)
	}
}

func logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func env(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func defaultStaticDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "web/static"
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "web", "static"))
}

func defaultDatasetsDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "data/datasets"
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "data", "datasets"))
}

func datasetTitle(name string) string {
	titles := map[string]string{
		"small_basic":           "Малый базовый",
		"medium_balanced":       "Средний сбалансированный",
		"lab_heavy":             "Много лабораторных",
		"constrained_rooms":     "Дефицит аудиторий",
		"overloaded_impossible": "Перегруженный",
		"conflict_pressure":     "Конфликтная нагрузка",
	}
	if title, ok := titles[name]; ok {
		return title
	}
	return name
}
