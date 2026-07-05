package scheduler

import (
	"math"
	"math/rand"
	"sort"
	"time"
)

var algorithmTitles = map[AlgorithmName]string{
	AlgorithmAnnealing:  "Имитация отжига",
	AlgorithmGreedy:     "Жадный алгоритм",
	AlgorithmSequential: "Последовательный перебор",
	AlgorithmRandom:     "Случайный порядок",
}

type task struct {
	ID           string
	Lesson       Lesson
	SessionIndex int
	GroupSize    int
}

type candidate struct {
	Slot TimeSlot
	Room Room
}

type assignment struct {
	Task      task
	Candidate candidate
	Placed    bool
}

func Generate(dataset Dataset, algorithm AlgorithmName, options ScheduleOptions) (ScheduleResult, error) {
	start := time.Now()
	if err := ValidateDataset(dataset); err != nil {
		return ScheduleResult{}, err
	}
	if options.Seed == 0 {
		options.Seed = 42
	}
	if options.Iterations <= 0 {
		options.Iterations = 8000
	}
	if options.InitialTemp <= 0 {
		options.InitialTemp = 150
	}
	if options.CoolingRate <= 0 || options.CoolingRate >= 1 {
		options.CoolingRate = 0.997
	}

	var entries []ScheduleEntry
	switch algorithm {
	case AlgorithmAnnealing:
		entries = anneal(dataset, options)
	case AlgorithmSequential:
		entries = greedyLike(dataset, AlgorithmSequential, options.Seed)
	case AlgorithmRandom:
		entries = greedyLike(dataset, AlgorithmRandom, options.Seed)
	default:
		algorithm = AlgorithmGreedy
		entries = greedyLike(dataset, AlgorithmGreedy, options.Seed)
	}

	conflicts := ValidateSchedule(dataset, entries)
	unscheduled := unscheduledItems(dataset, entries)
	score := scoreEntries(dataset, entries)
	result := ScheduleResult{
		Algorithm:   algorithm,
		Title:       algorithmTitles[algorithm],
		Entries:     entries,
		Unscheduled: unscheduled,
		Conflicts:   conflicts,
	}
	result.Stats = AlgorithmStats{
		Algorithm:          algorithm,
		Title:              result.Title,
		ScheduledCount:     len(entries),
		UnscheduledCount:   len(unscheduled),
		ConflictCount:      len(conflicts),
		UtilizationPercent: utilization(dataset, entries),
		Score:              score,
		ElapsedMS:          float64(time.Since(start).Microseconds()) / 1000,
	}
	return result, nil
}

func Compare(dataset Dataset, options ScheduleOptions) (ScheduleResponse, error) {
	annealing, err := Generate(dataset, AlgorithmAnnealing, options)
	if err != nil {
		return ScheduleResponse{}, err
	}
	greedy, err := Generate(dataset, AlgorithmGreedy, options)
	if err != nil {
		return ScheduleResponse{}, err
	}
	sequential, err := Generate(dataset, AlgorithmSequential, options)
	if err != nil {
		return ScheduleResponse{}, err
	}
	randomResult, err := Generate(dataset, AlgorithmRandom, options)
	if err != nil {
		return ScheduleResponse{}, err
	}
	return ScheduleResponse{
		Annealing: annealing,
		Greedy:    greedy,
		Comparisons: []AlgorithmStats{
			annealing.Stats,
			greedy.Stats,
			sequential.Stats,
			randomResult.Stats,
		},
	}, nil
}

func anneal(dataset Dataset, options ScheduleOptions) []ScheduleEntry {
	rng := rand.New(rand.NewSource(options.Seed))
	tasks := buildTasks(dataset)
	candidates := candidatesByTask(dataset, tasks)
	current := initialAssignments(tasks, candidates, rng)
	currentScore := scoreAssignments(dataset, current)
	best := cloneAssignments(current)
	bestScore := currentScore
	temp := options.InitialTemp

	for i := 0; i < options.Iterations; i++ {
		next := cloneAssignments(current)
		idx := rng.Intn(len(next))
		taskCandidates := candidates[next[idx].Task.ID]
		if len(taskCandidates) == 0 || rng.Intn(len(taskCandidates)+1) == 0 {
			next[idx].Placed = false
		} else {
			next[idx].Candidate = taskCandidates[rng.Intn(len(taskCandidates))]
			next[idx].Placed = true
		}
		nextScore := scoreAssignments(dataset, next)
		delta := nextScore - currentScore
		if delta <= 0 || rng.Float64() < math.Exp(float64(-delta)/temp) {
			current = next
			currentScore = nextScore
			if nextScore < bestScore {
				best = cloneAssignments(next)
				bestScore = nextScore
			}
		}
		temp *= options.CoolingRate
		if temp < 0.01 {
			temp = 0.01
		}
	}
	return entriesFromAssignments(dataset, best)
}

