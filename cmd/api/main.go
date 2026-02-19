package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"

	"github.com/srgjo27/scalable_ticket/internal/adapter/handler"
	"github.com/srgjo27/scalable_ticket/internal/adapter/repository/postgres"
	"github.com/srgjo27/scalable_ticket/internal/core/services"
	"github.com/srgjo27/scalable_ticket/internal/platform/database"
)

func loadEnv(filepath string) {
	file, err := os.Open(filepath)

	if err != nil {
		log.Println("File .env tidak ditemukan, menggunakan variabel OS bawaan.")
		return
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			os.Setenv(key, value)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Gagal membaca file .env: %v\n", err)
	}
}

func main() {
	loadEnv(".env")

	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}

	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}

	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "postgres"
	}

	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		dbPassword = ""
	}

	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "scalable_ticket"
	}

	dbConfig := database.Config{
		Host:     dbHost,
		Port:     dbPort,
		User:     dbUser,
		Password: dbPassword,
		DBName:   dbName,
	}

	db, err := database.NewPostgresDB(dbConfig)

	if err != nil {
		log.Fatalf("Failed to connect to db after retries: %v", err)
	}

	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "localhost"
	}

	redisPort := os.Getenv("REDIS_PORT")
	if redisPort == "" {
		redisPort = "6379"
	}

	log.Printf("Connecting to Redis at %s:%s...", redisHost, redisPort)

	redisClient := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", redisHost, redisPort),
		DB:   0,
	})

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("Redis connected successfully!")

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)
	defer db.Close()

	seatRepo := postgres.NewSeatRepository(db)
	bookingRepo := postgres.NewBookingRepository(db)

	bookingService := services.NewBookingService(seatRepo, bookingRepo, redisClient)

	bookingHandler := handler.NewBookingHandler(bookingService)

	go func() {
		bookingService.RunBackgroundCleanup(context.Background())
	}()

	mux := http.NewServeMux()

	mux.HandleFunc("/bookings", bookingHandler.CreateBooking)

	mux.HandleFunc("/seats", bookingHandler.GetSeats)

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
