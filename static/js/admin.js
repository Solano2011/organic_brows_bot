const tg = window.Telegram.WebApp;
tg.expand();

const monthNames = ['Январь', 'Февраль', 'Март', 'Апрель', 'Май', 'Июнь', 'Июль', 'Август', 'Сентябрь', 'Октябрь', 'Ноябрь', 'Декабрь'];
let view = new Date();
view.setDate(1);
let days = {};
let breaksByDate = {};
let selectedDates = [];

const calendar = document.getElementById('calendar');
const form = document.getElementById('dayForm');
const workingInput = document.getElementById('isWorking');
const hoursFields = document.getElementById('hoursFields');

document.getElementById('prevMonth').onclick = () => { view.setMonth(view.getMonth() - 1); loadMonth(); };
document.getElementById('nextMonth').onclick = () => { view.setMonth(view.getMonth() + 1); loadMonth(); };
workingInput.onchange = toggleHours;
form.onsubmit = saveDay;
document.getElementById('addBreak').onclick = () => addBreakRow('13:00', '14:00');

function toggleHours() {
    hoursFields.hidden = !workingInput.checked;
}

function dateKey(year, month, day) {
    return year + '-' + String(month).padStart(2, '0') + '-' + String(day).padStart(2, '0');
}

async function loadMonth() {
    const year = view.getFullYear();
    const month = view.getMonth() + 1;
    document.getElementById('monthTitle').textContent = monthNames[view.getMonth()] + ' ' + year;
    const response = await fetch('/api/admin/schedule?month=' + month + '&year=' + year);
    const data = await response.json();
    const list = Array.isArray(data) ? data : (data.days || []);
    days = {};
    list.forEach(item => { days[item.Date] = item; });
    breaksByDate = data.breaks || {};
    if (data.slotStepMinutes) {
        document.getElementById('slotStep').value = String(data.slotStepMinutes);
    }
    renderCalendar();
    updateSelectionLabel();
}

function renderCalendar() {
    const year = view.getFullYear();
    const month = view.getMonth();
    const firstWeekday = (new Date(year, month, 1).getDay() + 6) % 7;
    const count = new Date(year, month + 1, 0).getDate();
    calendar.innerHTML = '';
    for (let i = 0; i < firstWeekday; i++) {
        const blank = document.createElement('div');
        blank.className = 'day-cell empty';
        calendar.appendChild(blank);
    }
    for (let day = 1; day <= count; day++) {
        const key = dateKey(year, month + 1, day);
        const saved = days[key];
        const button = document.createElement('button');
        button.type = 'button';
        button.className = 'day-cell ' + (saved && saved.IsWorkingDay === false ? 'dayoff' : 'working');
        if (selectedDates.includes(key)) button.classList.add('selected');
        button.textContent = String(day);
        button.onclick = () => toggleDate(key);
        calendar.appendChild(button);
    }
}

function toggleDate(key) {
    const index = selectedDates.indexOf(key);
    if (index >= 0) {
        selectedDates.splice(index, 1);
    } else {
        selectedDates.push(key);
        if (selectedDates.length === 1) {
            const saved = days[key];
            workingInput.checked = !saved || saved.IsWorkingDay !== false;
            document.getElementById('startTime').value = (saved && saved.StartTime) || '10:00';
            document.getElementById('endTime').value = (saved && saved.EndTime) || '20:00';
            setBreaks(breaksByDate[key] || []);
            toggleHours();
        }
    }
    selectedDates.sort();
    if (!selectedDates.length) setBreaks([]);
    form.hidden = selectedDates.length === 0;
    document.getElementById('saveStatus').textContent = '';
    updateSelectionLabel();
    renderCalendar();
}

function updateSelectionLabel() {
    const label = document.getElementById('selectedDateLabel');
    if (!selectedDates.length) {
        label.textContent = 'Выберите дни в календаре';
        return;
    }
    const pretty = selectedDates.map(date => date.split('-').reverse().join('.')).join(', ');
    label.textContent = 'Выбрано: ' + selectedDates.length + ' — ' + pretty;
}

async function saveDay(event) {
    event.preventDefault();
    if (!selectedDates.length) return;
    const payload = {
        Dates: selectedDates.slice(),
        IsWorkingDay: workingInput.checked,
        StartTime: document.getElementById('startTime').value || '10:00',
        EndTime: document.getElementById('endTime').value || '20:00',
        SlotStepMinutes: Number(document.getElementById('slotStep').value),
        breaks: collectBreaks()
    };
    const response = await fetch('/api/admin/schedule', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
    });
    const status = document.getElementById('saveStatus');
    if (!response.ok) {
        status.textContent = 'Не удалось сохранить';
        return;
    }
    payload.Dates.forEach(date => {
        days[date] = {
            Date: date,
            IsWorkingDay: payload.IsWorkingDay,
            StartTime: payload.StartTime,
            EndTime: payload.EndTime
        };
        breaksByDate[date] = payload.breaks.slice();
    });
    status.textContent = 'Сохранено';
    renderCalendar();
}

function addBreakRow(start, end) {
    const row = document.createElement('div');
    row.className = 'break-row';
    const startInput = document.createElement('input');
    startInput.type = 'time';
    startInput.className = 'input-field break-start';
    startInput.value = start || '13:00';
    const endInput = document.createElement('input');
    endInput.type = 'time';
    endInput.className = 'input-field break-end';
    endInput.value = end || '14:00';
    const remove = document.createElement('button');
    remove.type = 'button';
    remove.className = 'break-remove';
    remove.textContent = '✕';
    remove.setAttribute('aria-label', 'Удалить перерыв');
    remove.onclick = () => row.remove();
    row.append(startInput, endInput, remove);
    document.getElementById('breaksList').appendChild(row);
}

function setBreaks(list) {
    const box = document.getElementById('breaksList');
    box.innerHTML = '';
    (list || []).forEach(item => addBreakRow(item.startTime || item.Start, item.endTime || item.End));
}

function collectBreaks() {
    return Array.from(document.querySelectorAll('.break-row')).map(row => ({
        startTime: row.querySelector('.break-start').value,
        endTime: row.querySelector('.break-end').value
    })).filter(item => item.startTime && item.endTime);
}

loadMonth();
