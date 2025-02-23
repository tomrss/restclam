//nolint:zerologlint
package main

import (
	"github.com/rs/zerolog"
	clamd "github.com/tomrss/restclam/pkg/clamdv0"
)

var _ clamd.Logger = &clamdV0LogDriver{}

type clamdV0LogEventDriver struct {
	event *zerolog.Event
}

type clamdV0LogDriver struct {
	logger *zerolog.Logger
}

func newClamdV0LogDriver(logger *zerolog.Logger) *clamdV0LogDriver {
	return &clamdV0LogDriver{logger}
}

func (d *clamdV0LogDriver) Trace() clamd.LogEvent {
	return &clamdV0LogEventDriver{d.logger.Trace()}
}

func (d *clamdV0LogDriver) Debug() clamd.LogEvent {
	return &clamdV0LogEventDriver{d.logger.Debug()}
}

func (d *clamdV0LogDriver) Info() clamd.LogEvent {
	return &clamdV0LogEventDriver{d.logger.Info()}
}

func (d *clamdV0LogDriver) Warn() clamd.LogEvent {
	return &clamdV0LogEventDriver{d.logger.Warn()}
}

func (d *clamdV0LogDriver) Error() clamd.LogEvent {
	return &clamdV0LogEventDriver{d.logger.Error()}
}

func (e *clamdV0LogEventDriver) Str(key string, val string) clamd.LogEvent {
	e.event = e.event.Str(key, val)
	return e
}

func (e *clamdV0LogEventDriver) Int(key string, val int) clamd.LogEvent {
	e.event = e.event.Int(key, val)
	return e
}
func (e *clamdV0LogEventDriver) Uint(key string, val uint) clamd.LogEvent {
	e.event = e.event.Uint(key, val)
	return e
}

func (e *clamdV0LogEventDriver) Bool(key string, val bool) clamd.LogEvent {
	e.event = e.event.Bool(key, val)
	return e
}

func (e *clamdV0LogEventDriver) Err(err error) clamd.LogEvent {
	e.event = e.event.Err(err)
	return e
}

func (e *clamdV0LogEventDriver) Msg(msg string) {
	e.event.Msg(msg)
}
