package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
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
	if err := builder.SetupStep(
		"Payment Processor", build, handle,
	); err != nil {
		slog.Error("Failed to setup payment processor",
			slog.Any("error", err))
		os.Exit(1)
	}
}

func build(step *builder.Step) *builder.Step {
	return step.
		Required("order", api.TypeObject).
		Output("payment_result", api.TypeObject).
		WithAsyncExecution().
		WithTimeout(5 * api.Minute)
}

func handle(ctx context.Context, args api.Args) (api.StepResult, error) {
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

	meta, ok := ctx.Value(builder.MetadataKey).(api.Metadata)
	if !ok {
		return *api.NewResult().WithError(
			fmt.Errorf("metadata not found in context"),
		), nil
	}
	hookURL, ok := meta["webhook_url"].(string)
	if !ok || hookURL == "" {
		return *api.NewResult().WithError(
			fmt.Errorf("no webhook URL provided"),
		), nil
	}

	go func() {
		slog.Info("Starting async payment processing",
			slog.String("order_id", orderID))

		// Simulate payment gateway processing time (5-15 seconds)
		processingTime := time.Duration(5+rand.Intn(10)) * time.Second
		time.Sleep(processingTime)

		// Simulate payment success/failure (90% success rate)
		success := rand.Float64() < 0.9

		var result api.StepResult
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

			result = *api.NewResult().WithOutput(
				"payment_result", paymentResult,
			)

		} else {
			// Simulate payment failure scenarios
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

			result = *api.NewResult().WithError(
				fmt.Errorf("payment failed: %s", reason),
			)
		}

		if err := sendWebhookResult(hookURL, result); err != nil {
			slog.Error("Failed to send webhook result",
				slog.Any("error", err))
		}
	}()

	return api.StepResult{Success: true}, nil
}

func sendWebhookResult(webhookURL string, result api.StepResult) error {
	jsonData, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	slog.Info("Calling webhook",
		slog.String("webhook_url", webhookURL))

	resp, err := http.Post(
		webhookURL, "application/json", bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	slog.Info("Webhook response",
		slog.String("webhook_url", webhookURL),
		slog.Int("status_code", resp.StatusCode))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}
