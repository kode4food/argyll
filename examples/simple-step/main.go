package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/log"
	argyll "github.com/kode4food/argyll/sdk/go"
)

const version = "dev"

func main() {
	engineURL := os.Getenv("ARGYLL_ENGINE_URL")
	if engineURL == "" {
		engineURL = "http://localhost:8080"
	}

	logger := log.New("simple-step-example", os.Getenv("ENV"), version)
	slog.SetDefault(logger)

	// Example 1: Simple text transformation (Lua script)
	textFormatterScript := `return {
	formatted_text = "[" .. name .. "] " .. text
}`

	client := argyll.NewClient(engineURL, 30*time.Second)

	err := client.NewStep().WithName("Text Formatter").
		WithID("text-formatter").
		WithDescription("format text with a label").
		WithTags(
			"domain:text", "capability:transform", "language:lua", "example",
		).
		Required("text").
		Required("name").
		Output("formatted_text").
		WithScript(api.ScriptConfig{
			Language: api.ScriptLangLua,
			Source:   textFormatterScript,
		}).
		Register(context.Background())

	if err != nil {
		slog.Error("Failed to register text formatter", log.Error(err))
		os.Exit(1)
	}

	// Example 2: Data transformation (Lua script)
	priceCalculatorScript := `local subtotal = quantity * unit_price
local tax = subtotal * 0.08
local shipping = quantity > 5 and 0.0 or 9.99
return {
	subtotal = subtotal,
	tax = tax,
	shipping = shipping,
	total = subtotal + tax + shipping
}`

	err = client.NewStep().WithName("Price Calculator").
		WithID("price-calculator").
		WithDescription("calculate pricing totals").
		WithTags(
			"domain:pricing", "capability:calculate", "language:lua", "example",
		).
		Required("quantity").
		Required("unit_price").
		Output("subtotal").
		Output("tax").
		Output("shipping").
		Output("total").
		WithScript(api.ScriptConfig{
			Language: api.ScriptLangLua,
			Source:   priceCalculatorScript,
		}).
		Register(context.Background())

	if err != nil {
		slog.Error("Failed to register price calculator", log.Error(err))
		os.Exit(1)
	}

	// Example 3: Conditional logic (ArgEx script)
	eligibilityCheckerScript := `let reason = case {
	age < 18: "age below minimum (18)",
	age > 75: "age above maximum (75)",
	income < 30000: "income below minimum ($30,000)",
	credit_score < 600: "credit score below minimum (600)",
	"all criteria met",
}: {
	eligible: reason == "all criteria met",
	reason: reason,
	risk_level: case {
		credit_score >= 750: "low",
		credit_score >= 650: "medium",
		"high",
	},
}`

	err = client.NewStep().WithName("Eligibility Checker").
		WithID("eligibility-checker").
		WithDescription("evaluate eligibility based on risk inputs").
		WithTags(
			"domain:risk", "capability:decision", "language:argex", "example",
		).
		Required("age").
		Required("income").
		Required("credit_score").
		Output("eligible").
		Output("reason").
		Output("risk_level").
		WithScript(api.ScriptConfig{
			Language: api.ScriptLangArgyll,
			Source:   eligibilityCheckerScript,
		}).
		Register(context.Background())

	if err != nil {
		slog.Error("Failed to register eligibility checker", log.Error(err))
		os.Exit(1)
	}

	slog.Info("All script-based steps registered successfully")
}
