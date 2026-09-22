package service

import (
	"context"
	"hookah-bot/internal/domain"
)

type BookingSvc struct {
	repo domain.BookingRepository
}

func NewBookingService(repo domain.BookingRepository) *BookingSvc {
	return &BookingSvc{repo: repo}
}

func (s *BookingSvc) StartBookingDraft(ctx context.Context, userID int64, serviceName string) error {
	return s.repo.SaveDraft(ctx, userID, serviceName)
}

func (s *BookingSvc) SetBookingDate(ctx context.Context, userID int64, date string) error {
	return s.repo.SetDraftDate(ctx, userID, date)
}

func (s *BookingSvc) SetDraftTimeAndContacts(ctx context.Context, userID int64, timeSlot string, name string, phone string, comment string) error {
	return s.repo.SetDraftTimeAndContacts(ctx, userID, timeSlot, name, phone, comment)
}

func (s *BookingSvc) CompleteBookingDraft(ctx context.Context, userID int64, timeSlot string, name string, phone string, comment string) (*domain.Booking, error) {
	return s.repo.CompleteBooking(ctx, userID, timeSlot, name, phone, comment)
}

func (s *BookingSvc) GetUserBooking(ctx context.Context, userID int64) (*domain.Booking, error) {
	return s.repo.GetByUserID(ctx, userID)
}

func (s *BookingSvc) GetUserDraft(ctx context.Context, userID int64) (*domain.Booking, error) {
	return s.repo.GetDraftByUserID(ctx, userID)
}

func (s *BookingSvc) CancelBooking(ctx context.Context, userID int64) error {
	return s.repo.Delete(ctx, userID)
}

func (s *BookingSvc) CancelConfirmedBooking(ctx context.Context, userID int64) error {
	return s.repo.DeleteConfirmed(ctx, userID)
}

func (s *BookingSvc) CancelDraftBooking(ctx context.Context, userID int64) error {
	return s.repo.DeleteDraft(ctx, userID)
}

func (s *BookingSvc) DeleteBookingByID(ctx context.Context, id int) error {
	return s.repo.DeleteBookingByID(ctx, id)
}

func (s *BookingSvc) GetAllActiveBookings(ctx context.Context) ([]domain.Booking, error) {
	return s.repo.GetAllActive(ctx)
}

func (s *BookingSvc) ResetAllBookings(ctx context.Context) error {
	return s.repo.ResetAll(ctx)
}
