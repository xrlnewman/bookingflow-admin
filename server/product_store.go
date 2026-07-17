package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

// BookingStore contains the service catalog and customer booking workflow.
// It is implemented by both MemoryStore and SQLStore so the API can run in
// hermetic tests or against MySQL without changing service-layer behavior.
type BookingStore interface {
	ListServices(context.Context, string, bool) ([]ServiceOffering, error)
	GetService(context.Context, string) (ServiceOffering, error)
	ListAvailability(context.Context, string, string) ([]AvailabilitySlot, error)
	GetAvailability(context.Context, string) (AvailabilitySlot, error)
	ListBookings(context.Context, int, int, string, string) ([]Booking, int, error)
	GetBooking(context.Context, string) (Booking, error)
	CreateBooking(context.Context, Booking) (Booking, error)
	UpdateBookingStatus(context.Context, string, string, string) (Booking, BookingEvent, error)
	RescheduleBooking(context.Context, string, string, string, string) (Booking, BookingEvent, error)
	RefundBooking(context.Context, string) (Booking, BookingEvent, error)
	ListBookingEvents(context.Context, string) ([]BookingEvent, error)
	CreateReview(context.Context, BookingReview) (BookingReview, error)
	GetReview(context.Context, string) (BookingReview, error)
}

func seedProductData(s *MemoryStore) {
	created := nowUTC()
	s.services = map[string]ServiceOffering{
		"svc-cleaning": {ID: "svc-cleaning", Name: "深度保洁", Category: "家政保洁", Description: "厨房、卫生间重点清洁，服务前确认清单", DurationMinutes: 120, PriceCents: 19900, Active: true, CreatedAt: created, UpdatedAt: created},
		"svc-aircon":   {ID: "svc-aircon", Name: "空调清洗", Category: "家电清洗", Description: "拆洗滤网与蒸发器，清新一夏", DurationMinutes: 90, PriceCents: 12900, Active: true, CreatedAt: created, UpdatedAt: created},
		"svc-install":  {ID: "svc-install", Name: "家电安装", Category: "上门维修", Description: "水电、门锁、小家电快速排障", DurationMinutes: 120, PriceCents: 8900, Active: true, CreatedAt: created, UpdatedAt: created},
	}
	s.availability = map[string]AvailabilitySlot{
		"slot-cleaning-0900": {ID: "slot-cleaning-0900", ServiceID: "svc-cleaning", StartsAt: "2026-07-20T09:00:00+08:00", EndsAt: "2026-07-20T11:00:00+08:00", Capacity: 1, Remaining: 1, Status: AvailabilityOpen},
		"slot-cleaning-1300": {ID: "slot-cleaning-1300", ServiceID: "svc-cleaning", StartsAt: "2026-07-20T13:00:00+08:00", EndsAt: "2026-07-20T15:00:00+08:00", Capacity: 2, Remaining: 2, Status: AvailabilityOpen},
		"slot-aircon-1000":   {ID: "slot-aircon-1000", ServiceID: "svc-aircon", StartsAt: "2026-07-20T10:00:00+08:00", EndsAt: "2026-07-20T11:30:00+08:00", Capacity: 2, Remaining: 2, Status: AvailabilityOpen},
		"slot-install-1500":  {ID: "slot-install-1500", ServiceID: "svc-install", StartsAt: "2026-07-20T15:00:00+08:00", EndsAt: "2026-07-20T17:00:00+08:00", Capacity: 1, Remaining: 1, Status: AvailabilityOpen},
	}
}

func (s *MemoryStore) ListServices(_ context.Context, category string, activeOnly bool) ([]ServiceOffering, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]ServiceOffering, 0, len(s.services))
	for _, service := range s.services {
		if activeOnly && !service.Active {
			continue
		}
		if category != "" && service.Category != category {
			continue
		}
		out = append(out, service)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *MemoryStore) GetService(_ context.Context, id string) (ServiceOffering, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	service, ok := s.services[id]
	if !ok {
		return ServiceOffering{}, ErrNotFound
	}
	return service, nil
}