func greedyLike(dataset Dataset, algorithm AlgorithmName, seed int64) []ScheduleEntry {
	rng := rand.New(rand.NewSource(seed))
	ctx := newContext(dataset)
	tasks := buildTasks(dataset)
	candidates := candidatesByTask(dataset, tasks)
	if algorithm == AlgorithmGreedy {
		sort.Slice(tasks, func(i, j int) bool {
			left := len(candidates[tasks[i].ID])
			right := len(candidates[tasks[j].ID])
			if left != right {
				return left < right
			}
			if tasks[i].Lesson.Priority != tasks[j].Lesson.Priority {
				return tasks[i].Lesson.Priority > tasks[j].Lesson.Priority
			}
			return tasks[i].GroupSize > tasks[j].GroupSize
		})
	}
	if algorithm == AlgorithmRandom {
		rng.Shuffle(len(tasks), func(i, j int) { tasks[i], tasks[j] = tasks[j], tasks[i] })
	}

	var entries []ScheduleEntry
	roomBusy := map[string]bool{}
	teacherBusy := map[string]bool{}
	groupBusy := map[string]bool{}
	for _, t := range tasks {
		list := append([]candidate(nil), candidates[t.ID]...)
		if algorithm == AlgorithmRandom {
			rng.Shuffle(len(list), func(i, j int) { list[i], list[j] = list[j], list[i] })
		}
		if algorithm == AlgorithmGreedy {
			sort.Slice(list, func(i, j int) bool {
				wi := list[i].Room.Capacity - t.GroupSize
				wj := list[j].Room.Capacity - t.GroupSize
				if wi != wj {
					return wi < wj
				}
				return list[i].Slot.Order < list[j].Slot.Order
			})
		}
		for _, c := range list {
			if roomBusy[c.Room.ID+"|"+c.Slot.ID] || teacherBusy[t.Lesson.TeacherID+"|"+c.Slot.ID] {
				continue
			}
			blocked := false
			for _, groupID := range t.Lesson.GroupIDs {
				if groupBusy[groupID+"|"+c.Slot.ID] {
					blocked = true
					break
				}
			}
			if blocked {
				continue
			}
			roomBusy[c.Room.ID+"|"+c.Slot.ID] = true
			teacherBusy[t.Lesson.TeacherID+"|"+c.Slot.ID] = true
			for _, groupID := range t.Lesson.GroupIDs {
				groupBusy[groupID+"|"+c.Slot.ID] = true
			}
			entries = append(entries, entryFromCandidate(ctx, t, c))
			break
		}
	}
	return entries
}

func buildTasks(dataset Dataset) []task {
	ctx := newContext(dataset)
	tasks := []task{}
	for _, lesson := range dataset.Lessons {
		sessions := lesson.Sessions
		if sessions <= 0 {
			sessions = 1
		}
		for i := 1; i <= sessions; i++ {
			tasks = append(tasks, task{ID: lesson.ID + "-" + itoa(i), Lesson: lesson, SessionIndex: i, GroupSize: groupSize(ctx, lesson.GroupIDs)})
		}
	}
	return tasks
}

func candidatesByTask(dataset Dataset, tasks []task) map[string][]candidate {
	ctx := newContext(dataset)
	slots := sortedSlots(dataset)
	rooms := sortedRooms(dataset)
	out := map[string][]candidate{}
	for _, t := range tasks {
		teacher := ctx.teachers[t.Lesson.TeacherID]
		groups := make([]Group, 0, len(t.Lesson.GroupIDs))
		for _, groupID := range t.Lesson.GroupIDs {
			groups = append(groups, ctx.groups[groupID])
		}
		for _, slot := range slots {
			if contains(teacher.Unavailable, slot.ID) {
				continue
			}
			groupUnavailable := false
			for _, group := range groups {
				if contains(group.Unavailable, slot.ID) {
					groupUnavailable = true
					break
				}
			}
			if groupUnavailable {
				continue
			}
			for _, room := range rooms {
				if room.Capacity >= t.GroupSize && roomMatches(t.Lesson.RoomType, room.RoomType) {
					out[t.ID] = append(out[t.ID], candidate{Slot: slot, Room: room})
				}
			}
		}
	}
	return out
}

