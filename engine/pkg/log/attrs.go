package log

import "log/slog"

func FlowID[T ~string](id T) slog.Attr {
	return slog.String("flow_id", string(id))
}

func StepID[T ~string](id T) slog.Attr {
	return slog.String("step_id", string(id))
}

func Status[T ~string](status T) slog.Attr {
	return slog.String("status", string(status))
}

func Token[T ~string](token T) slog.Attr {
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
