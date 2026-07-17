package main

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// BookingService owns validation, idempotency and the customer booking lifecycle.
type BookingService struct {
	store BookingStore
	idem  idempotencyStore
}

func NewBookingService(store BookingStore, idem idempotencyStore) *BookingService {
	return &BookingService{store: store, idem: idem}
}

func requireBookingKey(key string) error {
	if strings.TrimSpace(key) == "" {
		return ErrMissingIdempotencyKey
	}
	return nil
}

func (s *BookingService) CreateBooking(ctx context.Context, input CreateBookingInput, key string) (Booking, error) {
	if err := requireBookingKey(key); err != nil {
		return Booking{}, err
	}
	if strings.TrimSpace(input.ServiceID) == "" || strings.TrimSpace(input.SlotID) == "" || strings.TrimSpace(input.CustomerName) == "" || strings.TrimSpace(input.StartsAt) == "" || strings.TrimSpace(input.EndsAt) == "" {
		return Booking{}, fmt.Errorf("%w: service, slot, customer and time are required", ErrInvalidInput)
	}
	resourceKey := "booking:create:" + key
	if existing, ok, err := s.idem.Get(ctx, resourceKey); err != nil {
		return Booking{}, err
	} else if ok {
		return s.store.GetBooking(ctx, existing)
	}
	service, err := s.store.GetService(ctx, input.ServiceID)
	if err != nil {
		return Booking{}, err
	}
	if !service.Active {
		return Booking{}, fmt.Errorf("%w: service is inactive", ErrInvalidInput)
	}
	release, err := s.idem.Lock(ctx, "booking:slot-lock:"+input.ServiceID+":"+input.SlotID, 10*time.Second)
	if err != nil {
		return Booking{}, err
	}
	defer release()
	if existing, ok, err := s.idem.Get(ctx, resourceKey); err != nil {
		return Booking{}, err
	} else if ok {
		return s.store.GetBooking(ctx, existing)
	}
	booking, err := s.store.CreateBooking(ctx, Booking{ServiceID: input.ServiceID, ServiceName: service.Name, SlotID: input.SlotID, CustomerID: input.CustomerID, CustomerName: input.CustomerName, CustomerPhone: input.CustomerPhone, StartsAt: input.StartsAt, EndsAt: input.EndsAt, Status: BookingPending, PaymentStatus: PaymentPending, AmountCents: service.PriceCents})
	if err != nil {
		return Booking{}, err
	}
	if err := s.idem.Set(ctx, resourceKey, booking.ID, 24*time.Hour); err != nil {
		return Booking{}, err
	}
	return booking, nil
}

func (s *BookingService) UpdateBookingStatus(ctx context.Context, id, status, actor, key string) (Booking, error) {
	if err := requireBookingKey(key); err != nil {
		return Booking{}, err
	}
	status = strings.TrimSpace(status)
	if !validBookingStatus(status) {
		return Booking{}, fmt.Errorf("%w: unknown status", ErrInvalidInput)
	}
	resourceKey := "booking:status:" + id + ":" + key
	if existing, ok, err := s.idem.Get(ctx, resourceKey); err != nil {
		return Booking{}, err
	} else if ok {
		return s.store.GetBooking(ctx, existing)
	}
	release, err := s.idem.Lock(ctx, "booking:status-lock:"+id, 10*time.Second)
	if err != nil {
		return Booking{}, err
	}
	defer release()
	if existing, ok, err := s.idem.Get(ctx, resourceKey); err != nil {
		return Booking{}, err
	} else if ok {
		return s.store.GetBooking(ctx, existing)
	}
	booking, _, err := s.store.UpdateBookingStatus(ctx, id, status, actor)
	if err != nil {
		return Booking{}, err
	}
	if err := s.idem.Set(ctx, resourceKey, booking.ID, 24*time.Hour); err != nil {
		return Booking{}, err
	}
	return booking, nil
}

