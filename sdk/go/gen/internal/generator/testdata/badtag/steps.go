package badtag

type Args struct {
	Amount int `argyll:"amount,omitempty"`
}

//argyll:step
func Take(args Args) error {
	return nil
}
