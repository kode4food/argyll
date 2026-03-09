package call

type (
	// Call is a deferred error-returning function
	Call func() error

	// Applier mutates a value in place
	Applier[T any] func(T)

	// Constructor creates a new instance of T
	Constructor[T any] func() T
)

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

// Apply runs appliers in order
func Apply[T any](v T, apps ...Applier[T]) {
	for _, app := range apps {
		app(v)
	}
}

// Defaults builds a defaulted value, then applies the provided appliers
func Defaults[T any](defaults Constructor[T]) func(...Applier[T]) T {
	return func(apps ...Applier[T]) T {
		v := defaults()
		Apply(v, apps...)
		return v
	}
}
