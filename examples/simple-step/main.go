package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/kode4food/spuds/engine/pkg/api"
	"github.com/kode4food/spuds/engine/pkg/builder"
)

func main() {
	engineURL := os.Getenv("SPUDS_ENGINE_URL")
	if engineURL == "" {
		engineURL = "http://localhost:8080"
	}

	// Example 1: Simple text transformation (Ale script)
	textFormatterScript := `{
  :formatted_text (str "[" name "] " text)
}`

	textFormatter := &api.Step{
		ID:   "text-formatter",
		Name: "Text Formatter",
		Type: api.StepTypeScript,
		Attributes: map[api.Name]*api.AttributeSpec{
			"text": {
				Role: api.RoleRequired,
				Type: api.TypeString,
			},
			"name": {
				Role: api.RoleRequired,
				Type: api.TypeString,
			},
			"formatted_text": {
				Role: api.RoleOutput,
				Type: api.TypeString,
			},
		},
		Version: "1.0.0",
		Script: &api.ScriptConfig{
			Language: "ale",
			Script:   textFormatterScript,
		},
	}

	// Example 2: Data transformation (Ale script)
	priceCalculatorScript := `{
  :subtotal (* quantity unit_price)
  :tax (* quantity unit_price 0.08)
  :shipping (if (> quantity 5) 0.0 9.99)
  :total (+ (* quantity unit_price)
            (* quantity unit_price 0.08)
            (if (> quantity 5) 0.0 9.99))
}`

	priceCalculator := &api.Step{
		ID:   "price-calculator",
		Name: "Price Calculator",
		Type: api.StepTypeScript,
		Attributes: map[api.Name]*api.AttributeSpec{
			"quantity": {
				Role: api.RoleRequired,
				Type: api.TypeNumber,
			},
			"unit_price": {
				Role: api.RoleRequired,
				Type: api.TypeNumber,
			},
			"subtotal": {
				Role: api.RoleOutput,
				Type: api.TypeNumber,
			},
			"tax": {
				Role: api.RoleOutput,
				Type: api.TypeNumber,
			},
			"shipping": {
				Role: api.RoleOutput,
				Type: api.TypeNumber,
			},
			"total": {
				Role: api.RoleOutput,
				Type: api.TypeNumber,
			},
		},
		Version: "1.0.0",
		Script: &api.ScriptConfig{
			Language: "ale",
			Script:   priceCalculatorScript,
		},
	}

	// Example 3: Conditional logic (Lua script)
	eligibilityCheckerScript := `local age = tonumber(args.age)
local income = tonumber(args.income)
local credit_score = tonumber(args.credit_score)

local eligible = false
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

	eligibilityChecker := &api.Step{
		ID:   "eligibility-checker",
		Name: "Eligibility Checker",
		Type: api.StepTypeScript,
		Attributes: map[api.Name]*api.AttributeSpec{
			"age": {
				Role: api.RoleRequired,
				Type: api.TypeNumber,
			},
			"income": {
				Role: api.RoleRequired,
				Type: api.TypeNumber,
			},
			"credit_score": {
				Role: api.RoleRequired,
				Type: api.TypeNumber,
			},
			"eligible": {
				Role: api.RoleOutput,
				Type: api.TypeBoolean,
			},
			"reason": {
				Role: api.RoleOutput,
				Type: api.TypeString,
			},
			"risk_level": {
				Role: api.RoleOutput,
				Type: api.TypeString,
			},
		},
		Version: "1.0.0",
		Script: &api.ScriptConfig{
			Language: "lua",
			Script:   eligibilityCheckerScript,
		},
	}

	// Register all example steps
	client := builder.NewClient(engineURL, 30*time.Second)
	steps := []*api.Step{textFormatter, priceCalculator, eligibilityChecker}

	for _, step := range steps {
		if err := client.RegisterStep(context.Background(), step); err != nil {
			slog.Error("Failed to register script step",
				slog.String("step_id", string(step.ID)),
				slog.Any("error", err))
			os.Exit(1)
		}

		slog.Info("Successfully registered script step",
			slog.String("step_id", string(step.ID)),
			slog.String("name", string(step.Name)),
			slog.String("type", string(step.Type)),
			slog.String("language", step.Script.Language))
	}

	slog.Info("All script-based steps registered successfully",
		slog.Int("count", len(steps)))
}