func slotStatus(slot AvailabilitySlot) string {
	if slot.Remaining <= 0 {
		return AvailabilityFull
	}
	if parsed, err := time.Parse(time.RFC3339, slot.EndsAt); err == nil && parsed.Before(time.Now()) {
		return AvailabilityClosed
	}
	return AvailabilityOpen
}

func (s *MemoryStore) ListAvailability(_ context.Context, serviceID, date string) ([]AvailabilitySlot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]AvailabilitySlot, 0, len(s.availability))
	for _, slot := range s.availability {
		if serviceID != "" && slot.ServiceID != serviceID {
			continue
		}
		if date != "" && !strings.HasPrefix(slot.StartsAt, date) {
			continue
		}
		slot.Status = slotStatus(slot)
		out = append(out, slot)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartsAt < out[j].StartsAt })
	return out, nil
}

func (s *MemoryStore) GetAvailability(_ context.Context, id string) (AvailabilitySlot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	slot, ok := s.availability[id]
	if !ok {
		return AvailabilitySlot{}, ErrNotFound
	}
	slot.Status = slotStatus(slot)
	return slot, nil
}

func (s *MemoryStore) ListBookings(_ context.Context, page, pageSize int, status, customerID string) ([]Booking, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	all := make([]Booking, 0, len(s.bookings))
	for _, booking := range s.bookings {
		if status != "" && booking.Status != status {
			continue
		}
		if customerID != "" && booking.CustomerID != customerID {
			continue
		}
		all = append(all, booking)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].StartsAt < all[j].StartsAt })
	return paginate(all, page, pageSize)
}

func (s *MemoryStore) GetBooking(_ context.Context, id string) (Booking, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	booking, ok := s.bookings[id]
	if !ok {
		return Booking{}, ErrNotFound
	}
	return booking, nil
}

func (s *MemoryStore) CreateBooking(_ context.Context, booking Booking) (Booking, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if booking.ID == "" {
		booking.ID = s.next("BK")
	}
	service, ok := s.services[booking.ServiceID]
	if !ok || !service.Active {
		return Booking{}, ErrNotFound
	}
	slot, ok := s.availability[booking.SlotID]
	if !ok || slot.ServiceID != booking.ServiceID || slot.Remaining <= 0 {
		return Booking{}, ErrSlotUnavailable
	}
	if booking.ServiceName == "" {
		booking.ServiceName = service.Name
	}
	if booking.AmountCents == 0 {
		booking.AmountCents = service.PriceCents
	}
	if booking.Status == "" {
		booking.Status = BookingPending
	}
	if booking.PaymentStatus == "" {
		booking.PaymentStatus = PaymentPending
	}
	if booking.CreatedAt == "" {
		booking.CreatedAt = nowUTC()
	}
	booking.UpdatedAt = booking.CreatedAt
	slot.Remaining--
	slot.Status = slotStatus(slot)
	s.availability[slot.ID] = slot
	s.bookings[booking.ID] = booking
	s.bookingEvents[booking.ID] = append(s.bookingEvents[booking.ID], BookingEvent{ID: s.next("BE"), BookingID: booking.ID, EventType: "created", ToStatus: booking.Status, Actor: "customer", CreatedAt: nowUTC()})
	return booking, nil
}

func (s *MemoryStore) UpdateBookingStatus(_ context.Context, id, status, actor string) (Booking, BookingEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	booking, ok := s.bookings[id]
	if !ok {
		return Booking{}, BookingEvent{}, ErrNotFound
	}
	if !bookingTransitions[booking.Status][status] {
		return Booking{}, BookingEvent{}, ErrInvalidBookingTransition
	}
	if actor == "" {
		actor = "运营人员"
	}
	old := booking.Status
	booking.Status = status
	booking.UpdatedAt = nowUTC()
	s.bookings[id] = booking
	event := BookingEvent{ID: s.next("BE"), BookingID: id, EventType: "status_changed", FromStatus: old, ToStatus: status, Actor: actor, CreatedAt: nowUTC()}
	s.bookingEvents[id] = append(s.bookingEvents[id], event)
	return booking, event, nil
}

