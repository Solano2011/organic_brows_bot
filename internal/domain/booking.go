package domain

import (
	"context"
	"errors"
	"time"
)

var (
	ErrBookingNotFound = errors.New("бронь не найдена")
	ErrTimeSlotTaken   = errors.New("это время уже занято")
)

type Booking struct {
	ID          string    `json:"id"`
	UserID      int64     `json:"user_id"`
	UserName    string    `json:"name"`
	Phone       string    `json:"phone"`
	ServiceName string    `json:"service_name"`
	Comment     string    `json:"comment"`
	TimeSlot    string    `json:"timeslot"`
	Date        string    `json:"date"`
	CreatedAt   time.Time `json:"created_at"`
}

type BookingRepository interface {
	SaveDraft(ctx context.Context, userID int64, serviceName string) error
	SetDraftDate(ctx context.Context, userID int64, date string) error
	SetDraftTimeAndContacts(ctx context.Context, userID int64, timeSlot string, name string, phone string, comment string) error
	CompleteBooking(ctx context.Context, userID int64, timeSlot string, name string, phone string, comment string) (*Booking, error)
	GetByUserID(ctx context.Context, userID int64) (*Booking, error)
	GetDraftByUserID(ctx context.Context, userID int64) (*Booking, error)
	Delete(ctx context.Context, userID int64) error
	DeleteConfirmed(ctx context.Context, userID int64) error
	DeleteDraft(ctx context.Context, userID int64) error
	GetAllActive(ctx context.Context) ([]Booking, error)
	GetTakenTimeSlots(ctx context.Context, date string, serviceName string) ([]string, error)
	ResetAll(ctx context.Context) error
}

type BookingService interface {
	StartBookingDraft(ctx context.Context, userID int64, serviceName string) error
	SetBookingDate(ctx context.Context, userID int64, date string) error
	SetDraftTimeAndContacts(ctx context.Context, userID int64, timeSlot string, name string, phone string, comment string) error
	CompleteBookingDraft(ctx context.Context, userID int64, timeSlot string, name string, phone string, comment string) (*Booking, error)
	GetUserBooking(ctx context.Context, userID int64) (*Booking, error)
	GetUserDraft(ctx context.Context, userID int64) (*Booking, error)
	CancelBooking(ctx context.Context, userID int64) error
	CancelConfirmedBooking(ctx context.Context, userID int64) error
	CancelDraftBooking(ctx context.Context, userID int64) error
	GetAllActiveBookings(ctx context.Context) ([]Booking, error)
	ResetAllBookings(ctx context.Context) error
}
