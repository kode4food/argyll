package main

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"time"

	"github.com/kode4food/spuds/engine/pkg/api"
	"github.com/kode4food/spuds/engine/pkg/builder"
)

func main() {
	if err := builder.SetupStep(
		"Notification Sender", build, handle,
	); err != nil {
		slog.Error("Failed to setup notification sender",
			slog.Any("error", err))
		os.Exit(1)
	}
}

func build(step *builder.Step) *builder.Step {
	return step.
		Required("payment_result", api.TypeObject).
		Required("reservation", api.TypeObject).
		Required("user_info", api.TypeObject)
}

func handle(ctx context.Context, args api.Args) (api.StepResult, error) {
	// Extract payment result
	paymentResult, ok := args["payment_result"].(map[string]any)
	if !ok {
		return *api.NewResult().WithError(
			fmt.Errorf("payment_result must be an object"),
		), nil
	}

	// Extract reservation
	reservation, ok := args["reservation"].(map[string]any)
	if !ok {
		return *api.NewResult().WithError(
			fmt.Errorf("reservation must be an object"),
		), nil
	}

	// Extract user info
	userInfo, ok := args["user_info"].(map[string]any)
	if !ok {
		return *api.NewResult().WithError(
			fmt.Errorf("user_info must be an object"),
		), nil
	}

	orderID, _ := paymentResult["order_id"].(string)
	amount, _ := paymentResult["amount"].(float64)
	paymentID, _ := paymentResult["payment_id"].(string)
	reservationID, _ := reservation["reservation_id"].(string)
	userID, _ := userInfo["id"].(string)
	userName, _ := userInfo["name"].(string)
	userEmail, _ := userInfo["email"].(string)
	preferredMethod, _ := userInfo["preferred_method"].(string)

	slog.Info("Sending order confirmation notifications",
		slog.String("order_id", orderID),
		slog.String("user_name", userName),
		slog.String("preferred_method", preferredMethod))

	var messageIDs []string
	var channels []string

	// Send via preferred method
	switch preferredMethod {
	case "email":
		msgID, err := sendEmail(userEmail, orderID, amount)
		if err != nil {
			return *api.NewResult().WithError(
				fmt.Errorf("failed to send email: %w", err),
			), nil
		}
		messageIDs = append(messageIDs, msgID)
		channels = append(channels, "email")

	case "sms":
		msgID, err := sendSMS(userID, orderID)
		if err != nil {
			return *api.NewResult().WithError(
				fmt.Errorf("failed to send SMS: %w", err),
			), nil
		}
		messageIDs = append(messageIDs, msgID)
		channels = append(channels, "sms")

	case "webhook":
		payload := map[string]any{
			"order_id":       orderID,
			"payment_id":     paymentID,
			"reservation_id": reservationID,
			"amount":         amount,
		}
		msgID, err := sendWebhook(userID, orderID, payload)
		if err != nil {
			return *api.NewResult().WithError(
				fmt.Errorf("failed to send webhook: %w", err),
			), nil
		}
		messageIDs = append(messageIDs, msgID)
		channels = append(channels, "webhook")

	default:
		// Fallback to email if unknown method
		msgID, err := sendEmail(userEmail, orderID, amount)
		if err != nil {
			return *api.NewResult().WithError(
				fmt.Errorf("failed to send fallback email: %w", err),
			), nil
		}
		messageIDs = append(messageIDs, msgID)
		channels = append(channels, "email")
	}

	// Always send a secondary confirmation email for audit trail
	if preferredMethod != "email" {
		msgID, _ := sendEmail(userEmail, orderID, amount)
		messageIDs = append(messageIDs, msgID)
		channels = append(channels, "email_backup")
	}

	slog.Info("All notifications sent successfully",
		slog.String("order_id", orderID),
		slog.Int("notification_count", len(messageIDs)),
		slog.Any("channels", channels),
		slog.Any("message_ids", messageIDs))

	return *api.NewResult(), nil
}

func sendWebhook(userID, orderID string, payload map[string]any) (string, error) {
	time.Sleep(time.Duration(5+rand.Intn(5)) * time.Second)
	messageID := fmt.Sprintf("webhook-%d", time.Now().UnixNano())
	slog.Info("Webhook delivered",
		slog.String("to_user", userID),
		slog.String("message_id", messageID))
	return messageID, nil
}

func sendEmail(userEmail, orderID string, amount float64) (string, error) {
	time.Sleep(time.Duration(5+rand.Intn(5)) * time.Second)
	messageID := fmt.Sprintf("email-%d", time.Now().UnixNano())
	slog.Info("Email sent",
		slog.String("to", userEmail),
		slog.String("message_id", messageID))
	return messageID, nil
}

func sendSMS(userID, orderID string) (string, error) {
	time.Sleep(time.Duration(5+rand.Intn(5)) * time.Second)
	messageID := fmt.Sprintf("sms-%d", time.Now().UnixNano())
	slog.Info("SMS sent",
		slog.String("to_user", userID),
		slog.String("message_id", messageID))
	return messageID, nil
}
