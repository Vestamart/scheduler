const roomTypeOptions = [
  ["any", "Любая"],
  ["lecture", "Лекция"],
  ["practice", "Практика"],
  ["lab", "Лаборатория"],
];

const schemas = {
  timeslots: {
    title: "Слоты",
    fields: [
      { key: "id", label: "ID", type: "text" },
      { key: "day", label: "День", type: "text" },
      { key: "start", label: "Начало", type: "text" },
      { key: "end", label: "Конец", type: "text" },
      { key: "order", label: "Порядок", type: "number" },
    ],
  },
  rooms: {
    title: "Аудитории",
    fields: [
      { key: "id", label: "ID", type: "text" },
      { key: "name", label: "Название", type: "text" },
      { key: "capacity", label: "Мест", type: "number" },
      { key: "room_type", label: "Тип", type: "select", options: roomTypeOptions },
    ],
  },
  teachers: {
    title: "Преподаватели",
    fields: [
      { key: "id", label: "ID", type: "text" },
      { key: "name", label: "ФИО", type: "text" },
      { key: "unavailable", label: "Недоступные слоты", type: "list" },
    ],
  },
  groups: {
    title: "Группы",
    fields: [
      { key: "id", label: "ID", type: "text" },
      { key: "name", label: "Название", type: "text" },
      { key: "size", label: "Студентов", type: "number" },
      { key: "unavailable", label: "Недоступные слоты", type: "list" },
    ],
  },
  lessons: {
    title: "Занятия",
    fields: [
      { key: "id", label: "ID", type: "text" },
      { key: "subject", label: "Предмет", type: "text" },
      { key: "teacher_id", label: "Преподаватель", type: "text" },
      { key: "group_ids", label: "Группы", type: "list" },
      { key: "sessions", label: "Пар", type: "number" },
      { key: "room_type", label: "Тип ауд.", type: "select", options: roomTypeOptions },
      { key: "priority", label: "Приоритет", type: "number" },
    ],
  },
};

let dataset = null;
let result = null;
let activeSection = "lessons";
let datasetName = "sample";
let datasetPresets = [];
const elements = {};

const fallbackPresets = [
  { id: "sample", title: "Демонстрационный пример" },
  { id: "constrained_rooms", title: "Дефицит аудиторий" },
  { id: "lab_heavy", title: "Много лабораторных" },
  { id: "medium_balanced", title: "Средний сбалансированный" },
  { id: "overloaded_impossible", title: "Перегруженный" },
  { id: "conflict_pressure", title: "Конфликтная нагрузка" },
  { id: "small_basic", title: "Малый базовый" },
];

const presetDescriptions = {
  sample: "Базовый пример для первичной проверки интерфейса и расписания.",
  small_basic: "Небольшой сценарий для быстрой отладки алгоритма.",
  medium_balanced: "Средняя нагрузка без жесткого дефицита ресурсов.",
  lab_heavy: "Много лабораторных занятий и специализированных аудиторий.",
  constrained_rooms: "Ограниченный фонд аудиторий для стресс-теста.",
  overloaded_impossible: "Перегруженный набор, где часть занятий может не разместиться.",
  conflict_pressure: "Много занятий пересекаются по одной группе и одному преподавателю.",
};

const metricHelp = {
  scheduled_count: "Сколько занятий удалось поставить в расписание. Для сравнения алгоритмов большее значение лучше.",
  unscheduled_count: "Сколько занятий осталось без допустимого места. Рост обычно показывает дефицит слотов, аудиторий или слишком жесткие ограничения.",
  conflict_count: "Количество жестких нарушений: один преподаватель, группа или аудитория заняты одновременно. Хороший алгоритм обычно держит это значение равным 0, даже если часть занятий не размещена.",
  score: "Итоговый штраф качества. Он растет из-за неразмещенных занятий, конфликтов и мягких ограничений. Для сравнения алгоритмов меньше лучше.",
  utilization_percent: "Доля занятых аудиторно-временных слотов. Слишком низкая загрузка может означать избыточные ресурсы, слишком высокая - дефицит и риск неразмещенных занятий.",
  elapsed_ms: "Время расчета расписания в миллисекундах. Для одинакового качества меньшее время лучше.",
};

