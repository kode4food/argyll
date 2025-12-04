package log

import "log/slog"

// str captures types whose underlying type is string (e.g., custom ID types)
type str interface {
	~string
}

func FlowID[T str](id T) slog.Attr {
	return slog.String("flow_id", string(id))
}

func StepID[T str](id T) slog.Attr {
	return slog.String("step_id", string(id))
}

func Status[T str](status T) slog.Attr {
	return slog.String("status", string(status))
}

func Token[T str](token T) slog.Attr {
	return slog.String("token", string(token))
}

func Error(err error) slog.Attr {
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	return slog.String("error", msg)
}

func ErrorString(msg string) slog.Attr {
	return slog.String("error", msg)
}
