let tg = window.Telegram.WebApp;
tg.expand();

let selectedServiceName = "";
let selectedTime = "";
let selectedDate = "";

tg.MainButton.setText("ЗАПИСАТЬСЯ");
tg.MainButton.color = "#d4a574";
tg.MainButton.textColor = "#FFFFFF";
tg.MainButton.hide();

tg.MainButton.onClick(submitBooking);

const phoneInput = document.getElementById('userPhone');
if (phoneInput) {
    phoneInput.addEventListener('input', () => {
        let digits = phoneInput.value.replace(/\D/g, '');
        if (digits.startsWith('7')) {
            digits = digits.slice(1);
        }
        phoneInput.value = '+7' + digits.slice(0, 10);
        updateBookingButton();
    });
}

function isTodaySelected() {
    const now = new Date();
    const todayStr = now.getFullYear() + '-' + String(now.getMonth() + 1).padStart(2, '0') + '-' + String(now.getDate()).padStart(2, '0');
    return selectedDate === todayStr;
}

function isSlotUnavailable(timeStr) {
    if (!timeStr || !isTodaySelected()) return false;
    const parts = timeStr.split(':');
    const slot = new Date();
    slot.setHours(Number(parts[0]), Number(parts[1]), 0, 0);
    return slot.getTime() - Date.now() < 15 * 60 * 1000;
}

function markPastTimeSlots() {
    document.querySelectorAll('.time-chip').forEach(chip => {
        const time = chip.innerText.trim();
        if (isSlotUnavailable(time)) {
            chip.classList.add('disabled');
            chip.classList.remove('selected');
            if (selectedTime === time) selectedTime = '';
        } else {
            chip.classList.remove('disabled');
        }
    });
    updateBookingButton();
}

function updateBookingButton() {
    const phone = phoneInput ? phoneInput.value.trim() : '';
    if (selectedTime && !isSlotUnavailable(selectedTime) && /^\+7\d{10}$/.test(phone)) {
        tg.MainButton.show();
    } else {
        tg.MainButton.hide();
    }
}

window.onload = function() {
    const urlParams = new URLSearchParams(window.location.search);
    selectedServiceName = urlParams.get('service');

    if (!selectedServiceName) {
        document.getElementById('serviceSelector').classList.add('active');
        document.getElementById('mainServiceHeader').style.display = 'none';
        document.getElementById('dateSection').style.display = 'none';
        document.getElementById('timePicker').style.display = 'none';
        document.getElementById('contactForm').style.display = 'none';
    } else {
        document.getElementById('serviceTitle').innerText = selectedServiceName;
        generateDateButtons();
    }
};

function selectServiceFromMenu(serviceName) {
    selectedServiceName = serviceName;
    document.getElementById('serviceSelector').classList.remove('active');
    document.getElementById('backButtonContainer').style.display = 'block';
    document.getElementById('mainServiceHeader').style.display = 'block';
    document.getElementById('dateSection').style.display = 'block';
    document.getElementById('serviceTitle').innerText = serviceName;
    generateDateButtons();
}

function backToMenu() {
    document.getElementById('backButtonContainer').style.display = 'none';
    document.getElementById('mainServiceHeader').style.display = 'none';
    document.getElementById('dateSection').style.display = 'none';
    document.getElementById('timePicker').style.display = 'none';
    document.getElementById('timePicker').classList.remove('active');
    const contactForm = document.getElementById('contactForm');
    contactForm.style.display = 'none';
    contactForm.classList.remove('active');
    selectedServiceName = "";
    selectedDate = "";
    selectedTime = "";
    document.querySelectorAll('.date-chip').forEach(c => c.classList.remove('selected'));
    document.querySelectorAll('.time-chip').forEach(c => c.classList.remove('selected'));
    document.getElementById('dateScroll').innerHTML = '';
    document.getElementById('userName').value = '';
    document.getElementById('userPhone').value = '+7';
    document.getElementById('userComment').value = '';
    tg.MainButton.hide();
    document.getElementById('serviceSelector').classList.add('active');
}

function generateDateButtons() {
    const dateScroll = document.getElementById('dateScroll');
    if (!dateScroll) return;
    const today = new Date();
    const days = ['Вс', 'Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб'];
    const months = ['янв', 'фев', 'мар', 'апр', 'май', 'июн', 'июл', 'авг', 'сен', 'окт', 'ноя', 'дек'];
    for (let i = 0; i < 7; i++) {
        const date = new Date(today);
        date.setDate(today.getDate() + i);
        const dayName = i === 0 ? 'Сегодня' : days[date.getDay()];
        const dayNum = date.getDate() + ' ' + months[date.getMonth()];
        const dateStr = date.getFullYear() + '-' + String(date.getMonth() + 1).padStart(2, '0') + '-' + String(date.getDate()).padStart(2, '0');
        const chip = document.createElement('div');
        chip.className = 'date-chip' + (i === 0 ? ' selected' : '');
        chip.innerHTML = '<div class="date-day">' + dayName + '</div><div class="date-num">' + dayNum + '</div>';
        chip.onclick = () => selectDate(dateStr, chip);
        dateScroll.appendChild(chip);
    }
    const todayStr = today.getFullYear() + '-' + String(today.getMonth() + 1).padStart(2, '0') + '-' + String(today.getDate()).padStart(2, '0');
    selectedDate = todayStr;
    updateTimeAvailability();
}

