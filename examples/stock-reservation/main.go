package main

import (
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/log"
	argyll "github.com/kode4food/argyll/sdk/go"
)

type (
	// StockReservation records a quantity of a product held for an order
	StockReservation struct {
		ReservationID string `json:"reservation_id"`
		OrderID       string `json:"order_id"`
		ProductID     string `json:"product_id"`
		Quantity      int    `json:"quantity"`
		Status        string `json:"status"`
		ReservedAt    string `json:"reserved_at"`
		ExpiresAt     string `json:"expires_at"`
	}

	// inventory simulates a database of stock levels and active reservations
	inventory struct {
		mu           sync.Mutex
		stockLevels  map[string]int
		reservations map[string][]StockReservation
	}
)

const version = "dev"

func main() {
	engineURL := os.Getenv("ARGYLL_ENGINE_URL")
	if engineURL == "" {
		engineURL = "http://localhost:8080"
	}

	logger := log.New("stock-reservation-example", os.Getenv("ENV"), version)
	slog.SetDefault(logger)

	client := argyll.NewClient(engineURL, 30*time.Second)
	inv := newInventory()

	err := client.NewStep().WithName("Stock Reservation").
		WithLabels(api.Labels{
			"description": "reserve inventory for an order",
			"domain":      "inventory",
			"capability":  "reserve",
			"example":     "true",
		}).
		Required("order", api.TypeObject).
		Output("reservation", api.TypeObject).
		WithCompensateHandler(inv.compensate).
		Start(inv.handle)

	if err != nil {
		slog.Error("Failed to setup stock reservation", log.Error(err))
		os.Exit(1)
	}
}

func newInventory() *inventory {
	return &inventory{
		stockLevels: map[string]int{
			"prod-laptop":     50,
			"prod-mouse":      200,
			"prod-keyboard":   75,
			"prod-monitor":    30,
			"prod-headphones": 0,
		},
		reservations: map[string][]StockReservation{},
	}
}

func (inv *inventory) handle(
	_ *argyll.StepContext, args api.Args,
) (api.Args, error) {
	order, ok := args["order"].(map[string]any)
	if !ok {
		return nil, argyll.NewHTTPError(
			http.StatusBadRequest, "order must be an object",
		)
	}

	orderID, _ := order["id"].(string)
	productID, _ := order["product_id"].(string)
	quantityFloat, _ := order["quantity"].(float64)
	quantity := int(quantityFloat)

	slog.Info("Attempting stock reservation",
		slog.String("order_id", orderID),
		slog.String("product_id", productID),
		slog.Int("quantity", quantity))

	simulateLatency()

	// Thread-safe stock reservation
	inv.mu.Lock()
	defer inv.mu.Unlock()

	currentStock, ok := inv.stockLevels[productID]
	if !ok {
		slog.Warn("Product not found in stock system",
			slog.String("product_id", productID))
		return nil, argyll.NewHTTPError(
			http.StatusNotFound,
			fmt.Sprintf("product %s not found in stock system", productID),
		)
	}

	if currentStock < quantity {
		slog.Warn("Insufficient stock for reservation",
			slog.String("product_id", productID),
			slog.Int("requested", quantity),
			slog.Int("available", currentStock))
		return nil, argyll.NewHTTPError(
			http.StatusConflict,
			fmt.Sprintf("insufficient stock: requested %d, available %d",
				quantity, currentStock),
		)
	}

	// Reserve the stock
	inv.stockLevels[productID] = currentStock - quantity

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
	inv.reservations[productID] = append(
		inv.reservations[productID], reservation,
	)

	slog.Info("Stock reserved",
		slog.String("reservation_id", reservation.ReservationID),
		slog.String("product_id", productID),
		slog.Int("quantity", quantity),
		slog.Int("stock_before", currentStock),
		slog.Int("stock_after", inv.stockLevels[productID]))

	return api.Args{"reservation": reservation}, nil
}

func (inv *inventory) compensate(
	_ *argyll.StepContext, _ api.Args, outputs api.Args,
) error {
	reservation, ok := outputs["reservation"].(map[string]any)
	if !ok {
		return argyll.NewHTTPError(
			http.StatusBadRequest, "reservation must be an object",
		)
	}

	reservationID, _ := reservation["reservation_id"].(string)
	productID, _ := reservation["product_id"].(string)

	slog.Info("Attempting stock release",
		slog.String("reservation_id", reservationID),
		slog.String("product_id", productID))

	simulateLatency()

	inv.mu.Lock()
	defer inv.mu.Unlock()

	// Releasing an unknown reservation is a no-op, so a repeated
	// compensation cannot return the same stock twice
	held := inv.reservations[productID]
	idx := slices.IndexFunc(held, func(r StockReservation) bool {
		return r.ReservationID == reservationID
	})
	if idx < 0 {
		slog.Info("Reservation already released",
			slog.String("reservation_id", reservationID),
			slog.String("product_id", productID))
		return nil
	}

	quantity := held[idx].Quantity
	before := inv.stockLevels[productID]
	inv.stockLevels[productID] = before + quantity
	inv.reservations[productID] = slices.Delete(held, idx, idx+1)

	slog.Info("Stock released",
		slog.String("reservation_id", reservationID),
		slog.String("product_id", productID),
		slog.Int("quantity", quantity),
		slog.Int("stock_before", before),
		slog.Int("stock_after", inv.stockLevels[productID]))

	return nil
}

func simulateLatency() {
	time.Sleep(time.Duration(5+rand.Intn(5)) * time.Second)
}
