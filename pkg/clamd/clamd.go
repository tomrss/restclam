package clamd

import (
	"fmt"
	"io"
	"net"
	"time"
)

const (
	defaultConnectTimeout  time.Duration = 5 * time.Second
	defaultReadTimeout     time.Duration = 60 * time.Second
	defaultWriteTimeout    time.Duration = 5 * time.Second
	defaultStreamChunkSize int           = 2048
)

type Clamd struct {
	Network         string
	Address         string
	ConnectTimeout  time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	StreamChunkSize int
}

func Connect(network string, address string) (*Connection, error) {
	c := Clamd{
		Network:         network,
		Address:         address,
		ConnectTimeout:  defaultConnectTimeout,
		ReadTimeout:     defaultReadTimeout,
		WriteTimeout:    defaultWriteTimeout,
		StreamChunkSize: defaultStreamChunkSize,
	}

	return c.Connect()
}

func (c *Clamd) Connect() (*Connection, error) {
	if c.ConnectTimeout == 0 {
		c.ConnectTimeout = defaultConnectTimeout
	}
	if c.ReadTimeout == 0 {
		c.ReadTimeout = defaultReadTimeout
	}
	if c.WriteTimeout == 0 {
		c.WriteTimeout = defaultWriteTimeout
	}
	if c.StreamChunkSize == 0 {
		c.StreamChunkSize = defaultStreamChunkSize
	}

	conn, err := net.DialTimeout(c.Network, c.Address, c.ConnectTimeout)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrClamd, err)
	}

	return &Connection{
		readTimeout:     c.ReadTimeout,
		writeTimeout:    c.WriteTimeout,
		streamChunkSize: c.StreamChunkSize,
		conn:            conn,
	}, nil
}

func (c *Clamd) Ping() (string, error) {
	conn, err := c.Connect()
	if err != nil {
		return "", err
	}
	defer conn.Close()

	_, pong, err := conn.Ping()
	return pong, err
}

func (c *Clamd) Version() (string, error) {
	conn, err := c.Connect()
	if err != nil {
		return "", err
	}
	defer conn.Close()

	_, version, err := conn.Version()
	return version, err
}

func (c *Clamd) Stats() (string, error) {
	conn, err := c.Connect()
	if err != nil {
		return "", err
	}
	defer conn.Close()

	_, stats, err := conn.Stats()
	return stats, err
}

func (c *Clamd) Scan(path string) (*ScanResult, error) {
	conn, err := c.Connect()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	_, sr, err := conn.Scan(path)
	return sr, err
}

func (c *Clamd) Instream(r io.Reader) (*ScanResult, error) {
	conn, err := c.Connect()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	_, sr, err := conn.Instream(r)
	return sr, err
}
