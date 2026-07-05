package scheduler

func SampleDataset() Dataset {
	days := []string{"Понедельник", "Вторник", "Среда", "Четверг", "Пятница"}
	pairs := [][2]string{{"09:00", "10:30"}, {"10:45", "12:15"}, {"13:00", "14:30"}, {"14:45", "16:15"}}
	slots := make([]TimeSlot, 0, len(days)*len(pairs))
	order := 1
	for dayIndex, day := range days {
		for pairIndex, pair := range pairs {
			slots = append(slots, TimeSlot{
				ID:    "d" + itoa(dayIndex+1) + "-p" + itoa(pairIndex+1),
				Day:   day,
				Start: pair[0],
				End:   pair[1],
				Order: order,
			})
			order++
		}
	}

	return Dataset{
		TimeSlots: slots,
		Rooms: []Room{
			{ID: "aud-101", Name: "Ауд. 101", Capacity: 120, RoomType: RoomLecture},
			{ID: "aud-205", Name: "Ауд. 205", Capacity: 40, RoomType: RoomPractice},
			{ID: "lab-310", Name: "Лаб. 310", Capacity: 28, RoomType: RoomLab},
			{ID: "aud-420", Name: "Ауд. 420", Capacity: 60, RoomType: RoomAny},
		},
		Teachers: []Teacher{
			{ID: "teacher-ivanov", Name: "Иванов И.И.", Unavailable: []string{"d1-p1", "d5-p4"}},
			{ID: "teacher-petrova", Name: "Петрова А.С.", Unavailable: []string{"d3-p3"}},
			{ID: "teacher-sidorov", Name: "Сидоров П.П.", Unavailable: []string{"d2-p4"}},
			{ID: "teacher-kuznetsova", Name: "Кузнецова Е.В.", Unavailable: []string{"d4-p1"}},
			{ID: "teacher-smirnov", Name: "Смирнов Д.А.", Unavailable: []string{}},
		},
		Groups: []Group{
			{ID: "pi-101", Name: "ПИ-101", Size: 28, Unavailable: []string{"d5-p4"}},
			{ID: "pi-102", Name: "ПИ-102", Size: 26, Unavailable: []string{"d1-p1"}},
			{ID: "pi-201", Name: "ПИ-201", Size: 22, Unavailable: []string{"d3-p4"}},
		},
		Lessons: []Lesson{
			{ID: "algorithms", Subject: "Алгоритмы и структуры данных", TeacherID: "teacher-ivanov", GroupIDs: []string{"pi-101", "pi-102"}, Sessions: 2, RoomType: RoomLecture, Priority: 5},
			{ID: "databases", Subject: "Базы данных", TeacherID: "teacher-petrova", GroupIDs: []string{"pi-101"}, Sessions: 2, RoomType: RoomLab, Priority: 4},
			{ID: "web", Subject: "Веб-программирование", TeacherID: "teacher-sidorov", GroupIDs: []string{"pi-102"}, Sessions: 2, RoomType: RoomLab, Priority: 4},
			{ID: "math", Subject: "Дискретная математика", TeacherID: "teacher-smirnov", GroupIDs: []string{"pi-101", "pi-102"}, Sessions: 2, RoomType: RoomPractice, Priority: 4},
			{ID: "networks", Subject: "Компьютерные сети", TeacherID: "teacher-kuznetsova", GroupIDs: []string{"pi-201"}, Sessions: 2, RoomType: RoomLab, Priority: 3},
			{ID: "economics", Subject: "Экономика ИТ-проектов", TeacherID: "teacher-smirnov", GroupIDs: []string{"pi-101", "pi-102", "pi-201"}, Sessions: 1, RoomType: RoomLecture, Priority: 3},
			{ID: "project", Subject: "Проектный практикум", TeacherID: "teacher-sidorov", GroupIDs: []string{"pi-102", "pi-201"}, Sessions: 1, RoomType: RoomPractice, Priority: 5},
			{ID: "ai", Subject: "Основы искусственного интеллекта", TeacherID: "teacher-ivanov", GroupIDs: []string{"pi-201"}, Sessions: 2, RoomType: RoomLab, Priority: 3},
		},
	}
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for value > 0 {
		i--
		buf[i] = byte('0' + value%10)
		value /= 10
	}
	return string(buf[i:])
}
