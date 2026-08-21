package badnamed

type (
	Tags []string

	Args struct {
		Tags Tags
	}
)

//argyll:step
func Take(args Args) error {
	return nil
}