document.addEventListener("DOMContentLoaded", async () => {
  bindElements();
  bindEvents();
  renderTabs();
  await loadDatasetList();
  await loadSelectedDataset();
});

function bindElements() {
  for (const id of [
    "status", "tabs", "editor", "add-row", "load-dataset", "generate", "export-json",
    "import-json", "json-file", "metrics", "schedule",
    "schedule-count", "conflicts", "conflict-count", "unscheduled", "unscheduled-count",
    "iterations", "temperature", "cooling", "dataset-select", "dataset-summary",
    "preset-list",
  ]) {
    elements[toCamel(id)] = document.getElementById(id);
  }
}

function bindEvents() {
  elements.loadDataset.addEventListener("click", loadSelectedDataset);
  elements.datasetSelect.addEventListener("change", loadSelectedDataset);
  elements.presetList.addEventListener("click", (event) => {
    const button = event.target.closest("[data-preset]");
    if (!button) return;
    elements.datasetSelect.value = button.dataset.preset;
    loadSelectedDataset();
  });
  elements.generate.addEventListener("click", generateSchedule);
  elements.addRow.addEventListener("click", addRow);
  elements.exportJson.addEventListener("click", exportJson);
  elements.importJson.addEventListener("click", () => elements.jsonFile.click());
  elements.jsonFile.addEventListener("change", importJson);

  elements.tabs.addEventListener("click", (event) => {
    const button = event.target.closest("[data-section]");
    if (!button) return;
    activeSection = button.dataset.section;
    renderTabs();
    renderEditor();
  });

  elements.editor.addEventListener("input", updateCell);
  elements.editor.addEventListener("change", updateCell);
  elements.editor.addEventListener("click", (event) => {
    const button = event.target.closest("[data-delete-index]");
    if (!button || !dataset) return;
    dataset[activeSection].splice(Number(button.dataset.deleteIndex), 1);
    result = null;
    renderDatasetSummary();
    renderEditor();
    renderResults();
  });
}

async function loadDatasetList() {
  try {
    const response = await fetch("/api/datasets");
    await assertResponse(response);
    const datasets = await response.json();
    datasetPresets = mergePresets(fallbackPresets, datasets);
    renderPresetOptions();
    renderPresetButtons();
  } catch (error) {
    datasetPresets = fallbackPresets;
    renderPresetOptions();
    renderPresetButtons();
    setStatus(`Не удалось получить список датасетов: ${error.message}`, true);
  }
}

async function loadSelectedDataset() {
  datasetName = elements.datasetSelect.value || "sample";
  renderPresetButtons();
  const label = elements.datasetSelect.options[elements.datasetSelect.selectedIndex]?.textContent || "датасет";
  setStatus(`Загрузка: ${label}...`);
  try {
    const url = datasetName === "sample" ? "/api/sample" : `/api/datasets/${encodeURIComponent(datasetName)}`;
    const response = await fetch(url);
    await assertResponse(response);
    dataset = await response.json();
    result = null;
    renderAll();
    renderPresetButtons();
    await generateSchedule();
  } catch (error) {
    setStatus(error.message, true);
  }
}

function renderPresetOptions() {
  elements.datasetSelect.innerHTML = datasetPresets
    .map((item) => `<option value="${escapeHtml(item.id)}">${escapeHtml(item.title)}</option>`)
    .join("");
}

function mergePresets(primary, fromApi) {
  const byId = new Map(primary.map((item) => [item.id, item]));
  for (const item of fromApi || []) {
    if (item?.id) byId.set(item.id, { id: item.id, title: item.title || item.id });
  }
  return Array.from(byId.values());
}

function renderPresetButtons() {
  if (!elements.presetList) return;
  elements.presetList.innerHTML = datasetPresets.map((item) => `
    <button class="preset-card${item.id === datasetName ? " active" : ""}" type="button" data-preset="${escapeHtml(item.id)}">
      <span>${escapeHtml(item.title)}</span>
      <small>${escapeHtml(presetDescriptions[item.id] || "Пользовательский сценарий для проверки расписания.")}</small>
    </button>
  `).join("");
}

