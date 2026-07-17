package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type rowScanner interface{ Scan(...any) error }

func scanService(row rowScanner) (ServiceOffering, error) {
	var service ServiceOffering
	var active bool
	err := row.Scan(&service.ID, &service.Name, &service.Category, &service.Description, &service.DurationMinutes, &service.PriceCents, &active, &service.CreatedAt, &service.UpdatedAt)
	service.Active = active
	return service, err
}

func scanSlot(row rowScanner) (AvailabilitySlot, error) {
	var slot AvailabilitySlot
	err := row.Scan(&slot.ID, &slot.ServiceID, &slot.StartsAt, &slot.EndsAt, &slot.Capacity, &slot.Remaining, &slot.Status)
	return slot, err
}

func scanBooking(row rowScanner) (Booking, error) {
	var booking Booking
	err := row.Scan(&booking.ID, &booking.ServiceID, &booking.ServiceName, &booking.SlotID, &booking.CustomerID, &booking.CustomerName, &booking.CustomerPhone, &booking.StartsAt, &booking.EndsAt, &booking.Status, &booking.PaymentStatus, &booking.AmountCents, &booking.CreatedAt, &booking.UpdatedAt)
	return booking, err
}

func (s *SQLStore) ListServices(ctx context.Context, category string, activeOnly bool) ([]ServiceOffering, error) {
	query := `SELECT id,name,category,description,duration_minutes,price_cents,active,created_at,updated_at FROM services`
	args := []any{}
	conditions := []string{}
	if category != "" {
		conditions = append(conditions, "category=?")
		args = append(args, category)
	}
	if activeOnly {
		conditions = append(conditions, "active=1")
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY name"
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []ServiceOffering{}
	for rows.Next() {
		service, err := scanService(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, service)
	}
	return result, rows.Err()
}

func (s *SQLStore) GetService(ctx context.Context, id string) (ServiceOffering, error) {
	service, err := scanService(s.db.QueryRowContext(ctx, `SELECT id,name,category,description,duration_minutes,price_cents,active,created_at,updated_at FROM services WHERE id=?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return ServiceOffering{}, ErrNotFound
	}
	return service, err
}

func (s *SQLStore) ListAvailability(ctx context.Context, serviceID, date string) ([]AvailabilitySlot, error) {
	query := `SELECT id,service_id,starts_at,ends_at,capacity,remaining,status FROM availability_slots`
	args := []any{}
	conditions := []string{}
	if serviceID != "" {
		conditions = append(conditions, "service_id=?")
		args = append(args, serviceID)
	}
	if date != "" {
		conditions = append(conditions, "starts_at LIKE CONCAT(?,'%')")
		args = append(args, date)
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY starts_at"
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []AvailabilitySlot{}
	for rows.Next() {
		slot, err := scanSlot(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, slot)
	}
	return result, rows.Err()
}

func (s *SQLStore) GetAvailability(ctx context.Context, id string) (AvailabilitySlot, error) {
	slot, err := scanSlot(s.db.QueryRowContext(ctx, `SELECT id,service_id,starts_at,ends_at,capacity,remaining,status FROM availability_slots WHERE id=?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return AvailabilitySlot{}, ErrNotFound
	}
	return slot, err
}

func (s *SQLStore) ListBookings(ctx context.Context, page, pageSize int, status, customerID string) ([]Booking, int, error) {
	conditions := []string{}
	args := []any{}
	if status != "" {
		conditions = append(conditions, "status=?")
		args = append(args, status)
	}
	if customerID != "" {
		conditions = append(conditions, "customer_id=?")
		args = append(args, customerID)
	}
	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}
	var total int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM bookings"+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	page, pageSize = normalizePage(page, pageSize)
	queryArgs := append([]any{}, args...)
	queryArgs = append(queryArgs, pageSize, (page-1)*pageSize)
	rows, err := s.db.QueryContext(ctx, `SELECT id,service_id,service_name,slot_id,customer_id,customer_name,customer_phone,starts_at,ends_at,status,payment_status,amount_cents,created_at,updated_at FROM bookings`+where+` ORDER BY starts_at LIMIT ? OFFSET ?`, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	result := []Booking{}
	for rows.Next() {
		booking, err := scanBooking(rows)
		if err != nil {
			return nil, 0, err
		}
		result = append(result, booking)
	}
	return result, total, rows.Err()
}

func (s *SQLStore) GetBooking(ctx context.Context, id string) (Booking, error) {
	booking, err := scanBooking(s.db.QueryRowContext(ctx, `SELECT id,service_id,service_name,slot_id,customer_id,customer_name,customer_phone,starts_at,ends_at,status,payment_status,amount_cents,created_at,updated_at FROM bookings WHERE id=?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Booking{}, ErrNotFound
	}
	return booking, err
}

func (s *SQLStore) CreateBooking(ctx context.Context, booking Booking) (Booking, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Booking{}, err
	}
	defer tx.Rollback()
	var slot AvailabilitySlot
	slot, err = scanSlot(tx.QueryRowContext(ctx, `SELECT id,service_id,starts_at,ends_at,capacity,remaining,status FROM availability_slots WHERE id=? FOR UPDATE`, booking.SlotID))
	if errors.Is(err, sql.ErrNoRows) {
		return Booking{}, ErrSlotUnavailable
	}
	if err != nil {
		return Booking{}, err
	}
	if slot.ServiceID != booking.ServiceID || slot.Remaining <= 0 {
		return Booking{}, ErrSlotUnavailable
	}
	if booking.ID == "" {
		booking.ID = fmt.Sprintf("BK-%d", time.Now().UnixNano())
	}
	if booking.CreatedAt == "" {
		booking.CreatedAt = nowUTC()
	}
	booking.UpdatedAt = booking.CreatedAt
	if booking.Status == "" {
		booking.Status = BookingPending
	}
	if booking.PaymentStatus == "" {
		booking.PaymentStatus = PaymentPending
	}
	slot.Remaining--
	slot.Status = slotStatus(slot)
	if _, err = tx.ExecContext(ctx, `UPDATE availability_slots SET remaining=?,status=? WHERE id=?`, slot.Remaining, slot.Status, slot.ID); err != nil {
		return Booking{}, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO bookings (id,service_id,service_name,slot_id,customer_id,customer_name,customer_phone,starts_at,ends_at,status,payment_status,amount_cents,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, booking.ID, booking.ServiceID, booking.ServiceName, booking.SlotID, booking.CustomerID, booking.CustomerName, booking.CustomerPhone, booking.StartsAt, booking.EndsAt, booking.Status, booking.PaymentStatus, booking.AmountCents, booking.CreatedAt, booking.UpdatedAt); err != nil {
		return Booking{}, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO booking_events (id,booking_id,event_type,from_status,to_status,actor,note,created_at) VALUES (?,?,?,?,?,?,?,?)`, fmt.Sprintf("BE-%d", time.Now().UnixNano()), booking.ID, "created", "", booking.Status, "customer", "", nowUTC()); err != nil {
		return Booking{}, err
	}
	if err = tx.Commit(); err != nil {
		return Booking{}, err
	}
	return booking, nil
}

func (s *SQLStore) UpdateBookingStatus(ctx context.Context, id, status, actor string) (Booking, BookingEvent, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Booking{}, BookingEvent{}, err
	}
	defer tx.Rollback()
	booking, err := scanBooking(tx.QueryRowContext(ctx, `SELECT id,service_id,service_name,slot_id,customer_id,customer_name,customer_phone,starts_at,ends_at,status,payment_status,amount_cents,created_at,updated_at FROM bookings WHERE id=? FOR UPDATE`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Booking{}, BookingEvent{}, ErrNotFound
	}
	if err != nil {
		return Booking{}, BookingEvent{}, err
	}
	if !bookingTransitions[booking.Status][status] {
		return Booking{}, BookingEvent{}, ErrInvalidBookingTransition
	}
	if actor == "" {
		actor = "运营人员"
	}
	old := booking.Status
	booking.Status, booking.UpdatedAt = status, nowUTC()
	if _, err = tx.ExecContext(ctx, `UPDATE bookings SET status=?,updated_at=? WHERE id=?`, booking.Status, booking.UpdatedAt, id); err != nil {
		return Booking{}, BookingEvent{}, err
	}
	event := BookingEvent{ID: fmt.Sprintf("BE-%d", time.Now().UnixNano()), BookingID: id, EventType: "status_changed", FromStatus: old, ToStatus: status, Actor: actor, CreatedAt: nowUTC()}
	if _, err = tx.ExecContext(ctx, `INSERT INTO booking_events (id,booking_id,event_type,from_status,to_status,actor,note,created_at) VALUES (?,?,?,?,?,?,?,?)`, event.ID, id, event.EventType, event.FromStatus, event.ToStatus, event.Actor, event.Note, event.CreatedAt); err != nil {
		return Booking{}, BookingEvent{}, err
	}
	if err = tx.Commit(); err != nil {
		return Booking{}, BookingEvent{}, err
	}
	return booking, event, nil
}

func (s *SQLStore) RescheduleBooking(ctx context.Context, id, slotID, startsAt, endsAt string) (Booking, BookingEvent, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Booking{}, BookingEvent{}, err
	}
	defer tx.Rollback()
	booking, err := scanBooking(tx.QueryRowContext(ctx, `SELECT id,service_id,service_name,slot_id,customer_id,customer_name,customer_phone,starts_at,ends_at,status,payment_status,amount_cents,created_at,updated_at FROM bookings WHERE id=? FOR UPDATE`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Booking{}, BookingEvent{}, ErrNotFound
	}
	if err != nil {
		return Booking{}, BookingEvent{}, err
	}
	var target AvailabilitySlot
	target, err = scanSlot(tx.QueryRowContext(ctx, `SELECT id,service_id,starts_at,ends_at,capacity,remaining,status FROM availability_slots WHERE id=? FOR UPDATE`, slotID))
	if errors.Is(err, sql.ErrNoRows) {
		return Booking{}, BookingEvent{}, ErrSlotUnavailable
	}
	if err != nil {
		return Booking{}, BookingEvent{}, err
	}
	if target.ServiceID != booking.ServiceID || (slotID != booking.SlotID && target.Remaining <= 0) {
		return Booking{}, BookingEvent{}, ErrSlotUnavailable
	}
	if slotID != booking.SlotID {
		var old AvailabilitySlot
		old, err = scanSlot(tx.QueryRowContext(ctx, `SELECT id,service_id,starts_at,ends_at,capacity,remaining,status FROM availability_slots WHERE id=? FOR UPDATE`, booking.SlotID))
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return Booking{}, BookingEvent{}, err
		}
		if old.ID != "" {
			old.Remaining++
			old.Status = slotStatus(old)
			if _, err = tx.ExecContext(ctx, `UPDATE availability_slots SET remaining=?,status=? WHERE id=?`, old.Remaining, old.Status, old.ID); err != nil {
				return Booking{}, BookingEvent{}, err
			}
		}
		target.Remaining--
		target.Status = slotStatus(target)
		if _, err = tx.ExecContext(ctx, `UPDATE availability_slots SET remaining=?,status=? WHERE id=?`, target.Remaining, target.Status, target.ID); err != nil {
			return Booking{}, BookingEvent{}, err
		}
	}
	oldSlot, oldStarts, oldEnds := booking.SlotID, booking.StartsAt, booking.EndsAt
	booking.SlotID, booking.StartsAt, booking.EndsAt, booking.UpdatedAt = slotID, startsAt, endsAt, nowUTC()
	if _, err = tx.ExecContext(ctx, `UPDATE bookings SET slot_id=?,starts_at=?,ends_at=?,updated_at=? WHERE id=?`, booking.SlotID, booking.StartsAt, booking.EndsAt, booking.UpdatedAt, id); err != nil {
		return Booking{}, BookingEvent{}, err
	}
	event := BookingEvent{ID: fmt.Sprintf("BE-%d", time.Now().UnixNano()), BookingID: id, EventType: "rescheduled", Actor: "customer", Note: fmt.Sprintf("%s %s-%s -> %s %s-%s", oldSlot, oldStarts, oldEnds, slotID, startsAt, endsAt), CreatedAt: nowUTC()}
	if _, err = tx.ExecContext(ctx, `INSERT INTO booking_events (id,booking_id,event_type,from_status,to_status,actor,note,created_at) VALUES (?,?,?,?,?,?,?,?)`, event.ID, id, event.EventType, "", "", event.Actor, event.Note, event.CreatedAt); err != nil {
		return Booking{}, BookingEvent{}, err
	}
	if err = tx.Commit(); err != nil {
		return Booking{}, BookingEvent{}, err
	}
	return booking, event, nil
}

func (s *SQLStore) RefundBooking(ctx context.Context, id string) (Booking, BookingEvent, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Booking{}, BookingEvent{}, err
	}
	defer tx.Rollback()
	booking, err := scanBooking(tx.QueryRowContext(ctx, `SELECT id,service_id,service_name,slot_id,customer_id,customer_name,customer_phone,starts_at,ends_at,status,payment_status,amount_cents,created_at,updated_at FROM bookings WHERE id=? FOR UPDATE`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Booking{}, BookingEvent{}, ErrNotFound
	}
	if err != nil {
		return Booking{}, BookingEvent{}, err
	}
	if booking.PaymentStatus == PaymentRefunded {
		return Booking{}, BookingEvent{}, ErrBookingAlreadyRefunded
	}
	booking.PaymentStatus, booking.UpdatedAt = PaymentRefunded, nowUTC()
	if _, err = tx.ExecContext(ctx, `UPDATE bookings SET payment_status=?,updated_at=? WHERE id=?`, booking.PaymentStatus, booking.UpdatedAt, id); err != nil {
		return Booking{}, BookingEvent{}, err
	}
	event := BookingEvent{ID: fmt.Sprintf("BE-%d", time.Now().UnixNano()), BookingID: id, EventType: "refunded", Actor: "customer", CreatedAt: nowUTC()}
	if _, err = tx.ExecContext(ctx, `INSERT INTO booking_events (id,booking_id,event_type,from_status,to_status,actor,note,created_at) VALUES (?,?,?,?,?,?,?,?)`, event.ID, id, event.EventType, "", "", event.Actor, "", event.CreatedAt); err != nil {
		return Booking{}, BookingEvent{}, err
	}
	if err = tx.Commit(); err != nil {
		return Booking{}, BookingEvent{}, err
	}
	return booking, event, nil
}

func (s *SQLStore) ListBookingEvents(ctx context.Context, id string) ([]BookingEvent, error) {
	if _, err := s.GetBooking(ctx, id); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,booking_id,event_type,from_status,to_status,actor,note,created_at FROM booking_events WHERE booking_id=? ORDER BY created_at,id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []BookingEvent{}
	for rows.Next() {
		var event BookingEvent
		if err := rows.Scan(&event.ID, &event.BookingID, &event.EventType, &event.FromStatus, &event.ToStatus, &event.Actor, &event.Note, &event.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, event)
	}
	return result, rows.Err()
}

func (s *SQLStore) CreateReview(ctx context.Context, review BookingReview) (BookingReview, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return BookingReview{}, err
	}
	defer tx.Rollback()
	var booking Booking
	booking, err = scanBooking(tx.QueryRowContext(ctx, `SELECT id,service_id,service_name,slot_id,customer_id,customer_name,customer_phone,starts_at,ends_at,status,payment_status,amount_cents,created_at,updated_at FROM bookings WHERE id=? FOR UPDATE`, review.BookingID))
	if errors.Is(err, sql.ErrNoRows) {
		return BookingReview{}, ErrNotFound
	}
	if err != nil {
		return BookingReview{}, err
	}
	if booking.Status != BookingCompleted {
		return BookingReview{}, fmt.Errorf("%w: booking must be completed", ErrInvalidInput)
	}
	var exists string
	if err = tx.QueryRowContext(ctx, `SELECT id FROM booking_reviews WHERE booking_id=?`, review.BookingID).Scan(&exists); err == nil {
		return BookingReview{}, ErrReviewExists
	} else if !errors.Is(err, sql.ErrNoRows) {
		return BookingReview{}, err
	}
	if review.Rating < 1 || review.Rating > 5 || strings.TrimSpace(review.Content) == "" {
		return BookingReview{}, ErrInvalidInput
	}
	if review.ID == "" {
		review.ID = fmt.Sprintf("REV-%d", time.Now().UnixNano())
	}
	if review.CustomerName == "" {
		review.CustomerName = booking.CustomerName
	}
	review.CreatedAt = nowUTC()
	if _, err = tx.ExecContext(ctx, `INSERT INTO booking_reviews (id,booking_id,customer_name,rating,content,created_at) VALUES (?,?,?,?,?,?)`, review.ID, review.BookingID, review.CustomerName, review.Rating, review.Content, review.CreatedAt); err != nil {
		return BookingReview{}, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO booking_events (id,booking_id,event_type,from_status,to_status,actor,note,created_at) VALUES (?,?,?,?,?,?,?,?)`, fmt.Sprintf("BE-%d", time.Now().UnixNano()), review.BookingID, "reviewed", "", "", review.CustomerName, review.Content, review.CreatedAt); err != nil {
		return BookingReview{}, err
	}
	if err = tx.Commit(); err != nil {
		return BookingReview{}, err
	}
	return review, nil
}

func (s *SQLStore) GetReview(ctx context.Context, id string) (BookingReview, error) {
	var review BookingReview
	err := s.db.QueryRowContext(ctx, `SELECT id,booking_id,customer_name,rating,content,created_at FROM booking_reviews WHERE id=?`, id).Scan(&review.ID, &review.BookingID, &review.CustomerName, &review.Rating, &review.Content, &review.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return BookingReview{}, ErrNotFound
	}
	return review, err
}

var _ BookingStore = (*SQLStore)(nil)