func initialAssignments(tasks []task, candidates map[string][]candidate, rng *rand.Rand) []assignment {
	out := make([]assignment, len(tasks))
	for i, t := range tasks {
		list := candidates[t.ID]
		out[i] = assignment{Task: t}
		if len(list) > 0 {
			out[i].Candidate = list[rng.Intn(len(list))]
			out[i].Placed = true
		}
	}
	return out
}

func cloneAssignments(input []assignment) []assignment {
	out := make([]assignment, len(input))
	copy(out, input)
	return out
}

func entriesFromAssignments(dataset Dataset, assignments []assignment) []ScheduleEntry {
	ctx := newContext(dataset)
	entries := []ScheduleEntry{}
	for _, item := range assignments {
		if !item.Placed {
			continue
		}
		entries = append(entries, entryFromCandidate(ctx, item.Task, item.Candidate))
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Day != entries[j].Day {
			return entries[i].Day < entries[j].Day
		}
		if entries[i].Start != entries[j].Start {
			return entries[i].Start < entries[j].Start
		}
		return entries[i].Room < entries[j].Room
	})
	return entries
}

func entryFromCandidate(ctx context, t task, c candidate) ScheduleEntry {
	teacher := ctx.teachers[t.Lesson.TeacherID]
	groups := make([]string, 0, len(t.Lesson.GroupIDs))
	for _, groupID := range t.Lesson.GroupIDs {
		groups = append(groups, ctx.groups[groupID].Name)
	}
	return ScheduleEntry{
		ID:           t.ID,
		LessonID:     t.Lesson.ID,
		Subject:      t.Lesson.Subject,
		SessionIndex: t.SessionIndex,
		TeacherID:    t.Lesson.TeacherID,
		Teacher:      teacher.Name,
		GroupIDs:     append([]string(nil), t.Lesson.GroupIDs...),
		Groups:       groups,
		RoomID:       c.Room.ID,
		Room:         c.Room.Name,
		TimeSlotID:   c.Slot.ID,
		Day:          c.Slot.Day,
		Start:        c.Slot.Start,
		End:          c.Slot.End,
	}
}

func scoreAssignments(dataset Dataset, assignments []assignment) int {
	return scoreEntries(dataset, entriesFromAssignments(dataset, assignments))
}

func scoreEntries(dataset Dataset, entries []ScheduleEntry) int {
	conflicts := ValidateSchedule(dataset, entries)
	tasks := buildTasks(dataset)
	score := (len(tasks) - len(entries)) * 100000
	score += len(conflicts) * 200000
	ctx := newContext(dataset)
	lessonDays := map[string]map[string]bool{}
	groupDayLoad := map[string]int{}
	teacherDayLoad := map[string]int{}
	for _, entry := range entries {
		room := ctx.rooms[entry.RoomID]
		score += max(0, room.Capacity-groupSize(ctx, entry.GroupIDs))
		if lessonDays[entry.LessonID] == nil {
			lessonDays[entry.LessonID] = map[string]bool{}
		}
		if lessonDays[entry.LessonID][entry.Day] {
			score += 35
		}
		lessonDays[entry.LessonID][entry.Day] = true
		groupDayLoad[entry.Day+"|"+entry.Groups[0]]++
		teacherDayLoad[entry.Day+"|"+entry.TeacherID]++
	}
	for _, load := range groupDayLoad {
		if load > 3 {
			score += (load - 3) * 20
		}
	}
	for _, load := range teacherDayLoad {
		if load > 3 {
			score += (load - 3) * 15
		}
	}
	return score
}

func unscheduledItems(dataset Dataset, entries []ScheduleEntry) []UnscheduledLesson {
	placed := map[string]bool{}
	for _, entry := range entries {
		placed[entry.ID] = true
	}
	items := []UnscheduledLesson{}
	for _, t := range buildTasks(dataset) {
		if !placed[t.ID] {
			items = append(items, UnscheduledLesson{ID: t.ID, LessonID: t.Lesson.ID, Subject: t.Lesson.Subject, SessionIndex: t.SessionIndex, Reason: "не найдено допустимое место без превышения штрафов"})
		}
	}
	return items
}

func utilization(dataset Dataset, entries []ScheduleEntry) float64 {
	capacity := len(dataset.TimeSlots) * len(dataset.Rooms)
	if capacity == 0 {
		return 0
	}
	return math.Round(float64(len(entries))/float64(capacity)*1000) / 10
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
