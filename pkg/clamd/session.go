package clamd

import (
	"fmt"
	"io"
	"time"
)

const (
	defaultHeartbeatInterval time.Duration = 10 * time.Second
)

type Session struct {
	opts SessionOpts
	conn *Connection
}

type SessionOpts struct {
	HeartbeatInterval time.Duration
	ConnectRetries    RetryOpts
	CommandRetries    RetryOpts
}

type RetryOpts struct {
	MaxRetries int
	Backoff    func(retryCount int) time.Duration
}

func OpenSession(network string, address string) (*Session, error) {
	return OpenSessionForClamd(&Clamd{
		Network:         network,
		Address:         address,
		ConnectTimeout:  defaultConnectTimeout,
		ReadTimeout:     defaultReadTimeout,
		WriteTimeout:    defaultWriteTimeout,
		StreamChunkSize: defaultStreamChunkSize,
	})
}

func OpenSessionForClamd(c *Clamd) (*Session, error) {
	return OpenSessionWithOpts(c, SessionOpts{
		HeartbeatInterval: defaultHeartbeatInterval,
		ConnectRetries:    RetryOpts{0, nil},
		CommandRetries:    RetryOpts{0, nil},
	})
}

func OpenSessionWithOpts(c *Clamd, opts SessionOpts) (*Session, error) {
	s := &Session{opts: opts}

	err := s.connectClamd(c)
	if err != nil {
		return nil, err
	}

	if err := s.conn.Idsession(); err != nil {
		return nil, fmt.Errorf("unable to open session: %w", err)
	}

	return s, nil
}

func (s *Session) Close() error {
	if s.conn == nil {
		return nil
	}

	err := s.conn.End()
	if err != nil {
		return fmt.Errorf("unable to end clamd session: %w", err)
	}

	if err := s.conn.Close(); err != nil {
		return fmt.Errorf("unable to close clamd connection: %w", err)
	}

	s.conn = nil
	return nil
}

// delegate command methods

func (s *Session) Ping() (int, string, error) {
	return s.conn.Ping()
}

func (s *Session) Version() (int, string, error) {
	return s.conn.Version()
}

func (s *Session) Stats() (int, string, error) {
	return s.conn.Stats()
}

func (s *Session) Scan(path string) (int, *ScanResult, error) {
	return s.conn.Scan(path)
}

func (s *Session) Instream(r io.Reader) (int, *ScanResult, error) {
	return s.conn.Instream(r)
}

func (s *Session) connectClamd(c *Clamd) error {
	maxRetries := s.opts.ConnectRetries.MaxRetries
	if maxRetries == 0 {
		maxRetries = 1
	}

	var lastErr error
	for retry := range maxRetries {
		conn, err := c.Connect()
		if err == nil {
			s.conn = conn
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
func (s *Session) heartbeat() (int, error) {
	requestID, pong, err := s.conn.Ping()
	if err != nil {
		return -1, fmt.Errorf("unable to keep alive session: %w", err)
	}
	if pong != "PONG" {
		return requestID, fmt.Errorf("%w: invalid PING response: %s", err, pong)
	}

	// everything ok
	return requestID, nil
}
