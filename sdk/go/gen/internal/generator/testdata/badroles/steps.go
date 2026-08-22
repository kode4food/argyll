package badroles

type Args struct {
	Amount int `argyll:"amount;role:const;default:1"`
}

//argyll:step
func Take(args Args) error {
	return nil
}
