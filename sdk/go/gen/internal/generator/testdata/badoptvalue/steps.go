package badoptvalue

type Args struct {
	Amount int `argyll:"amount;default"`
}

//argyll:step
func Take(args Args) error {
	return nil
}
