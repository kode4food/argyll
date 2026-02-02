package main

import (
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/sdks/go-builder"
	"github.com/kode4food/argyll/engine/pkg/log"
)

const version = "dev"

func main() {
	engineURL := os.Getenv("ARGYLL_ENGINE_URL")
	if engineURL == "" {
		engineURL = "http://localhost:8080"
	}

	logger := log.New("notification-sender-example", os.Getenv("ENV"), version)
	slog.SetDefault(logger)

	client := builder.NewClient(engineURL, 30*time.Second)

	err := client.NewStep().WithName("Notification Sender").
		WithLabels(api.Labels{
			"description": "send order confirmation notifications",
			"domain":      "notifications",
			"capability":  "send",
			"example":     "true",
		}).
		Required("payment_result", api.TypeObject).
		Required("reservation", api.TypeObject).
		Required("user_info", api.TypeObject).
		Start(handle)

	if err != nil {
		slog.Error("Failed to setup notification sender",
			log.Error(err))
		os.Exit(1)
	}
}

func handle(_ *builder.StepContext, args api.Args) (api.StepResult, error) {
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
		slog.String("channels", strings.Join(channels, ",")),
		slog.String("message_ids", strings.Join(messageIDs, ",")))

	return *api.NewResult(), nil
}

func sendWebhook(userID, _ string, _ map[string]any) (string, error) {
	time.Sleep(time.Duration(5+rand.Intn(5)) * time.Second)
	messageID := fmt.Sprintf("webhook-%d", time.Now().UnixNano())
	slog.Info("Webhook delivered",
		slog.String("to_user", userID),
		slog.String("message_id", messageID))
	return messageID, nil
}

func sendEmail(userEmail, _ string, _ float64) (string, error) {
	time.Sleep(time.Duration(5+rand.Intn(5)) * time.Second)
	messageID := fmt.Sprintf("email-%d", time.Now().UnixNano())
	slog.Info("Email sent",
		slog.String("to", userEmail),
		slog.String("message_id", messageID))
	return messageID, nil
}

func sendSMS(userID, _ string) (string, error) {
	time.Sleep(time.Duration(5+rand.Intn(5)) * time.Second)
	messageID := fmt.Sprintf("sms-%d", time.Now().UnixNano())
	slog.Info("SMS sent",
		slog.String("to_user", userID),
		slog.String("message_id", messageID))
	return messageID, nil
}
