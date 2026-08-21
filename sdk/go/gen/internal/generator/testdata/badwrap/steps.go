package badwrap

//argyll:wrap left, right -> total, extra
func Add(left, right int) (int, error) {
	return left + right, nil
}