async function generateSchedule() {
  if (!dataset) return;
  elements.generate.disabled = true;
  setStatus("Выполняется имитация отжига...");
  try {
    const response = await fetch("/api/schedule", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        dataset,
        options: {
          include_comparison: false,
          seed: 42,
          iterations: Number(elements.iterations.value) || 8000,
          initial_temp: Number(elements.temperature.value) || 150,
          cooling_rate: Number(elements.cooling.value) || 0.997,
        },
      }),
    });
    await assertResponse(response);
    result = await response.json();
    renderResults();
    setStatus("Расписание сформировано.");
  } catch (error) {
    result = null;
    renderResults();
    setStatus(error.message, true);
  } finally {
    elements.generate.disabled = false;
  }
}

function renderAll() {
  renderTabs();
  renderDatasetSummary();
  renderEditor();
  renderResults();
  renderIcons();
}

function renderTabs() {
  elements.tabs.innerHTML = Object.entries(schemas)
    .map(([key, schema]) => {
      const count = dataset?.[key]?.length ?? 0;
      return `<button class="tab-button${key === activeSection ? " active" : ""}" type="button" data-section="${key}">
        <span>${escapeHtml(schema.title)}</span><strong>${count}</strong>
      </button>`;
    })
    .join("");
}

function renderDatasetSummary() {
  if (!dataset) {
    elements.datasetSummary.textContent = "";
    return;
  }
  const taskCount = (dataset.lessons || []).reduce((sum, lesson) => sum + (Number(lesson.sessions) || 1), 0);
  elements.datasetSummary.textContent = `${taskCount} занятий к размещению · ${dataset.timeslots?.length || 0} слотов · ${dataset.rooms?.length || 0} ауд.`;
}

function renderEditor() {
  if (!dataset) {
    elements.editor.innerHTML = "";
    return;
  }
  const schema = schemas[activeSection];
  const rows = dataset[activeSection] || [];
  const header = schema.fields.map((field) => `<th>${escapeHtml(field.label)}</th>`).join("");
  const body = rows.map((row, index) => {
    const cells = schema.fields.map((field) => `<td>${fieldControl(field, row[field.key], index)}</td>`).join("");
    return `<tr>${cells}<td><button class="row-action" type="button" title="Удалить строку" data-delete-index="${index}"><i data-lucide="trash-2"></i></button></td></tr>`;
  }).join("");
  elements.editor.innerHTML = `<div class="table-shell"><table class="editor-table"><thead><tr>${header}<th></th></tr></thead><tbody>${body}</tbody></table></div>`;
  renderIcons();
}

function fieldControl(field, rawValue, index) {
  const value = field.type === "list" ? formatList(rawValue) : rawValue ?? "";
  const common = `data-index="${index}" data-field="${field.key}" data-type="${field.type}"`;
  if (field.type === "select") {
    const options = field.options.map(([optionValue, label]) => `<option value="${escapeHtml(optionValue)}"${optionValue === value ? " selected" : ""}>${escapeHtml(label)}</option>`).join("");
    return `<select class="field-select" ${common}>${options}</select>`;
  }
  return `<input class="field-input" type="${field.type === "number" ? "number" : "text"}" value="${escapeHtml(value)}" ${common} />`;
}

function updateCell(event) {
  const input = event.target.closest("[data-field]");
  if (!input || !dataset) return;
  dataset[activeSection][Number(input.dataset.index)][input.dataset.field] = parseFieldValue(input.dataset.type, input.value);
  result = null;
  renderDatasetSummary();
  renderResults();
}

function addRow() {
  if (!dataset) return;
  dataset[activeSection].push(defaultRow(activeSection));
  result = null;
  renderTabs();
  renderDatasetSummary();
  renderEditor();
  renderResults();
}

