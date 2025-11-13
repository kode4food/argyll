package main

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/kode4food/spuds/engine/pkg/api"
	"github.com/kode4food/spuds/engine/pkg/builder"
)

type ProductInfo struct {
	ProductID       string  `json:"product_id"`
	Name            string  `json:"name"`
	Price           float64 `json:"price"`
	AvailableStock  int     `json:"available_stock"`
	ReservedStock   int     `json:"reserved_stock"`
	MinimumOrderQty int     `json:"minimum_order_qty"`
	MaximumOrderQty int     `json:"maximum_order_qty"`
	Category        string  `json:"category"`
	ShippingWeight  float64 `json:"shipping_weight_kg"`
}

// Simulated inventory database with mutex for thread-safe access
var (
	inventoryDB = map[string]ProductInfo{
		"prod-laptop": {
			ProductID:       "prod-laptop",
			Name:            "Professional Laptop",
			Price:           1299.99,
			AvailableStock:  50,
			ReservedStock:   5,
			MinimumOrderQty: 1,
			MaximumOrderQty: 10,
			Category:        "electronics",
			ShippingWeight:  2.5,
		},
		"prod-mouse": {
			ProductID:       "prod-mouse",
			Name:            "Wireless Mouse",
			Price:           29.99,
			AvailableStock:  200,
			ReservedStock:   10,
			MinimumOrderQty: 1,
			MaximumOrderQty: 50,
			Category:        "accessories",
			ShippingWeight:  0.2,
		},
		"prod-keyboard": {
			ProductID:       "prod-keyboard",
			Name:            "Mechanical Keyboard",
			Price:           149.99,
			AvailableStock:  75,
			ReservedStock:   8,
			MinimumOrderQty: 1,
			MaximumOrderQty: 25,
			Category:        "accessories",
			ShippingWeight:  1.2,
		},
		"prod-monitor": {
			ProductID:       "prod-monitor",
			Name:            "4K Monitor 27\"",
			Price:           449.99,
			AvailableStock:  30,
			ReservedStock:   3,
			MinimumOrderQty: 1,
			MaximumOrderQty: 5,
			Category:        "electronics",
			ShippingWeight:  6.5,
		},
		"prod-headphones": {
			ProductID:       "prod-headphones",
			Name:            "Noise-Canceling Headphones",
			Price:           249.99,
			AvailableStock:  0, // Out of stock
			ReservedStock:   0,
			MinimumOrderQty: 1,
			MaximumOrderQty: 10,
			Category:        "accessories",
			ShippingWeight:  0.4,
		},
	}
	inventoryMutex sync.RWMutex
)

func main() {
	engineURL := os.Getenv("SPUDS_ENGINE_URL")
	if engineURL == "" {
		engineURL = "http://localhost:8080"
	}

	client := builder.NewClient(engineURL, 30*time.Second)

	err := client.NewStep("Inventory Resolver").
		Optional("product_id", api.TypeString, `"prod-laptop"`).
		Output("product_info", api.TypeObject).
		Start(handle)

	if err != nil {
		slog.Error("Failed to setup inventory resolver",
			slog.Any("error", err))
		os.Exit(1)
	}
}

func handle(ctx context.Context, args api.Args) (api.StepResult, error) {
	time.Sleep(time.Duration(5+rand.Intn(5)) * time.Second)

	productID, hasProductID := args["product_id"].(string)
	if !hasProductID || productID == "" {
		// Default to laptop if no product specified
		productID = "prod-laptop"
		slog.Info("No product_id provided, using default",
			slog.String("product_id", productID))
	}

	inventoryMutex.RLock()
	productInfo, exists := inventoryDB[productID]
	inventoryMutex.RUnlock()

	if !exists {
		slog.Warn("Product not found in inventory",
			slog.String("product_id", productID))
		return *api.NewResult().WithError(
			fmt.Errorf("product not found: %s", productID),
		), nil
	}

	slog.Info("Product inventory resolved",
		slog.String("product_id", productInfo.ProductID),
		slog.String("name", productInfo.Name),
		slog.Int("available_stock", productInfo.AvailableStock),
		slog.Float64("price", productInfo.Price))

	return *api.NewResult().WithOutput("product_info", productInfo), nil
}
