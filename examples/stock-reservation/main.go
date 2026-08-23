package main

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/kode4food/argyll/engine/pkg/log"
	argyll "github.com/kode4food/argyll/sdk/go"
	"github.com/kode4food/argyll/sdk/go/gen"
)

type (
	// Order identifies the product and quantity being reserved
	Order struct {
		ID        string `json:"id"`
		ProductID string `json:"product_id"`
		Quantity  int    `json:"quantity"`
	}

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

var inv = newInventory()

//go:generate go run github.com/kode4food/argyll/sdk/go/gen/cmd/argyll-gen .

func main() {
	logger := log.New("stock-reservation-example", os.Getenv("ENV"), version)
	slog.SetDefault(logger)

	if err := gen.Serve(context.Background(), ArgyllSteps()...); err != nil {
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

//argyll:wrap stock-reservation;name:Stock Reservation
//argyll:compensate releaseStock
//argyll:labels description: reserve inventory for an order
//argyll:labels domain: inventory; capability: reserve; example: true
func reserveStock(order Order) (reservation StockReservation, err error) {
	orderID := order.ID
	productID := order.ProductID
	quantity := order.Quantity

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
		return StockReservation{}, argyll.NewHTTPError(
			http.StatusNotFound,
			fmt.Sprintf("product %s not found in stock system", productID),
		)
	}

	if currentStock < quantity {
		slog.Warn("Insufficient stock for reservation",
			slog.String("product_id", productID),
			slog.Int("requested", quantity),
			slog.Int("available", currentStock))
		return StockReservation{}, argyll.NewHTTPError(
			http.StatusConflict,
			fmt.Sprintf("insufficient stock: requested %d, available %d",
				quantity, currentStock),
		)
	}

	// Reserve the stock
	inv.stockLevels[productID] = currentStock - quantity

	reservation = StockReservation{
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

	return reservation, nil
}

func releaseStock(reservation StockReservation) error {
	reservationID := reservation.ReservationID
	productID := reservation.ProductID

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
