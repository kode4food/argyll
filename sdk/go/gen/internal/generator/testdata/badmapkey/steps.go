package badmapkey

type Args struct {
	Counts map[int]string
}

//argyll:step
func Take(args Args) error {
	return nil
}
