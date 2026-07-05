package scheduler

import (
	"errors"
	"fmt"
	"sort"
)

type context struct {
	slots    map[string]TimeSlot
	rooms    map[string]Room
	teachers map[string]Teacher
	groups   map[string]Group
	lessons  map[string]Lesson
}

func newContext(dataset Dataset) context {
	ctx := context{
		slots:    map[string]TimeSlot{},
		rooms:    map[string]Room{},
		teachers: map[string]Teacher{},
		groups:   map[string]Group{},
		lessons:  map[string]Lesson{},
	}
	for _, item := range dataset.TimeSlots {
		ctx.slots[item.ID] = item
	}
	for _, item := range dataset.Rooms {
		ctx.rooms[item.ID] = item
	}
	for _, item := range dataset.Teachers {
		ctx.teachers[item.ID] = item
	}
	for _, item := range dataset.Groups {
		ctx.groups[item.ID] = item
	}
	for _, item := range dataset.Lessons {
		ctx.lessons[item.ID] = item
	}
	return ctx
}

func ValidateDataset(dataset Dataset) error {
	if len(dataset.TimeSlots) == 0 || len(dataset.Rooms) == 0 || len(dataset.Teachers) == 0 || len(dataset.Groups) == 0 || len(dataset.Lessons) == 0 {
		return errors.New("все разделы данных должны содержать хотя бы одну запись")
	}
	if err := unique("timeslots", idsSlots(dataset.TimeSlots)); err != nil {
		return err
	}
	if err := unique("rooms", idsRooms(dataset.Rooms)); err != nil {
		return err
	}
	if err := unique("teachers", idsTeachers(dataset.Teachers)); err != nil {
		return err
	}
	if err := unique("groups", idsGroups(dataset.Groups)); err != nil {
		return err
	}
	if err := unique("lessons", idsLessons(dataset.Lessons)); err != nil {
		return err
	}
	ctx := newContext(dataset)
	for _, teacher := range dataset.Teachers {
		for _, slotID := range teacher.Unavailable {
			if _, ok := ctx.slots[slotID]; !ok {
				return fmt.Errorf("преподаватель %s ссылается на неизвестный слот %s", teacher.ID, slotID)
			}
		}
	}
	for _, group := range dataset.Groups {
		if group.Size <= 0 {
			return fmt.Errorf("группа %s должна иметь положительный размер", group.ID)
		}
		for _, slotID := range group.Unavailable {
			if _, ok := ctx.slots[slotID]; !ok {
				return fmt.Errorf("группа %s ссылается на неизвестный слот %s", group.ID, slotID)
			}
		}
	}
	for _, room := range dataset.Rooms {
		if room.Capacity <= 0 {
			return fmt.Errorf("аудитория %s должна иметь положительную вместимость", room.ID)
		}
	}
	for _, lesson := range dataset.Lessons {
		if lesson.Sessions <= 0 {
			return fmt.Errorf("занятие %s должно иметь положительное число пар", lesson.ID)
		}
		if _, ok := ctx.teachers[lesson.TeacherID]; !ok {
			return fmt.Errorf("занятие %s ссылается на неизвестного преподавателя %s", lesson.ID, lesson.TeacherID)
		}
		for _, groupID := range lesson.GroupIDs {
			if _, ok := ctx.groups[groupID]; !ok {
				return fmt.Errorf("занятие %s ссылается на неизвестную группу %s", lesson.ID, groupID)
			}
		}
	}
	return nil
}

