package clamd

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

type SessionPool struct {
	maxIdle      int
	mut          sync.Mutex
	freeSessions []*Session
	newSession   func() (*Session, error)
	sessionIDSeq int
	logger       Logger
}

// TODO use func opts: type SessionPoolOpts func(s *Session)
type SessionPoolOpts struct {
	PrewarmthSessions int
	MaxIdleSessions   int
	ConnectMaxRetries int
	NewSession        func() (*Session, error)
	HeartbeatInterval time.Duration
	HeartbeatTimeout  time.Duration
	CommandTimeout    time.Duration
	Logger            Logger
}

func InitSessionPool(opts SessionPoolOpts) (*SessionPool, error) {
	var sessionsCap int
	if opts.PrewarmthSessions == 0 {
		sessionsCap = 5
	} else {
		sessionsCap = 2 * opts.PrewarmthSessions
	}

	if opts.Logger == nil {
		opts.Logger = &noopLogger{}
	}

	sessionC := make(chan *Session, opts.PrewarmthSessions)
	errorC := make(chan error)

	for range opts.PrewarmthSessions {
		go func() {
			s, err := opts.NewSession()
			if err != nil {
				errorC <- err
				return
			}

			sessionC <- s
		}()
	}

	sessions := make([]*Session, opts.PrewarmthSessions, sessionsCap)
	errs := make([]error, 0, opts.PrewarmthSessions)

	sessionIDSeq := 0
	for i := range opts.PrewarmthSessions {
		select {
		case err := <-errorC:
			// honestly not so much useful, probably if error we will
			// have same error, but who knows
			opts.Logger.Error().Err(err).Msg("BLEEEB")
			errs = append(errs, err)
		case s := <-sessionC:
			sessionIDSeq++
			s.id = sessionIDSeq
			sessions[i] = s
			opts.Logger.Debug().Int("sessionId", s.id).Msg("session prewarmed")
		}
	}

	if len(errs) != 0 {
		// cleanup successfully open sessions if any
		closeSessionsIgnoreErrors(sessions)
		return nil, fmt.Errorf("error(s) while prewarming clamd sessions: %w",
			errors.Join(errs...))
	}

	go runSupervisor(opts.Logger, opts.NewSession, sessions)

	return &SessionPool{
		maxIdle:      opts.MaxIdleSessions,
		mut:          sync.Mutex{},
		freeSessions: sessions,
		newSession:   opts.NewSession,
		sessionIDSeq: sessionIDSeq,
		logger:       opts.Logger,
	}, nil
}

func (p *SessionPool) Get() (*Session, error) {
	p.mut.Lock()
	defer p.mut.Unlock()

	lenFree := len(p.freeSessions)
	if lenFree == 0 {
		s, err := p.newSession()
		if err != nil {
			return nil, err
		}

		p.sessionIDSeq++
		s.id = p.sessionIDSeq

		return s, nil
	}

	s := p.freeSessions[lenFree-1]
	p.freeSessions = p.freeSessions[:lenFree-1]
	return s, nil
}

func (p *SessionPool) Put(s *Session) {
	p.mut.Lock()
	defer p.mut.Unlock()

	if len(p.freeSessions) < p.maxIdle {
		// return to pool
		p.freeSessions = append(p.freeSessions, s)
	} else {
		// too much idle, close this one async
		go func() {
			if err := s.Close(); err != nil {
				p.logger.Warn().Int("sessionId", s.id).Err(err).Msg("error closing session")
			}
			p.logger.Debug().
				Int("sessionId", s.id).
				Msg("max idle reached, closing session instead of returning to pool")
		}()
	}
}

func (p *SessionPool) Close() {
	closeSessionsIgnoreErrors(p.freeSessions)
	p.logger.Info().Msg("clamd session pool closed")
}

func closeSessionsIgnoreErrors(sessions []*Session) {
	if len(sessions) == 0 {
		return
	}

	// close async all of them
	done := make(chan bool, len(sessions))
	for _, s := range sessions {
		go func(session *Session) {
			defer func() {
				//nolint
				if r := recover(); r != nil {
					// happily ignore this, maybe send a warning in the eventstream
				}
				done <- true
			}()
			session.Close()
		}(s)
	}

	// wait them all to close
	for range sessions {
		<-done
	}
	close(done)
}

// TODO very bad func arguments
//
//nolint:cyclop
func runSupervisor(logger Logger, sessionFactory func() (*Session, error), sessions []*Session) {
	// TODO use streamlined function to log, no need to switch except for reconnect
	for event := range mergeChannels(sessions) {
		switch event.Type {
		case SessionHeartBeat:
			logger.Trace().Int("sessionId", event.Session.id).Msg("heartbeat")
		case SessionDisconnected:
			// todo ugly if
			if event.Severity > EventWarning {
				logger.Warn().Int("sessionId", event.Session.id).Msg("session disconnected")
			} else {
				logger.Info().Int("sessionId", event.Session.id).Msg("session disconnected")
			}
			go func() {
				logger.Info().Int("sessionId", event.Session.id).Msg("trying to reconnect")

				// replace session with new one
				newSession, err := sessionFactory()
				if err != nil {
					logger.Error().
						Int("sessionId", event.Session.id).
						Err(err).
						Msg("max retry reached")
					// TODO
					panic(err)
				}

				var oldSession *Session
				for _, s := range sessions {
					if s == event.Session {
						oldSession = s
						break
					}
				}

				if oldSession == nil {
					// todo, this is a very severe error always due to a bug
					panic("received event for a non-managed session")
				}

				// replace session
				logger.Info().Int("sessionId", event.Session.id).Msg("reconnected")
				newSession.id = event.Session.id

				// todo this OBVIOUSLY does not work :)
				//nolint:ineffassign,wastedassign
				oldSession = newSession
			}()

		case SessionClosed:
			// todo
			logger.Info().Int("sessionId", event.Session.id).Msg("closed")
		case SessionError:
			// todo
			logger.Info().Int("sessionId", event.Session.id).Err(event.Error).Msg("closed")
		}
	}

	logger.Debug().Msg("session supervisor exiting")
}

// https://go.dev/blog/pipelines
func mergeChannels(sessions []*Session) <-chan SessionEvent {
	var wg sync.WaitGroup
	out := make(chan SessionEvent)

	// Start an output goroutine for each input channel in sessions.
	// output copies values from session disconnect channel to out
	// until disconnect is closed, then calls wg.Done.
	output := func(s *Session) {
		for event := range s.Events() {
			out <- event
		}
		wg.Done()
	}

	wg.Add(len(sessions))

	for _, s := range sessions {
		go output(s)
	}

	// Start a goroutine to close out once all the output goroutines are
	// done.  This must start after the wg.Add call.
	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}
