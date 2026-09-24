const tg = window.Telegram.WebApp;
tg.expand();

const monthNames = ['Январь', 'Февраль', 'Март', 'Апрель', 'Май', 'Июнь', 'Июль', 'Август', 'Сентябрь', 'Октябрь', 'Ноябрь', 'Декабрь'];
let view = new Date();
view.setDate(1);
let days = {};
let selectedDate = '';

const calendar = document.getElementById('calendar');
const form = document.getElementById('dayForm');
const workingInput = document.getElementById('isWorking');
const hoursFields = document.getElementById('hoursFields');

document.getElementById('prevMonth').onclick = () => { view.setMonth(view.getMonth() - 1); loadMonth(); };
document.getElementById('nextMonth').onclick = () => { view.setMonth(view.getMonth() + 1); loadMonth(); };
workingInput.onchange = toggleHours;
form.onsubmit = saveDay;

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
    const list = await response.json();
    days = {};
    (list || []).forEach(item => { days[item.Date] = item; });
    renderCalendar();
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
        if (key === selectedDate) button.classList.add('selected');
        button.textContent = String(day);
        button.onclick = () => openDay(key);
        calendar.appendChild(button);
    }
}

function openDay(key) {
    selectedDate = key;
    const saved = days[key];
    document.getElementById('selectedDateLabel').textContent = key.split('-').reverse().join('.');
    workingInput.checked = !saved || saved.IsWorkingDay !== false;
    document.getElementById('startTime').value = (saved && saved.StartTime) || '10:00';
    document.getElementById('endTime').value = (saved && saved.EndTime) || '20:00';
    form.hidden = false;
    document.getElementById('saveStatus').textContent = '';
    toggleHours();
    renderCalendar();
}

async function saveDay(event) {
    event.preventDefault();
    const payload = {
        Date: selectedDate,
        IsWorkingDay: workingInput.checked,
        StartTime: document.getElementById('startTime').value || '10:00',
        EndTime: document.getElementById('endTime').value || '20:00'
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
    days[selectedDate] = payload;
    status.textContent = 'Сохранено';
    renderCalendar();
}

loadMonth();
