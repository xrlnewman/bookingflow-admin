package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBookingLifecycleIsIdempotentAndTimelineAware(t *testing.T) {
	store := NewMemoryStore()
	svc := NewBookingService(store, newMemoryIdempotency())
	ctx := context.Background()

	input := CreateBookingInput{ServiceID: "svc-cleaning", SlotID: "slot-cleaning-0900", CustomerID: "CUS-001", CustomerName: "杭州星河家庭", CustomerPhone: "13800000001", StartsAt: "2026-07-20T09:00:00+08:00", EndsAt: "2026-07-20T11:00:00+08:00"}
	first, err := svc.CreateBooking(ctx, input, "booking-create-1")
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.CreateBooking(ctx, input, "booking-create-1")
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID {
		t.Fatalf("idempotency returned %q then %q", first.ID, second.ID)
	}

	for _, status := range []string{BookingReserved, BookingCheckedIn, BookingServing, BookingCompleted} {
		first, err = svc.UpdateBookingStatus(ctx, first.ID, status, "customer", "status-"+status)
		if err != nil {
			t.Fatalf("status %s: %v", status, err)
		}
	}
	if _, err := svc.UpdateBookingStatus(ctx, first.ID, BookingServing, "customer", "illegal"); !errors.Is(err, ErrInvalidBookingTransition) {
		t.Fatalf("expected invalid booking transition, got %v", err)
	}
	if _, err := svc.RefundBooking(ctx, first.ID, "refund-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateReview(ctx, first.ID, CreateReviewInput{Rating: 5, Content: "服务准时，沟通顺畅"}, "review-1"); err != nil {
		t.Fatal(err)
	}

	events, err := store.ListBookingEvents(ctx, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) < 7 {
		t.Fatalf("events = %d, want at least 7", len(events))
	}
	if events[0].EventType != "created" || events[len(events)-1].EventType != "reviewed" {
		t.Fatalf("unexpected event timeline: %+v", events)
	}
}

func TestBookingServiceRejectsOverbookedSlot(t *testing.T) {
	store := NewMemoryStore()
	svc := NewBookingService(store, newMemoryIdempotency())
	input := CreateBookingInput{ServiceID: "svc-cleaning", SlotID: "slot-cleaning-0900", CustomerID: "CUS-001", CustomerName: "第一位客户", StartsAt: "2026-07-20T09:00:00+08:00", EndsAt: "2026-07-20T11:00:00+08:00"}
	if _, err := svc.CreateBooking(context.Background(), input, "slot-1"); err != nil {
		t.Fatal(err)
	}
	input.CustomerID, input.CustomerName = "CUS-002", "第二位客户"
	if _, err := svc.CreateBooking(context.Background(), input, "slot-2"); !errors.Is(err, ErrSlotUnavailable) {
		t.Fatalf("expected slot unavailable, got %v", err)
	}
}

func TestBookingRouterProductEndpoints(t *testing.T) {
	r := NewRouter(NewMemoryStore(), newMemoryIdempotency())
	services := httptest.NewRecorder()
	r.ServeHTTP(services, httptest.NewRequest(http.MethodGet, "/api/v1/services", nil))
	if services.Code != http.StatusOK || !bytes.Contains(services.Body.Bytes(), []byte("深度保洁")) {
		t.Fatalf("services response = %d, body=%s", services.Code, services.Body.String())
	}

	body := bytes.NewBufferString(`{"serviceId":"svc-cleaning","slotId":"slot-cleaning-0900","customerId":"CUS-001","customerName":"杭州星河家庭","startsAt":"2026-07-20T09:00:00+08:00","endsAt":"2026-07-20T11:00:00+08:00"}`)
	create := httptest.NewRequest(http.MethodPost, "/api/v1/bookings", body)
	create.Header.Set("Content-Type", "application/json")
	create.Header.Set("Idempotency-Key", "router-booking-1")
	created := httptest.NewRecorder()
	r.ServeHTTP(created, create)
	if created.Code != http.StatusCreated {
		t.Fatalf("booking create = %d, body=%s", created.Code, created.Body.String())
	}
	var envelope struct {
		Data Booking `json:"data"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Data.ID == "" || envelope.Data.Status != BookingPending {
		t.Fatalf("created booking = %+v", envelope.Data)
	}

	events := httptest.NewRecorder()
	r.ServeHTTP(events, httptest.NewRequest(http.MethodGet, "/api/v1/bookings/"+envelope.Data.ID+"/events", nil))
	if events.Code != http.StatusOK || !bytes.Contains(events.Body.Bytes(), []byte("created")) {
		t.Fatalf("booking events = %d, body=%s", events.Code, events.Body.String())
	}
}
