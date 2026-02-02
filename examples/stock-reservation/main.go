package main

import (
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/log"
	"github.com/kode4food/argyll/sdks/go-builder"
)

type StockReservation struct {
	ReservationID string `json:"reservation_id"`
	OrderID       string `json:"order_id"`
	ProductID     string `json:"product_id"`
	Quantity      int    `json:"quantity"`
	Status        string `json:"status"`
	ReservedAt    string `json:"reserved_at"`
	ExpiresAt     string `json:"expires_at"`
}

// Shared inventory tracker (simulates database)
var (
	stockLevels = map[string]int{
		"prod-laptop":     50,
		"prod-mouse":      200,
		"prod-keyboard":   75,
		"prod-monitor":    30,
		"prod-headphones": 0,
	}
	reservations = map[string][]StockReservation{}
	stockMutex   sync.Mutex
)

const version = "dev"

func main() {
	engineURL := os.Getenv("ARGYLL_ENGINE_URL")
	if engineURL == "" {
		engineURL = "http://localhost:8080"
	}

	logger := log.New("stock-reservation-example", os.Getenv("ENV"), version)
	slog.SetDefault(logger)

	client := builder.NewClient(engineURL, 30*time.Second)

	err := client.NewStep().WithName("Stock Reservation").
		WithLabels(api.Labels{
			"description": "reserve inventory for an order",
			"domain":      "inventory",
			"capability":  "reserve",
			"example":     "true",
		}).
		Required("order", api.TypeObject).
		Output("reservation", api.TypeObject).
		Start(handle)

	if err != nil {
		slog.Error("Failed to setup stock reservation",
			log.Error(err))
		os.Exit(1)
	}
}

func handle(_ *builder.StepContext, args api.Args) (api.StepResult, error) {
	order, ok := args["order"].(map[string]any)
	if !ok {
		return *api.NewResult().WithError(
			fmt.Errorf("order must be an object"),
		), nil
	}

	orderID, _ := order["id"].(string)
	productID, _ := order["product_id"].(string)
	quantityFloat, _ := order["quantity"].(float64)
	quantity := int(quantityFloat)

	slog.Info("Attempting stock reservation",
		slog.String("order_id", orderID),
		slog.String("product_id", productID),
		slog.Int("quantity", quantity))

	time.Sleep(time.Duration(5+rand.Intn(5)) * time.Second)

	// Thread-safe stock reservation
	stockMutex.Lock()
	defer stockMutex.Unlock()

	currentStock, ok := stockLevels[productID]
	if !ok {
		slog.Warn("Product not found in stock system",
			slog.String("product_id", productID))
		return *api.NewResult().WithError(
			fmt.Errorf("product %s not found in stock system",
				productID),
		), nil
	}

	if currentStock < quantity {
		slog.Warn("Insufficient stock for reservation",
			slog.String("product_id", productID),
			slog.Int("requested", quantity),
			slog.Int("available", currentStock))
		return *api.NewResult().WithError(
			fmt.Errorf("insufficient stock: requested %d, available %d",
				quantity, currentStock),
		), nil
	}

	// Reserve the stock
	stockLevels[productID] = currentStock - quantity

	reservation := StockReservation{
		ReservationID: fmt.Sprintf("RES-%d", time.Now().UnixNano()),
		OrderID:       orderID,
		ProductID:     productID,
		Quantity:      quantity,
		Status:        "reserved",
		ReservedAt:    time.Now().Format(time.RFC3339),
		ExpiresAt: time.Now().Add(30 * time.Minute).
			Format(time.RFC3339),
	}

	// Track reservation
	reservations[productID] = append(
		reservations[productID], reservation,
	)

	slog.Info("Stock reserved successfully",
		slog.String("reservation_id", reservation.ReservationID),
		slog.String("product_id", productID),
		slog.Int("quantity", quantity),
		slog.Int("remaining_stock", stockLevels[productID]))

	return *api.NewResult().WithOutput("reservation", reservation), nil
}
