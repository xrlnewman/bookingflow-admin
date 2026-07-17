package main

import "errors"

// ServiceOffering is a bookable service shown in the customer catalog.
type ServiceOffering struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Category        string `json:"category"`
	Description     string `json:"description"`
	DurationMinutes int    `json:"durationMinutes"`
	PriceCents      int    `json:"priceCents"`
	Active          bool   `json:"active"`
	CreatedAt       string `json:"createdAt"`
	UpdatedAt       string `json:"updatedAt"`
}

// AvailabilitySlot is a capacity-controlled service time window.
type AvailabilitySlot struct {
	ID        string `json:"id"`
	ServiceID string `json:"serviceId"`
	StartsAt  string `json:"startsAt"`
	EndsAt    string `json:"endsAt"`
	Capacity  int    `json:"capacity"`
	Remaining int    `json:"remaining"`
	Status    string `json:"status"`
}

// Booking is a customer order in the service lifecycle.
type Booking struct {
	ID            string `json:"id"`
	ServiceID     string `json:"serviceId"`
	ServiceName   string `json:"serviceName"`
	SlotID        string `json:"slotId"`
	CustomerID    string `json:"customerId"`
	CustomerName  string `json:"customerName"`
	CustomerPhone string `json:"customerPhone,omitempty"`
	StartsAt      string `json:"startsAt"`
	EndsAt        string `json:"endsAt"`
	Status        string `json:"status"`
	PaymentStatus string `json:"paymentStatus"`
	AmountCents   int    `json:"amountCents"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
}

// BookingEvent records every change in the booking timeline.
type BookingEvent struct {
	ID         string `json:"id"`
	BookingID  string `json:"bookingId"`
	EventType  string `json:"eventType"`
	FromStatus string `json:"fromStatus,omitempty"`
	ToStatus   string `json:"toStatus,omitempty"`
	Actor      string `json:"actor"`
	Note       string `json:"note,omitempty"`
	CreatedAt  string `json:"createdAt"`
}

// BookingReview is a customer review associated with a completed booking.
type BookingReview struct {
	ID           string `json:"id"`
	BookingID    string `json:"bookingId"`
	CustomerName string `json:"customerName"`
	Rating       int    `json:"rating"`
	Content      string `json:"content"`
	CreatedAt    string `json:"createdAt"`
}

// CreateBookingInput is accepted by POST /bookings.
type CreateBookingInput struct {
	ServiceID     string `json:"serviceId" binding:"required"`
	SlotID        string `json:"slotId" binding:"required"`
	CustomerID    string `json:"customerId"`
	CustomerName  string `json:"customerName" binding:"required"`
	CustomerPhone string `json:"customerPhone"`
	StartsAt      string `json:"startsAt" binding:"required"`
	EndsAt        string `json:"endsAt" binding:"required"`
}

// UpdateBookingInput is accepted by POST /bookings/:id/status.
type UpdateBookingInput struct {
	Status string `json:"status" binding:"required"`
	Actor  string `json:"actor"`
}

// RescheduleBookingInput is accepted by POST /bookings/:id/reschedule.
type RescheduleBookingInput struct {
	SlotID   string `json:"slotId" binding:"required"`
	StartsAt string `json:"startsAt" binding:"required"`
	EndsAt   string `json:"endsAt" binding:"required"`
}

// CreateReviewInput is accepted by POST /bookings/:id/review.
type CreateReviewInput struct {
	Rating  int    `json:"rating" binding:"required"`
	Content string `json:"content" binding:"required"`
}

const (
	BookingPending   = "待确认"
	BookingReserved  = "已预约"
	BookingCheckedIn = "已签到"
	BookingServing   = "服务中"
	BookingCompleted = "已完成"
	BookingCancelled = "已取消"

	PaymentPending  = "待支付"
	PaymentPaid     = "已支付"
	PaymentRefunded = "已退款"

	AvailabilityOpen   = "可预约"
	AvailabilityFull   = "已满"
	AvailabilityClosed = "已结束"
)

var (
	ErrInvalidBookingTransition = errors.New("invalid booking status transition")
	ErrSlotUnavailable          = errors.New("booking slot is unavailable")
	ErrReviewExists             = errors.New("booking review already exists")
	ErrBookingAlreadyRefunded   = errors.New("booking has already been refunded")
)

var bookingTransitions = map[string]map[string]bool{
	BookingPending:   {BookingReserved: true, BookingCancelled: true},
	BookingReserved:  {BookingCheckedIn: true, BookingCancelled: true},
	BookingCheckedIn: {BookingServing: true, BookingCancelled: true},
	BookingServing:   {BookingCompleted: true},
	BookingCompleted: {},
	BookingCancelled: {},
}

func validBookingStatus(status string) bool {
	_, ok := bookingTransitions[status]
	return ok
}