func ValidateSchedule(dataset Dataset, entries []ScheduleEntry) []ValidationConflict {
	ctx := newContext(dataset)
	var conflicts []ValidationConflict
	roomUsage := map[string]ScheduleEntry{}
	teacherUsage := map[string]ScheduleEntry{}
	groupUsage := map[string]ScheduleEntry{}

	for _, entry := range entries {
		slot, hasSlot := ctx.slots[entry.TimeSlotID]
		room, hasRoom := ctx.rooms[entry.RoomID]
		lesson, hasLesson := ctx.lessons[entry.LessonID]
		teacher, hasTeacher := ctx.teachers[entry.TeacherID]
		if !hasSlot || !hasRoom || !hasLesson || !hasTeacher {
			conflicts = append(conflicts, ValidationConflict{Type: "unknown_reference", Message: "запись содержит неизвестные идентификаторы", EntryIDs: []string{entry.ID}})
			continue
		}
		groupSize := groupSize(ctx, entry.GroupIDs)
		if room.Capacity < groupSize {
			conflicts = append(conflicts, ValidationConflict{Type: "room_capacity", Message: fmt.Sprintf("Аудитория %s вмещает %d, требуется %d мест.", room.Name, room.Capacity, groupSize), EntryIDs: []string{entry.ID}})
		}
		if !roomMatches(lesson.RoomType, room.RoomType) {
			conflicts = append(conflicts, ValidationConflict{Type: "room_type", Message: fmt.Sprintf("Для занятия %s нужен тип %s, выбрана аудитория типа %s.", lesson.Subject, lesson.RoomType, room.RoomType), EntryIDs: []string{entry.ID}})
		}
		if contains(teacher.Unavailable, slot.ID) {
			conflicts = append(conflicts, ValidationConflict{Type: "teacher_unavailable", Message: fmt.Sprintf("Преподаватель %s недоступен в слот %s %s.", teacher.Name, slot.Day, slot.Start), EntryIDs: []string{entry.ID}})
		}
		checkBusy(&conflicts, roomUsage, room.ID+"|"+slot.ID, entry, "room_busy", fmt.Sprintf("Аудитория %s занята в слот %s %s.", room.Name, slot.Day, slot.Start))
		checkBusy(&conflicts, teacherUsage, teacher.ID+"|"+slot.ID, entry, "teacher_busy", fmt.Sprintf("Преподаватель %s занят в слот %s %s.", teacher.Name, slot.Day, slot.Start))
		for _, groupID := range entry.GroupIDs {
			group, ok := ctx.groups[groupID]
			if !ok {
				conflicts = append(conflicts, ValidationConflict{Type: "unknown_group", Message: "запись содержит неизвестную группу", EntryIDs: []string{entry.ID}})
				continue
			}
			if contains(group.Unavailable, slot.ID) {
				conflicts = append(conflicts, ValidationConflict{Type: "group_unavailable", Message: fmt.Sprintf("Группа %s недоступна в слот %s %s.", group.Name, slot.Day, slot.Start), EntryIDs: []string{entry.ID}})
			}
			checkBusy(&conflicts, groupUsage, group.ID+"|"+slot.ID, entry, "group_busy", fmt.Sprintf("Группа %s занята в слот %s %s.", group.Name, slot.Day, slot.Start))
		}
	}
	return conflicts
}

func checkBusy(conflicts *[]ValidationConflict, usage map[string]ScheduleEntry, key string, entry ScheduleEntry, typ string, message string) {
	if previous, ok := usage[key]; ok {
		*conflicts = append(*conflicts, ValidationConflict{Type: typ, Message: message, EntryIDs: []string{previous.ID, entry.ID}})
		return
	}
	usage[key] = entry
}

func unique(name string, ids []string) error {
	seen := map[string]bool{}
	for _, id := range ids {
		if id == "" {
			return fmt.Errorf("%s содержит пустой идентификатор", name)
		}
		if seen[id] {
			return fmt.Errorf("%s содержит повторяющийся идентификатор %s", name, id)
		}
		seen[id] = true
	}
	return nil
}

func idsSlots(items []TimeSlot) []string {
	out := make([]string, len(items))
	for i, item := range items {
		out[i] = item.ID
	}
	return out
}

func idsRooms(items []Room) []string {
	out := make([]string, len(items))
	for i, item := range items {
		out[i] = item.ID
	}
	return out
}

func idsTeachers(items []Teacher) []string {
	out := make([]string, len(items))
	for i, item := range items {
		out[i] = item.ID
	}
	return out
}

func idsGroups(items []Group) []string {
	out := make([]string, len(items))
	for i, item := range items {
		out[i] = item.ID
	}
	return out
}

func idsLessons(items []Lesson) []string {
	out := make([]string, len(items))
	for i, item := range items {
		out[i] = item.ID
	}
	return out
}

func sortedSlots(dataset Dataset) []TimeSlot {
	slots := append([]TimeSlot(nil), dataset.TimeSlots...)
	sort.Slice(slots, func(i, j int) bool {
		if slots[i].Order != slots[j].Order {
			return slots[i].Order < slots[j].Order
		}
		return slots[i].ID < slots[j].ID
	})
	return slots
}

func sortedRooms(dataset Dataset) []Room {
	rooms := append([]Room(nil), dataset.Rooms...)
	sort.Slice(rooms, func(i, j int) bool {
		if rooms[i].Capacity != rooms[j].Capacity {
			return rooms[i].Capacity < rooms[j].Capacity
		}
		return rooms[i].ID < rooms[j].ID
	})
	return rooms
}

func groupSize(ctx context, groupIDs []string) int {
	total := 0
	for _, id := range groupIDs {
		total += ctx.groups[id].Size
	}
	return total
}

func roomMatches(required RoomType, actual RoomType) bool {
	return required == RoomAny || actual == RoomAny || required == actual
}

func contains(items []string, needle string) bool {
	for _, item := range items {
		if item == needle {
			return true
		}
	}
	return false
}
