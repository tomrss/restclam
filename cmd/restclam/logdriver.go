//nolint:zerologlint
package main

import (
	"github.com/rs/zerolog"
	"github.com/tomrss/restclam/pkg/clamd"
)

var _ clamd.Logger = &clamdLogDriver{}

type clamdLogEventDriver struct {
	event *zerolog.Event
}

type clamdLogDriver struct {
	logger *zerolog.Logger
}

func newClamdLogDriver(logger *zerolog.Logger) *clamdLogDriver {
	return &clamdLogDriver{logger}
}

func (d *clamdLogDriver) Trace() clamd.LogEvent {
	return &clamdLogEventDriver{d.logger.Trace()}
}

func (d *clamdLogDriver) Debug() clamd.LogEvent {
	return &clamdLogEventDriver{d.logger.Debug()}
}

func (d *clamdLogDriver) Info() clamd.LogEvent {
	return &clamdLogEventDriver{d.logger.Info()}
}

func (d *clamdLogDriver) Warn() clamd.LogEvent {
	return &clamdLogEventDriver{d.logger.Warn()}
}

func (d *clamdLogDriver) Error() clamd.LogEvent {
	return &clamdLogEventDriver{d.logger.Error()}
}

func (e *clamdLogEventDriver) Str(key string, val string) clamd.LogEvent {
	e.event = e.event.Str(key, val)
	return e
}

func (e *clamdLogEventDriver) Int(key string, val int) clamd.LogEvent {
	e.event = e.event.Int(key, val)
	return e
}
func (e *clamdLogEventDriver) Uint(key string, val uint) clamd.LogEvent {
	e.event = e.event.Uint(key, val)
	return e
}

func (e *clamdLogEventDriver) Bool(key string, val bool) clamd.LogEvent {
	e.event = e.event.Bool(key, val)
	return e
}

func (e *clamdLogEventDriver) Err(err error) clamd.LogEvent {
	e.event = e.event.Err(err)
	return e
}

func (e *clamdLogEventDriver) Msg(msg string) {
	e.event.Msg(msg)
}