function selectDate(dateStr, element) {
    selectedDate = dateStr;
    document.querySelectorAll('.date-chip').forEach(c => c.classList.remove('selected'));
    element.classList.add('selected');
    selectedTime = "";
    document.querySelectorAll('.time-chip').forEach(c => c.classList.remove('selected'));
    document.getElementById('contactForm').classList.remove('active');
    tg.MainButton.hide();
    const timePicker = document.getElementById('timePicker');
    timePicker.style.display = 'block';
    timePicker.classList.add('active');
    updateTimeAvailability();
    setTimeout(() => { timePicker.scrollIntoView({ behavior: 'smooth', block: 'nearest' }); }, 150);
}

function updateTimeAvailability() {
    if (!selectedDate) return;
    const timePicker = document.getElementById('timePicker');
    timePicker.classList.add('active');
    fetch('/api/availability?date=' + selectedDate)
        .then(response => response.json())
        .then(data => {
            const takenTimes = data.takenSlots || [];
            document.querySelectorAll('.time-chip').forEach(chip => {
                const time = chip.innerText.trim();
                chip.classList.remove('taken', 'selected');
                if (takenTimes.includes(time)) { chip.classList.add('taken'); }
            });
            markPastTimeSlots();
        })
        .catch(err => {
            console.error("Ошибка получения доступности:", err);
            markPastTimeSlots();
        });
    markPastTimeSlots();
}

function selectTime(time, element) {
    if (element.classList.contains('taken') || element.classList.contains('disabled')) return;
    if (isSlotUnavailable(time)) {
        tg.showAlert("Это время уже прошло, выберите другое");
        return;
    }
    selectedTime = time.trim();
    document.querySelectorAll('.time-chip').forEach(c => c.classList.remove('selected'));
    element.classList.add('selected');
    const contactForm = document.getElementById('contactForm');
    contactForm.style.display = 'block';
    contactForm.classList.add('active');
    updateBookingButton();
    setTimeout(() => { contactForm.scrollIntoView({ behavior: 'smooth', block: 'nearest' }); }, 150);
}

function submitBooking() {
    const nameEl = document.getElementById('userName');
    const phoneEl = document.getElementById('userPhone');
    const commentEl = document.getElementById('userComment');
    const name = nameEl ? nameEl.value.trim() : "";
    const phone = phoneEl ? phoneEl.value.trim() : "";
    const comment = commentEl ? commentEl.value.trim() : "";
    if (!name || !/^\+7\d{10}$/.test(phone)) { tg.showAlert("Пожалуйста, заполните имя и телефон!"); return; }
    if (!selectedTime || !selectedDate) { tg.showAlert("Выберите дату и время!"); return; }
    if (isSlotUnavailable(selectedTime)) { tg.showAlert("Это время уже прошло, выберите другое"); return; }
    if (!selectedServiceName) { tg.showAlert("Ошибка: услуга не выбрана!"); return; }
    const user = tg.initDataUnsafe?.user;
    if (!user || !user.id) { tg.showAlert("Ошибка: не удалось получить данные пользователя из Telegram!"); return; }
    tg.MainButton.showProgress();
    const bookingData = {
        serviceName: selectedServiceName, date: selectedDate, time: selectedTime,
        userId: user.id, name: name, phone: phone, comment: comment, initData: tg.initData || ""
    };
    fetch('/api/book', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(bookingData)
    })
    .then(response => {
        if (!response.ok) { return response.text().then(text => { throw new Error(text || 'Ошибка сервера'); }); }
        return response.json();
    })
    .then(data => {
        tg.MainButton.hideProgress();
        if (data.status === 'pending_confirmation') {
            tg.showAlert("Проверьте Telegram - требуется подтверждение замены записи");
        } else {
            tg.showAlert("Запись успешно создана! ✅");
        }
        setTimeout(() => { tg.close(); }, 500);
    })
    .catch(error => {
        console.error('Ошибка при записи:', error);
        tg.MainButton.hideProgress();
        tg.showAlert("Ошибка при записи: " + error.message);
    });
}

function toggleDescription(event, btn) {
    event.stopPropagation(); // Важно: предотвращаем клик по самой карточке (запись)
    const desc = btn.nextElementSibling;
    if (desc.style.display === 'block') {
        desc.style.display = 'none';
        btn.innerText = 'Подробнее об услуге';
    } else {
        desc.style.display = 'block';
        btn.innerText = 'Скрыть описание';
    }
}
