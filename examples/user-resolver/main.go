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

type UserInfo struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Email           string  `json:"email"`
	AccountType     string  `json:"account_type"`
	CreditLimit     float64 `json:"credit_limit"`
	MemberSince     string  `json:"member_since"`
	PreferredMethod string  `json:"preferred_method"`
}

// Simulated user database
var userDatabase = map[string]UserInfo{
	"user-123": {
		ID:              "user-123",
		Name:            "Alice Johnson",
		Email:           "alice@example.com",
		AccountType:     "premium",
		CreditLimit:     5000.00,
		MemberSince:     "2020-01-15",
		PreferredMethod: "email",
	},
	"user-456": {
		ID:              "user-456",
		Name:            "Bob Smith",
		Email:           "bob@example.com",
		AccountType:     "standard",
		CreditLimit:     1000.00,
		MemberSince:     "2023-06-01",
		PreferredMethod: "sms",
	},
	"user-789": {
		ID:              "user-789",
		Name:            "Carol Williams",
		Email:           "carol@example.com",
		AccountType:     "premium",
		CreditLimit:     10000.00,
		MemberSince:     "2019-03-20",
		PreferredMethod: "webhook",
	},
}

func main() {
	if err := builder.SetupStep(
		"User Resolver", build, handle,
	); err != nil {
		slog.Error("Failed to setup user resolver",
			slog.Any("error", err))
		os.Exit(1)
	}
}

func build(step *builder.Step) *builder.Step {
	return step.
		Optional("user_id", api.TypeString, `"user-123"`).
		Output("user_info", api.TypeObject)
}

func handle(ctx context.Context, args api.Args) (api.StepResult, error) {
	time.Sleep(time.Duration(5+rand.Intn(5)) * time.Second)

	userID, hasUserID := args["user_id"].(string)
	if !hasUserID || userID == "" {
		// Default to first user if no ID provided
		userID = "user-123"
		slog.Info("No user_id provided, using default",
			slog.String("user_id", userID))
	}

	userInfo, exists := userDatabase[userID]
	if !exists {
		slog.Warn("User not found", slog.String("user_id", userID))
		return *api.NewResult().WithError(
			fmt.Errorf("user not found: %s", userID),
		), nil
	}

	slog.Info("User resolved successfully",
		slog.String("user_id", userInfo.ID),
		slog.String("name", userInfo.Name),
		slog.String("account_type", userInfo.AccountType))

	return *api.NewResult().WithOutput("user_info", userInfo), nil
}
