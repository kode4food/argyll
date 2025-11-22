package main

import (
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"time"

	"github.com/kode4food/spuds/engine/pkg/api"
	"github.com/kode4food/spuds/engine/pkg/builder"
)

type PaymentResult struct {
	OrderID     string  `json:"order_id"`
	Amount      float64 `json:"amount"`
	Status      string  `json:"status"`
	PaymentID   string  `json:"payment_id"`
	ProcessedAt string  `json:"processed_at"`
}

func main() {
	engineURL := os.Getenv("SPUDS_ENGINE_URL")
	if engineURL == "" {
		engineURL = "http://localhost:8080"
	}

	client := builder.NewClient(engineURL, 30*time.Second)

	err := client.NewStep("Payment Processor").
		Required("order", api.TypeObject).
		Output("payment_result", api.TypeObject).
		WithAsyncExecution().
		WithTimeout(5 * api.Minute).
		Start(handle)

	if err != nil {
		slog.Error("Failed to setup payment processor",
			slog.Any("error", err))
		os.Exit(1)
	}
}

func handle(ctx *builder.StepContext, args api.Args) (api.StepResult, error) {
	order, ok := args["order"].(map[string]any)
	if !ok {
		return *api.NewResult().WithError(
			fmt.Errorf("order must be an object"),
		), nil
	}
	orderID, _ := order["id"].(string)
	grandTotal, _ := order["grand_total"].(float64)
	paymentMethod, _ := order["payment_method"].(string)

	slog.Info("Processing payment (async)",
		slog.Float64("amount", grandTotal),
		slog.String("order_id", orderID),
		slog.String("payment_method", paymentMethod))

	async, err := builder.NewAsyncContext(ctx)
	if err != nil {
		return *api.NewResult().WithError(err), nil
	}

	go func() {
		slog.Info("Starting async payment processing",
			slog.String("order_id", orderID),
			slog.String("flow_id", async.FlowID()),
			slog.String("step_id", async.StepID()))

		processingTime := time.Duration(5+rand.Intn(10)) * time.Second
		time.Sleep(processingTime)

		success := rand.Float64() < 0.5

		if success {
			paymentResult := PaymentResult{
				OrderID:     orderID,
				Amount:      grandTotal,
				Status:      "completed",
				PaymentID:   fmt.Sprintf("PAY-%d", time.Now().Unix()),
				ProcessedAt: time.Now().Format(time.RFC3339),
			}

			slog.Info("Payment completed successfully",
				slog.String("order_id", orderID),
				slog.String("payment_id", paymentResult.PaymentID),
				slog.Float64("amount", grandTotal))

			if err := async.Success(api.Args{
				"payment_result": paymentResult,
			}); err != nil {
				slog.Error("Failed to send webhook completion",
					slog.Any("error", err))
			}

		} else {
			failureReasons := []string{
				"insufficient funds",
				"card declined",
				"expired payment method",
				"fraud detection triggered",
				"payment gateway timeout",
			}
			reason := failureReasons[rand.Intn(len(failureReasons))]

			slog.Warn("Payment failed",
				slog.String("order_id", orderID),
				slog.String("reason", reason))

			if err := async.Fail(
				fmt.Errorf("payment failed: %s", reason),
			); err != nil {
				slog.Error("Failed to send webhook failure",
					slog.Any("error", err))
			}
		}
	}()

	return api.StepResult{Success: true}, nil
}
