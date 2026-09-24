package memory

import (
	"context"
	"strconv"
	"sync"

	"hookah-bot/internal/domain"
)

type BookingRepo struct {
	mu     sync.Mutex
	drafts map[int64]*domain.Booking
}

func NewBookingRepo() *BookingRepo {
	return &BookingRepo{
		drafts: make(map[int64]*domain.Booking),
	}
}

func (r *BookingRepo) SaveDraft(ctx context.Context, userID int64, serviceName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.drafts[userID] = &domain.Booking{
		UserID:      userID,
		ServiceName: serviceName,
	}
	return nil
}

func (r *BookingRepo) SetDraftDate(ctx context.Context, userID int64, date string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	b, exists := r.drafts[userID]
	if !exists {
		return domain.ErrBookingNotFound
	}
	b.Date = date
	return nil
}

func (r *BookingRepo) SetDraftTimeAndContacts(ctx context.Context, userID int64, timeSlot string, name string, phone string, comment string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	b, exists := r.drafts[userID]
	if !exists {
		return domain.ErrBookingNotFound
	}
	b.TimeSlot = timeSlot
	b.UserName = name
	b.Phone = phone
	b.Comment = comment
	return nil
}

func (r *BookingRepo) CompleteBooking(ctx context.Context, userID int64, timeSlot string, name string, phone string, comment string) (*domain.Booking, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	b, exists := r.drafts[userID]
	if !exists {
		return nil, domain.ErrBookingNotFound
	}

	// Проверяем, не занято ли это время для данной услуги и даты
	for _, existing := range r.drafts {
		if existing.TimeSlot == timeSlot && existing.ServiceName == b.ServiceName && existing.Date == b.Date && existing.UserID != userID {
			return nil, domain.ErrTimeSlotTaken
		}
	}

	b.TimeSlot = timeSlot
	b.UserName = name
	b.Phone = phone
	b.Comment = comment

	return b, nil
}

func (r *BookingRepo) GetByUserID(ctx context.Context, userID int64) (*domain.Booking, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	b, exists := r.drafts[userID]
	if !exists || b.TimeSlot == "" {
		return nil, domain.ErrBookingNotFound
	}
	return b, nil
}

func (r *BookingRepo) GetDraftByUserID(ctx context.Context, userID int64) (*domain.Booking, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	b, exists := r.drafts[userID]
	if !exists {
		return nil, domain.ErrBookingNotFound
	}
	return b, nil
}

func (r *BookingRepo) Delete(ctx context.Context, userID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.drafts, userID)
	return nil
}

func (r *BookingRepo) DeleteConfirmed(ctx context.Context, userID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if b, exists := r.drafts[userID]; exists && b.TimeSlot != "" {
		delete(r.drafts, userID)
	}
	return nil
}

func (r *BookingRepo) DeleteDraft(ctx context.Context, userID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if b, exists := r.drafts[userID]; exists && b.TimeSlot == "" {
		delete(r.drafts, userID)
	}
	return nil
}

func (r *BookingRepo) GetByID(ctx context.Context, id int) (*domain.Booking, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, b := range r.drafts {
		if b.ID == strconv.Itoa(id) {
			copy := *b
			return &copy, nil
		}
	}
	return nil, domain.ErrBookingNotFound
}

func (r *BookingRepo) MarkReminder24hSent(ctx context.Context, id int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, b := range r.drafts {
		if b.ID == strconv.Itoa(id) {
			b.Reminder24hSent = true
			return nil
		}
	}
	return nil
}

func (r *BookingRepo) MarkReminder1hSent(ctx context.Context, id int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, b := range r.drafts {
		if b.ID == strconv.Itoa(id) {
			b.Reminder1hSent = true
			return nil
		}
	}
	return nil
}

func (r *BookingRepo) DeleteBookingByID(ctx context.Context, id int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for userID, b := range r.drafts {
		if b.ID == strconv.Itoa(id) {
			delete(r.drafts, userID)
			return nil
		}
	}
	return nil
}

func (r *BookingRepo) GetAllActive(ctx context.Context) ([]domain.Booking, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var active []domain.Booking
	for _, b := range r.drafts {
		if b.TimeSlot != "" {
			active = append(active, *b)
		}
	}
	return active, nil
}

func (r *BookingRepo) ResetAll(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.drafts = make(map[int64]*domain.Booking)
	return nil
}

func (r *BookingRepo) GetTakenTimeSlots(ctx context.Context, date string, serviceName string) ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var result []string
	for _, b := range r.drafts {
		if b.TimeSlot != "" && b.Date == date && b.ServiceName == serviceName {
			result = append(result, b.TimeSlot)
		}
	}
	return result, nil
}
