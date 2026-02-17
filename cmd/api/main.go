package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	"github.com/srgjo27/scalable_ticket/internal/adapter/handler"
	"github.com/srgjo27/scalable_ticket/internal/adapter/repository/postgres"
	"github.com/srgjo27/scalable_ticket/internal/core/services"
)

func main() {
	connStr := "postgres://avows@localhost:5432/scalable_ticket?sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to open db connection: %v", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping db: %v", err)
	}

	defer db.Close()

	log.Println("Database connected successfully with tuned pool settings.")

	seatRepo := postgres.NewSeatRepository(db)
	bookingRepo := postgres.NewBookingRepository(db)

	bookingService := services.NewBookingService(seatRepo, bookingRepo)

	bookingHandler := handler.NewBookingHandler(bookingService)

	go func() {
		bookingService.RunBackgroundCleanup(context.Background())
	}()

	mux := http.NewServeMux()

	mux.HandleFunc("/bookings", bookingHandler.CreateBooking)

	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Println("Server starting on port :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server startup failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}