function defaultRow(section) {
  const suffix = Math.random().toString(36).slice(2, 7);
  const nextOrder = (dataset?.timeslots?.length || 0) + 1;
  return {
    timeslots: { id: `slot-${suffix}`, day: "Понедельник", start: "09:00", end: "10:30", order: nextOrder },
    rooms: { id: `room-${suffix}`, name: "Новая аудитория", capacity: 30, room_type: "practice" },
    teachers: { id: `teacher-${suffix}`, name: "Новый преподаватель", unavailable: [] },
    groups: { id: `group-${suffix}`, name: "Новая группа", size: 25, unavailable: [] },
    lessons: { id: `lesson-${suffix}`, subject: "Новое занятие", teacher_id: dataset?.teachers?.[0]?.id || "", group_ids: dataset?.groups?.[0]?.id ? [dataset.groups[0].id] : [], sessions: 1, room_type: "practice", priority: 3 },
  }[section];
}

function renderResults() {
  if (!result) {
    elements.metrics.innerHTML = "";
    elements.schedule.innerHTML = `<div class="empty-state">Нет рассчитанного расписания.</div>`;
    elements.scheduleCount.textContent = "";
    elements.conflicts.innerHTML = `<div class="empty-state">Нет данных.</div>`;
    elements.conflictCount.textContent = "";
    elements.unscheduled.innerHTML = `<div class="empty-state">Нет данных.</div>`;
    elements.unscheduledCount.textContent = "";
    return;
  }
  const main = result.annealing;
  renderMetrics(main.stats);
  renderSchedule(main.entries || []);
  renderConflicts(main.conflicts || []);
  renderUnscheduled(main.unscheduled || []);
}

function renderMetrics(stats) {
  elements.metrics.innerHTML = [
    metricCard("Размещено", stats.scheduled_count, "calendar-check", "green", metricHelp.scheduled_count),
    metricCard("Не размещено", stats.unscheduled_count, "circle-alert", stats.unscheduled_count ? "amber" : "green", metricHelp.unscheduled_count),
    metricCard("Конфликты", stats.conflict_count, "shield-alert", stats.conflict_count ? "red" : "green", metricHelp.conflict_count),
    metricCard("Штраф", stats.score, "gauge", "", metricHelp.score),
    metricCard("Загрузка", `${formatMetricNumber(stats.utilization_percent)}%`, "activity", "", metricHelp.utilization_percent),
    metricCard("Время", `${formatMetricNumber(stats.elapsed_ms)} мс`, "timer", "", metricHelp.elapsed_ms),
  ].join("");
  renderIcons();
}

function metricCard(label, value, icon, tone, help) {
  return `<div class="metric ${tone}" aria-label="${escapeHtml(`${label}: ${value}. ${help}`)}">
    <i data-lucide="${icon}"></i>
    <span>${escapeHtml(label)}</span>
    <strong tabindex="0" data-tooltip="${escapeHtml(help)}">${escapeHtml(value)}</strong>
  </div>`;
}

function formatMetricNumber(value) {
  const number = Number(value);
  if (!Number.isFinite(number)) return "0";
  return number.toFixed(number >= 100 ? 0 : 1);
}

function renderSchedule(entries) {
  elements.scheduleCount.textContent = `${entries.length} занятий`;
  if (!dataset || !entries.length) {
    elements.schedule.innerHTML = `<div class="empty-state">Расписание пустое.</div>`;
    return;
  }
  const rooms = dataset.rooms || [];
  const slotsByDay = groupSlotsByDay(dataset.timeslots || []);
  const entryMap = new Map();
  for (const entry of entries) {
    const key = `${entry.timeslot_id}|${entry.room_id}`;
    const list = entryMap.get(key) || [];
    list.push(entry);
    entryMap.set(key, list);
  }
  elements.schedule.innerHTML = Object.entries(slotsByDay).map(([day, slots]) => dayBlock(day, slots, rooms, entryMap)).join("");
}