func (s *MemoryStore) RescheduleBooking(_ context.Context, id, slotID, startsAt, endsAt string) (Booking, BookingEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	booking, ok := s.bookings[id]
	if !ok {
		return Booking{}, BookingEvent{}, ErrNotFound
	}
	target, ok := s.availability[slotID]
	if !ok || target.ServiceID != booking.ServiceID {
		return Booking{}, BookingEvent{}, ErrSlotUnavailable
	}
	if slotID != booking.SlotID && target.Remaining <= 0 {
		return Booking{}, BookingEvent{}, ErrSlotUnavailable
	}
	if slotID != booking.SlotID {
		if current, exists := s.availability[booking.SlotID]; exists {
			current.Remaining++
			current.Status = slotStatus(current)
			s.availability[current.ID] = current
		}
		target.Remaining--
		target.Status = slotStatus(target)
		s.availability[target.ID] = target
	}
	oldSlot, oldStart, oldEnd := booking.SlotID, booking.StartsAt, booking.EndsAt
	booking.SlotID, booking.StartsAt, booking.EndsAt, booking.UpdatedAt = slotID, startsAt, endsAt, nowUTC()
	s.bookings[id] = booking
	event := BookingEvent{ID: s.next("BE"), BookingID: id, EventType: "rescheduled", Actor: "customer", Note: fmt.Sprintf("%s %s-%s -> %s %s-%s", oldSlot, oldStart, oldEnd, slotID, startsAt, endsAt), CreatedAt: nowUTC()}
	s.bookingEvents[id] = append(s.bookingEvents[id], event)
	return booking, event, nil
}

func (s *MemoryStore) RefundBooking(_ context.Context, id string) (Booking, BookingEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	booking, ok := s.bookings[id]
	if !ok {
		return Booking{}, BookingEvent{}, ErrNotFound
	}
	if booking.PaymentStatus == PaymentRefunded {
		return Booking{}, BookingEvent{}, ErrBookingAlreadyRefunded
	}
	booking.PaymentStatus = PaymentRefunded
	booking.UpdatedAt = nowUTC()
	s.bookings[id] = booking
	event := BookingEvent{ID: s.next("BE"), BookingID: id, EventType: "refunded", Actor: "customer", CreatedAt: nowUTC()}
	s.bookingEvents[id] = append(s.bookingEvents[id], event)
	return booking, event, nil
}

func (s *MemoryStore) ListBookingEvents(_ context.Context, id string) ([]BookingEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.bookings[id]; !ok {
		return nil, ErrNotFound
	}
	return append([]BookingEvent(nil), s.bookingEvents[id]...), nil
}

func (s *MemoryStore) CreateReview(_ context.Context, review BookingReview) (BookingReview, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	booking, ok := s.bookings[review.BookingID]
	if !ok {
		return BookingReview{}, ErrNotFound
	}
	if booking.Status != BookingCompleted {
		return BookingReview{}, fmt.Errorf("%w: booking must be completed", ErrInvalidInput)
	}
	if _, exists := s.reviews[review.BookingID]; exists {
		return BookingReview{}, ErrReviewExists
	}
	if review.Rating < 1 || review.Rating > 5 || strings.TrimSpace(review.Content) == "" {
		return BookingReview{}, ErrInvalidInput
	}
	if review.ID == "" {
		review.ID = s.next("REV")
	}
	if review.CustomerName == "" {
		review.CustomerName = booking.CustomerName
	}
	review.CreatedAt = nowUTC()
	s.reviews[review.BookingID] = review
	s.bookingEvents[review.BookingID] = append(s.bookingEvents[review.BookingID], BookingEvent{ID: s.next("BE"), BookingID: review.BookingID, EventType: "reviewed", Actor: review.CustomerName, Note: review.Content, CreatedAt: review.CreatedAt})
	return review, nil
}

func (s *MemoryStore) GetReview(_ context.Context, id string) (BookingReview, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, review := range s.reviews {
		if review.ID == id {
			return review, nil
		}
	}
	return BookingReview{}, ErrNotFound
}

// Keep compile-time guarantees that the two stores expose the product API.
var _ BookingStore = (*MemoryStore)(nil)
var _ BookingStore = (*SQLStore)(nil)

// SQL implementations are in product_sql_store.go.
