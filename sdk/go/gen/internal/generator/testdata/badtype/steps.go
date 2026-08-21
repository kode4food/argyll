package badtype

type Args struct {
	Ch chan int
}

//argyll:step
func Take(args Args) error {
	return nil
}
