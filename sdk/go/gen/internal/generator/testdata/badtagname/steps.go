package badtagname

type Args struct {
	Amount int `argyll:"order amount"`
}

//argyll:step
func Take(args Args) error {
	return nil
}
