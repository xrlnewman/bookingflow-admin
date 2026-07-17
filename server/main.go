package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	ctx := context.Background()
	store, closeStore, err := NewStoreFromEnv(ctx)
	if err != nil {
		panic(err)
	}
	defer closeStore()
	idem, closeRedis, err := NewRedisFromEnv(ctx)
	if err != nil {
		panic(err)
	}
	defer closeRedis()

	r := NewRouter(store, idem)
	addr := strings.TrimSpace(os.Getenv("HTTP_ADDR"))
	if addr == "" {
		addr = ":8080"
	}
	if err := r.Run(addr); err != nil {
		panic(err)
	}
}

func NewRouter(store CareStore, idem idempotencyStore) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), traceMiddleware())
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Idempotency-Key")
		c.Header("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		if c.Request.Method == http.MethodOptions {
			c.Status(http.StatusNoContent)
			c.Abort()
			return
		}
		c.Next()
	})
	svc := NewCareService(store, idem)
	bookingStore, ok := store.(BookingStore)
	if !ok {
		bookingStore = NewMemoryStore()
	}
	bookingSvc := NewBookingService(bookingStore, idem)

	r.GET("/healthz", func(c *gin.Context) {
		respond(c, http.StatusOK, gin.H{"service": "bookingflow", "status": "ok", "time": time.Now().UTC()})
	})
	api := r.Group("/api/v1")
	api.GET("/dashboard", func(c *gin.Context) {
		data, err := store.Dashboard(c.Request.Context())
		if err != nil {
			fail(c, err)
			return
		}
		respond(c, http.StatusOK, data)
	})
	api.GET("/departments", func(c *gin.Context) {
		data, err := store.ListDepartments(c.Request.Context())
		if err != nil {
			fail(c, err)
			return
		}
		respond(c, http.StatusOK, gin.H{"list": data, "total": len(data)})
	})
	api.GET("/doctors", func(c *gin.Context) {
		data, err := store.ListDoctors(c.Request.Context())
		if err != nil {
			fail(c, err)
			return
		}
		respond(c, http.StatusOK, gin.H{"list": data, "total": len(data)})
	})
	api.GET("/patients", func(c *gin.Context) {
		page, pageSize := pageParams(c)
		list, total, err := store.ListPatients(c.Request.Context(), page, pageSize)
		if err != nil {
			fail(c, err)
			return
		}
		respond(c, http.StatusOK, pageData(list, total, page, pageSize))
	})
	api.GET("/appointments", func(c *gin.Context) {
		page, pageSize := pageParams(c)
		list, total, err := store.ListAppointments(c.Request.Context(), page, pageSize, c.Query("status"))
		if err != nil {
			fail(c, err)
			return
		}
		respond(c, http.StatusOK, pageData(list, total, page, pageSize))
	})
	api.GET("/appointments/:id", func(c *gin.Context) {
		a, err := store.GetAppointment(c.Request.Context(), c.Param("id"))
		if err != nil {
			fail(c, err)
			return
		}
		respond(c, http.StatusOK, a)
	})
	api.GET("/appointments/:id/events", func(c *gin.Context) {
		events, err := store.ListAppointmentEvents(c.Request.Context(), c.Param("id"))
		if err != nil {
			fail(c, err)
			return
		}
		respond(c, http.StatusOK, gin.H{"list": events, "total": len(events)})
	})
	api.POST("/appointments", func(c *gin.Context) {
		var input CreateAppointmentInput
		if err := c.ShouldBindJSON(&input); err != nil {
			fail(c, errors.Join(ErrInvalidInput, err))
			return
		}
		a, err := svc.CreateAppointment(c.Request.Context(), input, c.GetHeader("Idempotency-Key"))
		if err != nil {
			fail(c, err)
			return
		}
		respond(c, http.StatusCreated, a)
	})
	api.POST("/appointments/:id/checkin", func(c *gin.Context) {
		a, err := svc.CheckinAppointment(c.Request.Context(), c.Param("id"), c.GetHeader("Idempotency-Key"))
		if err != nil {
			fail(c, err)
			return
		}
		respond(c, http.StatusOK, a)
	})
	api.POST("/appointments/:id/status", func(c *gin.Context) {
		var input UpdateAppointmentStatusInput
		if err := c.ShouldBindJSON(&input); err != nil {
			fail(c, errors.Join(ErrInvalidInput, err))
			return
		}
		a, err := svc.UpdateAppointmentStatus(c.Request.Context(), c.Param("id"), input.Status, input.Actor, c.GetHeader("Idempotency-Key"))
		if err != nil {
			fail(c, err)
			return
		}
		respond(c, http.StatusOK, a)
	})
	api.GET("/followups", func(c *gin.Context) {
		page, pageSize := pageParams(c)
		list, total, err := store.ListFollowups(c.Request.Context(), page, pageSize, c.Query("status"))
		if err != nil {
			fail(c, err)
			return
		}
		respond(c, http.StatusOK, pageData(list, total, page, pageSize))
	})
	api.GET("/services", func(c *gin.Context) {
		services, err := bookingStore.ListServices(c.Request.Context(), c.Query("category"), c.DefaultQuery("active", "true") != "false")
		if err != nil {
			fail(c, err)
			return
		}
		respond(c, http.StatusOK, gin.H{"list": services, "total": len(services)})
	})
	api.GET("/availability", func(c *gin.Context) {
		slots, err := bookingStore.ListAvailability(c.Request.Context(), c.Query("serviceId"), c.Query("date"))
		if err != nil {
			fail(c, err)
			return
		}
		respond(c, http.StatusOK, gin.H{"list": slots, "total": len(slots)})
	})
	api.GET("/bookings", func(c *gin.Context) {
		page, pageSize := pageParams(c)
		list, total, err := bookingStore.ListBookings(c.Request.Context(), page, pageSize, c.Query("status"), c.Query("customerId"))
		if err != nil {
			fail(c, err)
			return
		}
		respond(c, http.StatusOK, pageData(list, total, page, pageSize))
	})
	api.GET("/bookings/:id/events", func(c *gin.Context) {
		events, err := bookingStore.ListBookingEvents(c.Request.Context(), c.Param("id"))
		if err != nil {
			fail(c, err)
			return
		}
		respond(c, http.StatusOK, gin.H{"list": events, "total": len(events)})
	})
	api.GET("/bookings/:id", func(c *gin.Context) {
		booking, err := bookingStore.GetBooking(c.Request.Context(), c.Param("id"))
		if err != nil {
			fail(c, err)
			return
		}
		respond(c, http.StatusOK, booking)
	})
	api.POST("/bookings", func(c *gin.Context) {
		var input CreateBookingInput
		if err := c.ShouldBindJSON(&input); err != nil {
			fail(c, errors.Join(ErrInvalidInput, err))
			return
		}
		booking, err := bookingSvc.CreateBooking(c.Request.Context(), input, c.GetHeader("Idempotency-Key"))
		if err != nil {
			fail(c, err)
			return
		}
		respond(c, http.StatusCreated, booking)
	})
	api.POST("/bookings/:id/status", func(c *gin.Context) {
		var input UpdateBookingInput
		if err := c.ShouldBindJSON(&input); err != nil {
			fail(c, errors.Join(ErrInvalidInput, err))
			return
		}
		booking, err := bookingSvc.UpdateBookingStatus(c.Request.Context(), c.Param("id"), input.Status, input.Actor, c.GetHeader("Idempotency-Key"))
		if err != nil {
			fail(c, err)
			return
		}
		respond(c, http.StatusOK, booking)
	})
	api.POST("/bookings/:id/reschedule", func(c *gin.Context) {
		var input RescheduleBookingInput
		if err := c.ShouldBindJSON(&input); err != nil {
			fail(c, errors.Join(ErrInvalidInput, err))
			return
		}
		booking, err := bookingSvc.RescheduleBooking(c.Request.Context(), c.Param("id"), input, c.GetHeader("Idempotency-Key"))
		if err != nil {
			fail(c, err)
			return
		}
		respond(c, http.StatusOK, booking)
	})
	api.POST("/bookings/:id/refund", func(c *gin.Context) {
		booking, err := bookingSvc.RefundBooking(c.Request.Context(), c.Param("id"), c.GetHeader("Idempotency-Key"))
		if err != nil {
			fail(c, err)
			return
		}
		respond(c, http.StatusOK, booking)
	})
	api.POST("/bookings/:id/review", func(c *gin.Context) {
		var input CreateReviewInput
		if err := c.ShouldBindJSON(&input); err != nil {
			fail(c, errors.Join(ErrInvalidInput, err))
			return
		}
		review, err := bookingSvc.CreateReview(c.Request.Context(), c.Param("id"), input, c.GetHeader("Idempotency-Key"))
		if err != nil {
			fail(c, err)
			return
		}
		respond(c, http.StatusCreated, review)
	})
	api.POST("/followups", func(c *gin.Context) {
		var input CreateFollowupInput
		if err := c.ShouldBindJSON(&input); err != nil {
			fail(c, errors.Join(ErrInvalidInput, err))
			return
		}
		f, err := svc.CreateFollowup(c.Request.Context(), input, c.GetHeader("Idempotency-Key"))
		if err != nil {
			fail(c, err)
			return
		}
		respond(c, http.StatusCreated, f)
	})
	api.POST("/followups/:id/complete", func(c *gin.Context) {
		f, err := svc.CompleteFollowup(c.Request.Context(), c.Param("id"), c.GetHeader("Idempotency-Key"))
		if err != nil {
			fail(c, err)
			return
		}
		respond(c, http.StatusOK, f)
	})
	return r
}

func traceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		trace := c.GetHeader("X-Trace-ID")
		if trace == "" {
			trace = "cf-" + time.Now().UTC().Format("20060102150405.000000000")
		}
		c.Set("traceId", trace)
		c.Header("X-Trace-ID", trace)
		c.Next()
	}
}
func respond(c *gin.Context, status int, data any) {
	trace, _ := c.Get("traceId")
	c.JSON(status, gin.H{"code": 0, "message": "ok", "data": data, "traceId": trace})
}
func fail(c *gin.Context, err error) {
	status := httpStatusForError(err)
	trace, _ := c.Get("traceId")
	c.JSON(status, gin.H{"code": status, "message": err.Error(), "data": nil, "traceId": trace})
}
func pageParams(c *gin.Context) (int, int) {
	return parseInt(c.DefaultQuery("page", "1"), 1), parseInt(c.DefaultQuery("pageSize", "20"), 20)
}
func pageData[T any](list []T, total, page, pageSize int) gin.H {
	return gin.H{"list": list, "total": total, "page": page, "pageSize": pageSize}
}
