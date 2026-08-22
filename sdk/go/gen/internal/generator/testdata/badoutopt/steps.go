package badoutopt

type Res struct {
	Score int `argyll:"score;role:optional"`
}

//argyll:step
func Take(args struct{ Name string }) Res {
	return Res{}
}
