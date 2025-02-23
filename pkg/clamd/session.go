package clamd

import (
	"fmt"
	"io"
	"regexp"
	"time"
)

type Session struct {
	opts  SessionOpts
	clamd *Clamd
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

	err := s.connectClamd()
	if err != nil {
		return nil, err
	}

	if err := s.clamd.Idsession(); err != nil {
		return nil, fmt.Errorf("unable to open session: %w", err)
	}

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
