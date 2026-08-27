// Package example contains the step functions used to exercise argyll-gen
package example

import (
	"errors"

	argyll "github.com/kode4food/argyll/sdk/go"
)

type (
	// RiskArgs are the inputs of the CalculateRisk step
	RiskArgs struct {
		CustomerID string
		Amount     int64
		Tags       []string
		Note       *string
	}

	// RiskResult are the outputs of the CalculateRisk step
	RiskResult struct {
		Score    int
		Approved bool
	}

	// Address is a nested struct input
	Address struct {
		City string
		Zip  string
	}

	// Node is a recursive type, through both a slice and a pointer
	Node struct {
		Name     string
		Children []Node
		Next     *Node
	}

	// EnrollArgs are the inputs of the Enroll step
	EnrollArgs struct {
		Address  Address
		Limits   map[string]int
		Currency string `argyll:"iso_currency"`
		Scratch  string `argyll:"-"`
	}

	// ChargeArgs are the inputs of the ChargeCard step
	ChargeArgs struct {
		OrderID  string `argyll:"for_each:true;collect:all;compensated:true"`
		Note     string `argyll:"role:optional"`
		Currency string `argyll:"role:optional;default:USD;deadline:5000"`
		Gateway  string `argyll:"role:const;value:stripe"`
		FlowID   string `argyll:"flow;role:meta;key:flow_id"`
		Amount   int64  `argyll-match:"lua:return args.amount > 0"`
	}

	// ChargeResult is the output of the ChargeCard step
	ChargeResult struct {
		ChargeID string `argyll:"compensated:true"`
	}
)

//go:generate go run ../gen/cmd/argyll-gen .

var ErrAmountNegative = errors.New("amount must not be negative")

//argyll:step
//argyll:memoize
//argyll:labels description: score a customer for risk; domain: risk
func CalculateRisk(args RiskArgs) (RiskResult, error) {
	if args.Amount < 0 {
		return RiskResult{}, ErrAmountNegative
	}
	return RiskResult{
		Score:    int(args.Amount / 100),
		Approved: args.Amount < 10_000,
	}, nil
}

//argyll:step
func Greet(args struct{ Name string }) struct{ Greeting string } {
	return struct{ Greeting string }{
		Greeting: "hello " + args.Name,
	}
}

//argyll:step
func Enroll(args EnrollArgs) struct {
	City     string
	Limits   map[string]int
	Currency string `argyll:"iso_currency"`
} {
	return struct {
		City     string
		Limits   map[string]int
		Currency string `argyll:"iso_currency"`
	}{
		City:     args.Address.City,
		Limits:   args.Limits,
		Currency: args.Currency,
	}
}

//argyll:step
func Walk(args struct{ Root Node }) struct{ Root Node } {
	return struct{ Root Node }{Root: args.Root}
}

//argyll:step
func Explode(args RiskArgs) RiskResult {
	panic("boom: " + args.CustomerID)
}

//argyll:step
func Audit(args struct{ Event string }) error {
	if args.Event == "" {
		return errors.New("empty event")
	}
	return nil
}

//argyll:wrap (customer-id, amount) -> (score, approved)
func ScoreCustomer(customerID string, amount int64) (int, bool, error) {
	if customerID == "" {
		return 0, false, errors.New("customer id required")
	}
	return int(amount / 100), amount < 10_000, nil
}

//argyll:wrap rate-customer-v2 -> (score, approved)
func RateCustomer(customerID string, amount int64) (int, bool, error) {
	if customerID == "" {
		return 0, false, errors.New("customer id required")
	}
	return int(amount / 100), amount < 10_000, nil
}

//argyll:wrap
func GradeCustomer(
	customerID string, amount int64,
) (score int, approved bool, err error) {
	if customerID == "" {
		return 0, false, errors.New("customer id required")
	}
	return int(amount / 100), amount < 10_000, nil
}

//argyll:step charge-card-v2;name:Charge Card (v2)
//argyll:compensate refundCard
//argyll:http timeout: 2500
//argyll:predicate return args.amount > 0
func ChargeCard(args ChargeArgs) ChargeResult {
	return ChargeResult{
		ChargeID: args.Gateway + ":" + args.OrderID + ":" + args.Currency,
	}
}

type refundCardArgs struct {
	OrderID  string
	ChargeID string
}

func refundCard(_ refundCardArgs) error {
	return nil
}

//argyll:step
//argyll:labels domain: risk
func Reject(args struct{ Reason string }) error {
	return argyll.NotFound("%s", args.Reason)
}
