package clamd

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestPool(t *testing.T) {
	p, err := InitSessionPool(SessionPoolOpts{
		PrewarmthSessions: 2,
		MaxIdleSessions:   5,
		NewSession: func() (*Session, error) {
			return OpenSession(SessionOpts{
				Network:           "unix",
				Address:           "/tmp/clamd.sock",
				HeartbeatInterval: 10 * time.Second,
			})
		},
	})
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	assert(t, 2, len(p.freeSessions))

	s1, err := p.Get()
	assertNoErr(t, err)
	assert(t, 1, len(p.freeSessions))

	s2, err := p.Get()
	assertNoErr(t, err)
	assert(t, 0, len(p.freeSessions))

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()

		_, err = s1.Instream(strings.NewReader("novirus"))
		if err != nil {
			t.Errorf("err scan %v", err)
		}
		_, err = s1.Instream(strings.NewReader("novirus"))
		if err != nil {
			t.Errorf("err scan %v", err)
		}
		_, err = s1.Instream(strings.NewReader("novirus"))
		if err != nil {
			t.Errorf("err scan %v", err)
		}
		_, err = s1.Instream(strings.NewReader("novirus"))
		if err != nil {
			t.Errorf("err scan %v", err)
		}
		_, err = s1.Instream(strings.NewReader("novirus"))
		if err != nil {
			t.Errorf("err scan %v", err)
		}
		_, err = s1.Instream(strings.NewReader("novirus"))
		if err != nil {
			t.Errorf("err scan %v", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		_, err = s2.Instream(strings.NewReader("novirus"))
		if err != nil {
			t.Errorf("err scan %v", err)
		}
		_, err = s2.Instream(strings.NewReader("novirus"))
		if err != nil {
			t.Errorf("err scan %v", err)
		}
		_, err = s2.Instream(strings.NewReader("novirus"))
		if err != nil {
			t.Errorf("err scan %v", err)
		}
		_, err = s2.Instream(strings.NewReader("novirus"))
		if err != nil {
			t.Errorf("err scan %v", err)
		}
		_, err = s2.Instream(strings.NewReader("novirus"))
		if err != nil {
			t.Errorf("err scan %v", err)
		}
		_, err = s2.Instream(strings.NewReader("novirus"))
		if err != nil {
			t.Errorf("err scan %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ch := make(chan bool)
	go func() {
		wg.Wait()
		ch <- true
	}()

	select {
	case <-ch:
		// done
	case <-ctx.Done():
		t.Errorf("context timeout exceeded")
	}
}

func assert(t *testing.T, expected any, actual any) {
	if expected != actual {
		t.Errorf("expected %v, got %v", expected, actual)
	}
}

func assertNoErr(t *testing.T, err error) {
	if err != nil {
		t.Errorf("expected no err, got %v", err)
	}
}
