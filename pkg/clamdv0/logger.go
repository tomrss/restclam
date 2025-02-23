package clamd

// interface friendly with rs/zerolog

type LogEvent interface {
	Str(key string, val string) LogEvent
	Int(key string, val int) LogEvent
	Uint(key string, val uint) LogEvent
	Bool(key string, val bool) LogEvent
	Err(err error) LogEvent
	Msg(msg string)
}

type Logger interface {
	Trace() LogEvent
	Debug() LogEvent
	Info() LogEvent
	Warn() LogEvent
	Error() LogEvent
}

// a noop implementation

// noopLogEvent does nothing.
type noopLogEvent struct{}

func (e *noopLogEvent) Str(_ string, _ string) LogEvent {
	return e
}

func (e *noopLogEvent) Int(_ string, _ int) LogEvent {
	return e
}

func (e *noopLogEvent) Uint(_ string, _ uint) LogEvent {
	return e
}

func (e *noopLogEvent) Bool(_ string, _ bool) LogEvent {
	return e
}

func (e *noopLogEvent) Err(_ error) LogEvent {
	return e
}

func (e *noopLogEvent) Msg(_ string) {
}

// noopLogger does nothing.
type noopLogger struct{}

func (l *noopLogger) Trace() LogEvent {
	return &noopLogEvent{}
}

func (l *noopLogger) Debug() LogEvent {
	return &noopLogEvent{}
}

func (l *noopLogger) Info() LogEvent {
	return &noopLogEvent{}
}

func (l *noopLogger) Warn() LogEvent {
	return &noopLogEvent{}
}

func (l *noopLogger) Error() LogEvent {
	return &noopLogEvent{}
}

// ensure interfaces are respected
var (
	_ LogEvent = &noopLogEvent{}
	_ Logger   = &noopLogger{}
)
