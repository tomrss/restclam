package clamd

import (
	"fmt"
	"io"
	"regexp"
	"time"
)

type SessionEventType int

const (
	SessionHeartBeat SessionEventType = iota
	SessionDisconnected
	SessionClosed
	SessionError
)

type SessionEventSeverity int

const (
	// Information
	EventInfo SessionEventSeverity = iota
	// Error that should not affect any operation
	EventWarning
	// Error that interrupts an operation in the session
	EventError
	// Error that brings down the whole session
	EventFatal
)

type SessionEvent struct {
	Type     SessionEventType
	Severity SessionEventSeverity
	Error    error
	Source   string
	Session  *Session
}

type Session struct {
	id   int
	opts SessionOpts

	// mutable state and channels
	clamd           *Clamd
	heartbeatTicker *time.Ticker
	events          chan SessionEvent
}

func (s *Session) EventStream() {
	panic("unimplemented")
}

type SessionOpts struct {
	Opts

	Network           string
	Address           string
	HeartbeatInterval time.Duration
	ConnectRetries    RetryOpts
	CommandRetries    RetryOpts
}

type RetryOpts struct {
	MaxRetries int
	Backoff    func(retryCount int) time.Duration
}

func OpenSession(opts SessionOpts) (*Session, error) {
	s := &Session{opts: opts}

	if err := s.connectClamd(); err != nil {
		return nil, err
	}

	if err := s.clamd.Idsession(); err != nil {
		return nil, err
	}

	s.events = make(chan SessionEvent)
	s.heartbeatTicker = time.NewTicker(s.opts.HeartbeatInterval)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.sessionEventDisconnected(r)
				s.heartbeatTicker.Stop()
				close(s.events)
			}
		}()

		forcefullyDisconnected := false
		for range s.heartbeatTicker.C {
			if err := s.heartbeat(); err != nil {
				s.sessionEventDisconnected(err)
				s.heartbeatTicker.Stop()
				forcefullyDisconnected = true
				break
			}
			s.sessionEventHeartbeat()
		}

		if !forcefullyDisconnected {
			// gracefully disconnected
			s.sessionEventClosed()
		}
		close(s.events)
	}()

	return s, nil
}

func (s *Session) Close() error {
	if s.clamd == nil {
		return nil
	}

	err := s.clamd.End()
	if err != nil {
		return fmt.Errorf("unable to end clamd session: %w", err)
	}

	s.heartbeatTicker.Stop()

	if err := s.clamd.Close(); err != nil {
		return fmt.Errorf("unable to close clamd connection: %w", err)
	}

	s.clamd = nil
	return nil
}

// delegate command methods

func (s *Session) Ping() (string, error) {
	return s.clamd.Ping()
}

func (s *Session) Version() (string, error) {
	return s.clamd.Version()
}

func (s *Session) Stats() (string, error) {
	return s.clamd.Stats()
}

func (s *Session) Scan(path string) (*ScanResult, error) {
	return s.clamd.Scan(path)
}

func (s *Session) Instream(r io.Reader) (*ScanResult, error) {
	return s.clamd.Instream(r)
}

func (s *Session) Events() <-chan SessionEvent {
	// just a wrapper to return readonly channel
	return s.events
}

func (s *Session) ID() int {
	return s.id
}

func (s *Session) connectClamd() error {
	maxRetries := s.opts.ConnectRetries.MaxRetries
	if maxRetries == 0 {
		maxRetries = 1
	}

	var lastErr error
	for retry := range maxRetries {
		clamd, err := ConnectWithOpts(s.opts.Network, s.opts.Address, s.opts.Opts)
		if err == nil {
			s.clamd = clamd
			return nil
		}

		lastErr = err
		if retry+1 < maxRetries {
			time.Sleep(s.opts.ConnectRetries.Backoff(retry))
		}
	}

	return fmt.Errorf("max retries reached: %w", lastErr)
}

// heartbeat keeps a session alive with a PING command
func (s *Session) heartbeat() error {
	pong, err := s.clamd.Ping()
	if err != nil {
		return fmt.Errorf("unable to keep alive session: %w", err)
	}
	if match, _ := regexp.MatchString("^[0-9]*:?\\s*PONG", pong); !match {
		return fmt.Errorf("%w: invalid PING response: %s", err, pong)
	}

	// everything ok
	return nil
}

func (s *Session) sessionEventDisconnected(cause any) {
	var causeErr error
	if err, ok := cause.(error); ok {
		causeErr = err
	} else {
		causeErr = fmt.Errorf("%w: %v", ErrClamd, cause)
	}
	s.events <- SessionEvent{
		Type:     SessionDisconnected,
		Severity: EventFatal,
		Error:    causeErr,
		Session:  s,
	}
}

func (s *Session) sessionEventClosed() {
	s.events <- SessionEvent{
		Type:     SessionClosed,
		Severity: EventInfo,
		Error:    nil,
		Session:  s,
	}
}

func (s *Session) sessionEventHeartbeat() {
	s.events <- SessionEvent{
		Type:     SessionHeartBeat,
		Severity: EventInfo,
		Error:    nil,
		Session:  s,
	}
}
