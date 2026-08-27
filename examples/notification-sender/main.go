package main

import (
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"strings"
	"time"

	argyll "github.com/kode4food/argyll/sdk/go"
)

type (
	// NotificationArgs are the inputs of the notification sender step
	NotificationArgs struct {
		PaymentResult PaymentResult
		Reservation   Reservation
		UserInfo      UserInfo
	}

	// PaymentResult is the outcome of the upstream payment step
	PaymentResult struct {
		OrderID   string
		PaymentID string
		Amount    float64
	}

	// Reservation is the stock held for the order
	Reservation struct {
		ReservationID string
	}

	// UserInfo identifies the recipient and how they want to be reached
	UserInfo struct {
		ID              string
		Name            string
		Email           string
		PreferredMethod string
	}
)

//go:generate go run github.com/kode4food/argyll/sdk/go/gen/cmd/argyll-gen -server .

//argyll:step
//argyll:labels description: send order confirmation notifications
//argyll:labels domain: notifications; capability: send; example: true
func NotificationSender(args NotificationArgs) error {
	payment := args.PaymentResult
	user := args.UserInfo

	slog.Info("Sending order confirmation notifications",
		slog.String("order_id", payment.OrderID),
		slog.String("user_name", user.Name),
		slog.String("preferred_method", user.PreferredMethod))

	var messageIDs []string
	var channels []string

	// Send via preferred method
	switch user.PreferredMethod {
	case "email":
		msgID, err := sendEmail(user.Email, payment.OrderID, payment.Amount)
		if err != nil {
			return sendFailed("email", err)
		}
		messageIDs = append(messageIDs, msgID)
		channels = append(channels, "email")

	case "sms":
		msgID, err := sendSMS(user.ID, payment.OrderID)
		if err != nil {
			return sendFailed("SMS", err)
		}
		messageIDs = append(messageIDs, msgID)
		channels = append(channels, "sms")

	case "webhook":
		payload := map[string]any{
			"order_id":       payment.OrderID,
			"payment_id":     payment.PaymentID,
			"reservation_id": args.Reservation.ReservationID,
			"amount":         payment.Amount,
		}
		msgID, err := sendWebhook(user.ID, payment.OrderID, payload)
		if err != nil {
			return sendFailed("webhook", err)
		}
		messageIDs = append(messageIDs, msgID)
		channels = append(channels, "webhook")

	default:
		// Fallback to email if unknown method
		msgID, err := sendEmail(user.Email, payment.OrderID, payment.Amount)
		if err != nil {
			return sendFailed("fallback email", err)
		}
		messageIDs = append(messageIDs, msgID)
		channels = append(channels, "email")
	}

	// Always send a secondary confirmation email for audit trail
	if user.PreferredMethod != "email" {
		msgID, _ := sendEmail(user.Email, payment.OrderID, payment.Amount)
		messageIDs = append(messageIDs, msgID)
		channels = append(channels, "email_backup")
	}

	slog.Info("All notifications sent successfully",
		slog.String("order_id", payment.OrderID),
		slog.Int("notification_count", len(messageIDs)),
		slog.String("channels", strings.Join(channels, ",")),
		slog.String("message_ids", strings.Join(messageIDs, ",")))

	return nil
}

func sendFailed(channel string, err error) error {
	return argyll.NewHTTPError(http.StatusBadGateway,
		fmt.Sprintf("failed to send %s: %v", channel, err))
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
