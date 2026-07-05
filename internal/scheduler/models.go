package scheduler

type RoomType string

const (
	RoomAny      RoomType = "any"
	RoomLecture  RoomType = "lecture"
	RoomPractice RoomType = "practice"
	RoomLab      RoomType = "lab"
)

type AlgorithmName string

const (
	AlgorithmAnnealing  AlgorithmName = "annealing"
	AlgorithmGreedy     AlgorithmName = "greedy"
	AlgorithmSequential AlgorithmName = "sequential"
	AlgorithmRandom     AlgorithmName = "random"
)

type TimeSlot struct {
	ID    string `json:"id"`
	Day   string `json:"day"`
	Start string `json:"start"`
	End   string `json:"end"`
	Order int    `json:"order"`
}

type Room struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Capacity int      `json:"capacity"`
	RoomType RoomType `json:"room_type"`
}

type Teacher struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Unavailable []string `json:"unavailable"`
}

type Group struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Size        int      `json:"size"`
	Unavailable []string `json:"unavailable"`
}

type Lesson struct {
	ID        string   `json:"id"`
	Subject   string   `json:"subject"`
	TeacherID string   `json:"teacher_id"`
	GroupIDs  []string `json:"group_ids"`
	Sessions  int      `json:"sessions"`
	RoomType  RoomType `json:"room_type"`
	Priority  int      `json:"priority"`
}

type Dataset struct {
	TimeSlots []TimeSlot `json:"timeslots"`
	Rooms     []Room     `json:"rooms"`
	Teachers  []Teacher  `json:"teachers"`
	Groups    []Group    `json:"groups"`
	Lessons   []Lesson   `json:"lessons"`
}

type ScheduleEntry struct {
	ID           string   `json:"id"`
	LessonID     string   `json:"lesson_id"`
	Subject      string   `json:"subject"`
	SessionIndex int      `json:"session_index"`
	TeacherID    string   `json:"teacher_id"`
	Teacher      string   `json:"teacher"`
	GroupIDs     []string `json:"group_ids"`
	Groups       []string `json:"groups"`
	RoomID       string   `json:"room_id"`
	Room         string   `json:"room"`
	TimeSlotID   string   `json:"timeslot_id"`
	Day          string   `json:"day"`
	Start        string   `json:"start"`
	End          string   `json:"end"`
}

type UnscheduledLesson struct {
	ID           string `json:"id"`
	LessonID     string `json:"lesson_id"`
	Subject      string `json:"subject"`
	SessionIndex int    `json:"session_index"`
	Reason       string `json:"reason"`
}

type ValidationConflict struct {
	Type     string   `json:"type"`
	Message  string   `json:"message"`
	EntryIDs []string `json:"entry_ids"`
}

type AlgorithmStats struct {
	Algorithm          AlgorithmName `json:"algorithm"`
	Title              string        `json:"title"`
	ScheduledCount     int           `json:"scheduled_count"`
	UnscheduledCount   int           `json:"unscheduled_count"`
	ConflictCount      int           `json:"conflict_count"`
	UtilizationPercent float64       `json:"utilization_percent"`
	Score              int           `json:"score"`
	ElapsedMS          float64       `json:"elapsed_ms"`
}

type ScheduleResult struct {
	Algorithm   AlgorithmName        `json:"algorithm"`
	Title       string               `json:"title"`
	Entries     []ScheduleEntry      `json:"entries"`
	Unscheduled []UnscheduledLesson  `json:"unscheduled"`
	Conflicts   []ValidationConflict `json:"conflicts"`
	Stats       AlgorithmStats       `json:"stats"`
}

type ScheduleOptions struct {
	Seed              int64   `json:"seed"`
	IncludeComparison bool    `json:"include_comparison"`
	Iterations        int     `json:"iterations"`
	InitialTemp       float64 `json:"initial_temp"`
	CoolingRate       float64 `json:"cooling_rate"`
}

type ScheduleRequest struct {
	Dataset Dataset         `json:"dataset"`
	Options ScheduleOptions `json:"options"`
}

type ScheduleResponse struct {
	Annealing   ScheduleResult   `json:"annealing"`
	Greedy      ScheduleResult   `json:"greedy"`
	Comparisons []AlgorithmStats `json:"comparisons"`
}