func (s *BookingService) RescheduleBooking(ctx context.Context, id string, input RescheduleBookingInput, key string) (Booking, error) {
	if err := requireBookingKey(key); err != nil {
		return Booking{}, err
	}
	if strings.TrimSpace(input.SlotID) == "" || strings.TrimSpace(input.StartsAt) == "" || strings.TrimSpace(input.EndsAt) == "" {
		return Booking{}, fmt.Errorf("%w: slot and time are required", ErrInvalidInput)
	}
	resourceKey := "booking:reschedule:" + id + ":" + key
	if existing, ok, err := s.idem.Get(ctx, resourceKey); err != nil {
		return Booking{}, err
	} else if ok {
		return s.store.GetBooking(ctx, existing)
	}
	booking, err := s.store.GetBooking(ctx, id)
	if err != nil {
		return Booking{}, err
	}
	release, err := s.idem.Lock(ctx, "booking:slot-lock:"+booking.ServiceID+":"+input.SlotID, 10*time.Second)
	if err != nil {
		return Booking{}, err
	}
	defer release()
	if existing, ok, err := s.idem.Get(ctx, resourceKey); err != nil {
		return Booking{}, err
	} else if ok {
		return s.store.GetBooking(ctx, existing)
	}
	updated, _, err := s.store.RescheduleBooking(ctx, id, input.SlotID, input.StartsAt, input.EndsAt)
	if err != nil {
		return Booking{}, err
	}
	if err := s.idem.Set(ctx, resourceKey, updated.ID, 24*time.Hour); err != nil {
		return Booking{}, err
	}
	return updated, nil
}

func (s *BookingService) RefundBooking(ctx context.Context, id, key string) (Booking, error) {
	if err := requireBookingKey(key); err != nil {
		return Booking{}, err
	}
	resourceKey := "booking:refund:" + id + ":" + key
	if existing, ok, err := s.idem.Get(ctx, resourceKey); err != nil {
		return Booking{}, err
	} else if ok {
		return s.store.GetBooking(ctx, existing)
	}
	release, err := s.idem.Lock(ctx, "booking:refund-lock:"+id, 10*time.Second)
	if err != nil {
		return Booking{}, err
	}
	defer release()
	if existing, ok, err := s.idem.Get(ctx, resourceKey); err != nil {
		return Booking{}, err
	} else if ok {
		return s.store.GetBooking(ctx, existing)
	}
	booking, _, err := s.store.RefundBooking(ctx, id)
	if err != nil {
		return Booking{}, err
	}
	if err := s.idem.Set(ctx, resourceKey, booking.ID, 24*time.Hour); err != nil {
		return Booking{}, err
	}
	return booking, nil
}

func (s *BookingService) CreateReview(ctx context.Context, id string, input CreateReviewInput, key string) (BookingReview, error) {
	if err := requireBookingKey(key); err != nil {
		return BookingReview{}, err
	}
	if input.Rating < 1 || input.Rating > 5 || strings.TrimSpace(input.Content) == "" {
		return BookingReview{}, fmt.Errorf("%w: rating must be 1-5 and content is required", ErrInvalidInput)
	}
	resourceKey := "booking:review:" + id + ":" + key
	if existing, ok, err := s.idem.Get(ctx, resourceKey); err != nil {
		return BookingReview{}, err
	} else if ok {
		return s.store.GetReview(ctx, existing)
	}
	release, err := s.idem.Lock(ctx, "booking:review-lock:"+id, 10*time.Second)
	if err != nil {
		return BookingReview{}, err
	}
	defer release()
	if existing, ok, err := s.idem.Get(ctx, resourceKey); err != nil {
		return BookingReview{}, err
	} else if ok {
		return s.store.GetReview(ctx, existing)
	}
	review, err := s.store.CreateReview(ctx, BookingReview{BookingID: id, Rating: input.Rating, Content: input.Content})
	if err != nil {
		return BookingReview{}, err
	}
	if err := s.idem.Set(ctx, resourceKey, review.ID, 24*time.Hour); err != nil {
		return BookingReview{}, err
	}
	return review, nil
}
