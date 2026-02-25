package engine

func performCalls(calls ...func() error) error {
	for _, call := range calls {
		if err := call(); err != nil {
			return err
		}
	}
	return nil
}

func withArg[Arg any](call func(Arg) error, arg Arg) func() error {
	return func() error {
		return call(arg)
	}
}

func withArgs[Arg1, Arg2 any](
	call func(Arg1, Arg2) error, arg1 Arg1, arg2 Arg2,
) func() error {
	return func() error {
		return call(arg1, arg2)
	}
}
