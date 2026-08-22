package badfieldattrs

type Args struct {
	Amount int `argyll:"amount -> (total)"`
}

//argyll:step
func Take(args Args) error {
	return nil
}
