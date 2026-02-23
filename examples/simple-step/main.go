package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/log"
	"github.com/kode4food/argyll/sdks/go-builder"
)

const version = "dev"

func main() {
	engineURL := os.Getenv("ARGYLL_ENGINE_URL")
	if engineURL == "" {
		engineURL = "http://localhost:8080"
	}

	logger := log.New("simple-step-example", os.Getenv("ENV"), version)
	slog.SetDefault(logger)

	// Example 1: Simple text transformation (Ale script)
	textFormatterScript := `{
  :formatted_text (str "[" name "] " text)
}`

	client := builder.NewClient(engineURL, 30*time.Second)

	err := client.NewStep().WithName("Text Formatter").
		WithID("text-formatter").
		WithLabels(api.Labels{
			"description": "format text with a label",
			"domain":      "text",
			"capability":  "transform",
			"language":    "ale",
			"example":     "true",
		}).
		Required("text", api.TypeString).
		Required("name", api.TypeString).
		Output("formatted_text", api.TypeString).
		WithScript(textFormatterScript).
		Register(context.Background())

	if err != nil {
		slog.Error("Failed to register text formatter", log.Error(err))
		os.Exit(1)
	}

	// Example 2: Data transformation (Ale script)
	priceCalculatorScript := `(let* ([subtotal (* quantity unit_price)]
       [tax      (* subtotal 0.08)]
       [shipping (if (> quantity 5) 0.0 9.99)])
  {:subtotal subtotal
   :tax      tax
   :shipping shipping
   :total    (+ subtotal tax shipping)})`

	err = client.NewStep().WithName("Price Calculator").
		WithID("price-calculator").
		WithLabels(api.Labels{
			"description": "calculate pricing totals",
			"domain":      "pricing",
			"capability":  "calculate",
			"language":    "ale",
			"example":     "true",
		}).
		Required("quantity", api.TypeNumber).
		Required("unit_price", api.TypeNumber).
		Output("subtotal", api.TypeNumber).
		Output("tax", api.TypeNumber).
		Output("shipping", api.TypeNumber).
		Output("total", api.TypeNumber).
		WithScript(priceCalculatorScript).
		Register(context.Background())

	if err != nil {
		slog.Error("Failed to register price calculator", log.Error(err))
		os.Exit(1)
	}

	// Example 3: Conditional logic (Lua script)
	eligibilityCheckerScript := `local eligible = false
local reason = ""

if age < 18 then
	reason = "age below minimum (18)"
elseif age > 75 then
	reason = "age above maximum (75)"
elseif income < 30000 then
	reason = "income below minimum ($30,000)"
elseif credit_score < 600 then
	reason = "credit score below minimum (600)"
else
	eligible = true
	reason = "all criteria met"
end

return {
	eligible = eligible,
	reason = reason,
	risk_level = credit_score >= 750 and "low" or
				(credit_score >= 650 and "medium" or "high")
}`

	err = client.NewStep().WithName("Eligibility Checker").
		WithID("eligibility-checker").
		WithLabels(api.Labels{
			"description": "evaluate eligibility based on risk inputs",
			"domain":      "risk",
			"capability":  "decision",
			"language":    "lua",
			"example":     "true",
		}).
		Required("age", api.TypeNumber).
		Required("income", api.TypeNumber).
		Required("credit_score", api.TypeNumber).
		Output("eligible", api.TypeBoolean).
		Output("reason", api.TypeString).
		Output("risk_level", api.TypeString).
		WithScriptLanguage("lua", eligibilityCheckerScript).
		Register(context.Background())

	if err != nil {
		slog.Error("Failed to register eligibility checker", log.Error(err))
		os.Exit(1)
	}

	slog.Info("All script-based steps registered successfully")
}
