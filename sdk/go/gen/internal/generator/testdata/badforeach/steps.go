package badforeach

type Args struct {
	Amount int `argyll:"amount;role:const;value:1;for_each:true"`
}

//argyll:step
func Take(args Args) error {
	return nil
}
