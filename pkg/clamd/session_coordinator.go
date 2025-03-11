package clamd

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"
)

type Coordinator struct {
	MinWorkers      int
	MaxWorkers      int
	Autoscale       bool
	ShutdownTimeout time.Duration

	backends      []Clamd
	workerID      sequence
	jobID         sequence
	jobs          chan job
	activeWorkers sync.WaitGroup
}

func (c *Coordinator) InitCoordinator(backends []Clamd, opts SessionOpts) error {
	c.workerID = newSequence(1)
	c.jobID = newSequence(1)
	c.jobs = make(chan job, c.MaxWorkers)
	c.backends = backends

	numBackends := len(backends)
	for i := range c.MinWorkers {
		go c.spawnWorker(&backends[i%numBackends], opts)
	}

	return nil
}

func (c *Coordinator) Shutdown() {
	fmt.Println("[coord] initiated graceful shutdown...")

	close(c.jobs)

	ctx, cancel := context.WithTimeout(context.Background(), c.ShutdownTimeout)
	defer cancel()

	allClosed := make(chan bool)

	go func() {
		c.activeWorkers.Wait()
		allClosed <- true
	}()

	select {
	case <-allClosed:
		fmt.Println("[coord] all workers successfully closed gracefully")
	case <-ctx.Done():
		fmt.Println("[coord] timeout waiting worker graceful shutdown, force shutdown")
	}
}

func (c *Coordinator) spawnWorker(clamd *Clamd, opts SessionOpts) {
	c.activeWorkers.Add(1)
	defer c.activeWorkers.Done()

	w := sessionWorker{c.workerID.next()}
	if err := w.run(clamd, opts, c.jobs); err != nil {
		// TODO spawn another worker!
		panic(fmt.Errorf("[worker %d] died: %w", w.id, err))
	}
	fmt.Printf("[worker %d] graceful shutdown", w.id)
}

func (c *Coordinator) simpleCommand(cmd func(s *Session) (string, error)) (string, error) {
	out := make(chan jobOutput)
	jobID := c.jobID.next()
	c.jobs <- job{
		ID: jobID,
		Fun: func(s *Session) jobOutput {
			resp, err := cmd(s)
			return jobOutput{
				JobID: jobID,
				Resp:  resp,
				Error: err,
			}
		},
		RespChan: out,
	}
	result := <-out
	return result.Resp, result.Error
}

func (c *Coordinator) Ping() (string, error) {
	return c.simpleCommand(func(s *Session) (string, error) {
		_, pong, err := s.Ping()
		return pong, err
	})
}

func (c *Coordinator) Version() (string, error) {
	return c.simpleCommand(func(s *Session) (string, error) {
		_, version, err := s.Version()
		return version, err
	})
}

func (c *Coordinator) Stats() (string, error) {
	return c.simpleCommand(func(s *Session) (string, error) {
		_, stats, err := s.Stats()
		return stats, err
	})
}

func (c *Coordinator) Scan(path string) (*ScanResult, error) {
	out := make(chan jobOutput)
	jobID := c.jobID.next()
	c.jobs <- job{
		ID: jobID,
		Fun: func(s *Session) jobOutput {
			_, scan, err := s.Scan(path)
			return jobOutput{
				JobID:      jobID,
				ScanResult: scan,
				Error:      err,
			}
		},
		RespChan: out,
	}
	result := <-out
	return result.ScanResult, result.Error
}

func (c *Coordinator) Instream(r io.Reader) (*ScanResult, error) {
	out := make(chan jobOutput)
	jobID := c.jobID.next()
	c.jobs <- job{
		ID: jobID,
		Fun: func(s *Session) jobOutput {
			_, scan, err := s.Instream(r)
			return jobOutput{
				JobID:      jobID,
				ScanResult: scan,
				Error:      err,
			}
		},
		RespChan: out,
	}
	result := <-out
	return result.ScanResult, result.Error
}

type sequence struct {
	val uint
	mu  sync.Mutex
}

func newSequence(start uint) sequence {
	return sequence{start, sync.Mutex{}}
}

func (s *sequence) next() uint {
	s.mu.Lock()
	defer s.mu.Unlock()

	v := s.val
	s.val++
	return v
}

type sessionWorker struct {
	id uint
}

type jobOutput struct {
	JobID      uint
	Resp       string
	ScanResult *ScanResult
	Error      error
}

type jobFun func(s *Session) jobOutput

type job struct {
	ID       uint
	Fun      jobFun
	RespChan chan<- jobOutput
}

func (w *sessionWorker) run(clamd *Clamd, opts SessionOpts, jobs chan job) error {
	s, err := OpenSessionWithOpts(clamd, opts)
	if err != nil {
		return err
	}

	heartbeatTicker := time.NewTicker(opts.HeartbeatInterval)

	defer func() {
		heartbeatTicker.Stop()
		s.Close()
		fmt.Printf("[worker %d] closed gracefully\n", w.id)
	}()

	for {
		select {
		case <-heartbeatTicker.C:
			if _, err := s.heartbeat(); err != nil {
				// this worker died
				return fmt.Errorf("[worker %d] missed heartbeat: %w", w.id, err)
			}
			fmt.Printf("[worker %d] heartbeat\n", w.id)
		case job, channelOpen := <-jobs:
			if !channelOpen {
				// client closed the channel, meaning this session
				// worker should be gracefully closed
				return nil
			}

			// launch the job and return result on the client response channel
			fmt.Printf("[job %d] processing by worker %d...\n", job.ID, w.id)
			result := job.Fun(s)
			fmt.Printf("[job %d] processed by worker %d\n", job.ID, w.id)
			job.RespChan <- result
		}
	}
}