function dayBlock(day, slots, rooms, entryMap) {
  const roomHeaders = rooms.map((room) => `<th>${escapeHtml(room.name)}</th>`).join("");
  const rows = slots.map((slot) => {
    const cells = rooms.map((room) => {
      const entries = entryMap.get(`${slot.id}|${room.id}`) || [];
      return `<td>${entries.length ? entries.map(lessonPill).join("") : `<span class="empty-cell">Свободно</span>`}</td>`;
    }).join("");
    return `<tr><td><span class="slot-time">${escapeHtml(slot.start)}-${escapeHtml(slot.end)}</span></td>${cells}</tr>`;
  }).join("");
  const dayCount = slots.reduce((count, slot) => count + rooms.reduce((sum, room) => sum + (entryMap.get(`${slot.id}|${room.id}`)?.length || 0), 0), 0);
  return `<div class="day-block"><div class="day-title"><h3>${escapeHtml(day)}</h3><span>${dayCount} занятий</span></div><div class="table-shell"><table class="day-table"><thead><tr><th>Время</th>${roomHeaders}</tr></thead><tbody>${rows}</tbody></table></div></div>`;
}

function lessonPill(entry) {
  return `<div class="lesson-pill"><strong>${escapeHtml(entry.subject)}</strong><span>${escapeHtml(entry.groups.join(", "))}</span><span>${escapeHtml(entry.teacher)} · ${escapeHtml(entry.room)}</span></div>`;
}

function renderConflicts(conflicts) {
  elements.conflictCount.textContent = String(conflicts.length);
  elements.conflicts.innerHTML = conflicts.length
    ? conflicts.map((conflict) => `<div class="event-item conflict"><strong>${escapeHtml(conflict.type)}</strong><span>${escapeHtml(conflict.message)}</span></div>`).join("")
    : `<div class="empty-state good">Конфликтов нет.</div>`;
}

function renderUnscheduled(items) {
  elements.unscheduledCount.textContent = String(items.length);
  elements.unscheduled.innerHTML = items.length
    ? items.map((item) => `<div class="event-item pending"><strong>${escapeHtml(item.subject)} · пара ${escapeHtml(item.session_index)}</strong><span>${escapeHtml(item.reason)}</span></div>`).join("")
    : `<div class="empty-state good">Все занятия размещены.</div>`;
}

function groupSlotsByDay(slots) {
  return slots.slice().sort((a, b) => (a.order || 0) - (b.order || 0)).reduce((acc, slot) => {
    if (!acc[slot.day]) acc[slot.day] = [];
    acc[slot.day].push(slot);
    return acc;
  }, {});
}

function exportJson() {
  if (!dataset) return;
  const blob = new Blob([JSON.stringify(dataset, null, 2)], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = `${datasetName || "schedule"}-data.json`;
  link.click();
  URL.revokeObjectURL(url);
}

async function importJson(event) {
  const file = event.target.files?.[0];
  if (!file) return;
  try {
    dataset = JSON.parse(await file.text());
    datasetName = file.name.replace(/\.json$/i, "");
    elements.datasetSelect.value = "sample";
    result = null;
    renderAll();
    renderPresetButtons();
    setStatus("Данные импортированы.");
  } catch (error) {
    setStatus(`Не удалось импортировать JSON: ${error.message}`, true);
  } finally {
    event.target.value = "";
  }
}

function parseFieldValue(type, value) {
  if (type === "number") {
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : 0;
  }
  if (type === "list") return String(value || "").split(",").map((item) => item.trim()).filter(Boolean);
  return value;
}

function formatList(value) {
  return Array.isArray(value) ? value.join(", ") : value || "";
}

function setStatus(message, isError = false) {
  elements.status.textContent = message;
  elements.status.classList.toggle("error", isError);
}

async function assertResponse(response) {
  if (response.ok) return;
  const payload = await response.json().catch(() => null);
  throw new Error(payload?.error || `Ошибка API: ${response.status}`);
}

function renderIcons() {
  if (window.lucide?.createIcons) window.lucide.createIcons();
}

function escapeHtml(value) {
  return String(value ?? "").replace(/[&<>"']/g, (char) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#039;" })[char]);
}

function toCamel(value) {
  return value.replace(/-([a-z])/g, (_, char) => char.toUpperCase());
}
