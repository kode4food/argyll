package call

// Call is a deferred error-returning function
type Call func() error

// Perform runs calls in order and stops on the first error
func Perform(calls ...Call) error {
	for _, call := range calls {
		if err := call(); err != nil {
			return err
		}
	}
	return nil
}

// WithArg binds one argument to a call
func WithArg[Arg any](call func(Arg) error, arg Arg) Call {
	return func() error {
		return call(arg)
	}
}

// WithArgs binds two arguments to a call
func WithArgs[Arg1, Arg2 any](
	call func(Arg1, Arg2) error, arg1 Arg1, arg2 Arg2,
) Call {
	return func() error {
		return call(arg1, arg2)
	}
}
