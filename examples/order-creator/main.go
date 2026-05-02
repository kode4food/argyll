package main

import (
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/log"
	"github.com/kode4food/argyll/sdks/go-builder"
)

type Order struct {
	ID             string  `json:"id"`
	UserID         string  `json:"user_id"`
	ProductID      string  `json:"product_id"`
	Quantity       int     `json:"quantity"`
	UnitPrice      float64 `json:"unit_price"`
	TotalAmount    float64 `json:"total_amount"`
	ShippingCost   float64 `json:"shipping_cost"`
	Tax            float64 `json:"tax"`
	GrandTotal     float64 `json:"grand_total"`
	Status         string  `json:"status"`
	PaymentMethod  string  `json:"payment_method"`
	ShippingMethod string  `json:"shipping_method"`
	CreatedAt      string  `json:"created_at"`
}

const version = "dev"

func main() {
	engineURL := os.Getenv("ARGYLL_ENGINE_URL")
	if engineURL == "" {
		engineURL = "http://localhost:8080"
	}

	logger := log.New("order-creator-example", os.Getenv("ENV"), version)
	slog.SetDefault(logger)

	client := builder.NewClient(engineURL, 30*time.Second)

	err := client.NewStep().WithName("Order Creator").
		WithLabels(api.Labels{
			"description": "create order records and validate business rules",
			"domain":      "orders",
			"capability":  "create",
			"example":     "true",
		}).
		Required("user_info", api.TypeObject).
		Required("product_info", api.TypeObject).
		Optional("quantity", api.TypeNumber, "1").
		WithPredicate(api.ScriptLangJPath,
			`$.product_info.name == "Professional Laptop"`).
		Output("order", api.TypeObject).
		Start(handle)

	if err != nil {
		slog.Error("Failed to setup order creator", log.Error(err))
		os.Exit(1)
	}
}

func handle(ctx *builder.StepContext, args api.Args) (api.Args, error) {
	// Extract and validate user info
	userInfo, ok := args["user_info"].(map[string]any)
	if !ok {
		return nil, builder.NewHTTPError(
			http.StatusBadRequest, "user_info must be an object",
		)
	}

	userID, _ := userInfo["id"].(string)
	userName, _ := userInfo["name"].(string)
	accountType, _ := userInfo["account_type"].(string)
	creditLimit, _ := userInfo["credit_limit"].(float64)

	// Extract and validate product info
	productInfo, ok := args["product_info"].(map[string]any)
	if !ok {
		return nil, builder.NewHTTPError(
			http.StatusBadRequest, "product_info must be an object",
		)
	}

	productID, _ := productInfo["product_id"].(string)
	productName, _ := productInfo["name"].(string)
	price, _ := productInfo["price"].(float64)
	availableStock, _ := productInfo["available_stock"].(float64)
	minOrderQty, _ := productInfo["minimum_order_qty"].(float64)
	maxOrderQty, _ := productInfo["maximum_order_qty"].(float64)
	shippingWeight, _ := productInfo["shipping_weight_kg"].(float64)

	// Get quantity (default to 1)
	quantity := 1
	if qtyArg, ok := args["quantity"].(float64); ok {
		quantity = int(qtyArg)
	}

	// Business logic validation
	if availableStock <= 0 {
		slog.Warn("Product out of stock",
			slog.String("product_id", productID),
			slog.String("product_name", productName))
		return nil, builder.NewHTTPError(
			http.StatusConflict,
			fmt.Sprintf("product %s is out of stock", productName),
		)
	}

	if quantity < int(minOrderQty) {
		return nil, builder.NewHTTPError(
			http.StatusBadRequest,
			fmt.Sprintf("quantity %d below minimum order quantity %d",
				quantity, int(minOrderQty)),
		)
	}

	if quantity > int(maxOrderQty) {
		return nil, builder.NewHTTPError(
			http.StatusBadRequest,
			fmt.Sprintf("quantity %d exceeds maximum order quantity %d",
				quantity, int(maxOrderQty)),
		)
	}

	if float64(quantity) > availableStock {
		return nil, builder.NewHTTPError(
			http.StatusConflict,
			fmt.Sprintf("insufficient stock: requested %d, available %d",
				quantity, int(availableStock)),
		)
	}

	// Calculate costs
	totalAmount := price * float64(quantity)
	shippingCost := calculateShipping(shippingWeight, float64(quantity))
	tax := totalAmount * 0.08 // 8% tax
	grandTotal := totalAmount + shippingCost + tax

	// Check credit limit for standard accounts
	if accountType == "standard" && grandTotal > creditLimit {
		slog.Warn("Order exceeds credit limit",
			slog.String("user_id", userID),
			slog.Float64("grand_total", grandTotal),
			slog.Float64("credit_limit", creditLimit))
		return nil, builder.NewHTTPError(
			http.StatusConflict,
			fmt.Sprintf("order total $%.2f exceeds credit limit $%.2f",
				grandTotal, creditLimit),
		)
	}

	time.Sleep(time.Duration(5+rand.Intn(5)) * time.Second)

	// Simulate random transient failures (50% chance)
	if rand.Float64() < 0.5 {
		errMsg := []string{
			"database connection timeout",
			"inventory service unavailable",
			"rate limit exceeded",
			"network timeout",
		}
		selectedErr := errMsg[rand.Intn(len(errMsg))]
		slog.Warn("Simulating transient failure (will retry)",
			log.Error(errors.New(selectedErr)),
			log.StepID(ctx.StepID))
		return nil, builder.NewHTTPError(
			http.StatusServiceUnavailable, selectedErr,
		)
	}

	orderID := fmt.Sprintf("ORDER-%d", time.Now().Unix())
	slog.Info("Order created successfully",
		slog.String("order_id", orderID),
		slog.String("user_name", userName),
		slog.String("product_name", productName),
		slog.Int("quantity", quantity),
		slog.Float64("grand_total", grandTotal))

	order := Order{
		ID:             orderID,
		UserID:         userID,
		ProductID:      productID,
		Quantity:       quantity,
		UnitPrice:      price,
		TotalAmount:    totalAmount,
		ShippingCost:   shippingCost,
		Tax:            tax,
		GrandTotal:     grandTotal,
		Status:         "pending_payment",
		PaymentMethod:  "credit",
		ShippingMethod: "standard",
		CreatedAt:      time.Now().Format(time.RFC3339),
	}

	return api.Args{"order": order}, nil
}

func calculateShipping(weightKg, quantity float64) float64 {
	totalWeight := weightKg * quantity
	baseRate := 5.00

	if totalWeight < 1 {
		return baseRate
	}
	if totalWeight < 5 {
		return baseRate + (totalWeight * 2.0)
	}
	return baseRate + (totalWeight * 1.5)
}
