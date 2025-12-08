package main

import (
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"time"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/builder"
	"github.com/kode4food/argyll/engine/pkg/log"
)

type PaymentResult struct {
	OrderID     string  `json:"order_id"`
	Amount      float64 `json:"amount"`
	Status      string  `json:"status"`
	PaymentID   string  `json:"payment_id"`
	ProcessedAt string  `json:"processed_at"`
}

const version = "dev"

func main() {
	engineURL := os.Getenv("ARGYLL_ENGINE_URL")
	if engineURL == "" {
		engineURL = "http://localhost:8080"
	}

	logger := log.New("payment-processor-example", os.Getenv("ENV"), version)
	slog.SetDefault(logger)

	client := builder.NewClient(engineURL, 30*time.Second)

	err := client.NewStep("Payment Processor").
		Required("order", api.TypeObject).
		Output("payment_result", api.TypeObject).
		WithAsyncExecution().
		WithTimeout(5 * api.Minute).
		Start(handle)

	if err != nil {
		slog.Error("Failed to setup payment processor",
			log.Error(err))
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
			log.FlowID(async.FlowID()),
			log.StepID(async.StepID()))

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
					log.Error(err))
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
					log.Error(err))
			}
		}
	}()

	return api.StepResult{Success: true}, nil
}
