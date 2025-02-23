package clamd

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var ErrClamd = errors.New("clamd error")

type ScanStatus string

const (
	StatusOK    ScanStatus = "OK"
	StatusFound ScanStatus = "FOUND"
	StatusError ScanStatus = "ERROR"
)

const (
	cmdInitializer byte = 'z'
	cmdTerminator  byte = 0x00

	defaultConnectTimeout  time.Duration = 5 * time.Second
	defaultReadTimeout     time.Duration = 60 * time.Second
	defaultWriteTimeout    time.Duration = 5 * time.Second
	defaultStreamChunkSize int           = 2048
)

var scanReplyRegex = regexp.MustCompile(`^([0-9]+)?:?\s*(.+?):\s+(.+)?\s?(OK|FOUND|ERROR)$`)

type ScanResult struct {
	Raw       []string
	Status    ScanStatus
	Error     string
	Virus     string
	FileName  string
	RequestID int
	Details   []string
}

type Clamd struct {
	readTimeout     time.Duration
	writeTimeout    time.Duration
	streamChunkSize int

	conn net.Conn
	mut  sync.Mutex
}

type Opts struct {
	ConnectTimeout  time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	StreamChunkSize int
}

func Connect(network string, address string) (*Clamd, error) {
	return ConnectWithOpts(network, address, Opts{})
}

func ConnectWithOpts(network string, address string, opts Opts) (*Clamd, error) {
	if opts.ConnectTimeout == 0 {
		opts.ConnectTimeout = defaultConnectTimeout
	}
	if opts.ReadTimeout == 0 {
		opts.ReadTimeout = defaultReadTimeout
	}
	if opts.WriteTimeout == 0 {
		opts.WriteTimeout = defaultWriteTimeout
	}
	if opts.StreamChunkSize == 0 {
		opts.StreamChunkSize = defaultStreamChunkSize
	}

	conn, err := net.DialTimeout(network, address, opts.ConnectTimeout)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrClamd, err)
	}

	return &Clamd{
		readTimeout:     opts.ReadTimeout,
		writeTimeout:    opts.WriteTimeout,
		streamChunkSize: opts.StreamChunkSize,
		conn:            conn,
		mut:             sync.Mutex{},
	}, nil
}

func (c *Clamd) Close() error {
	c.mut.Lock()
	defer c.mut.Unlock()

	return c.conn.Close()
}

func (c *Clamd) Ping() (string, error) {
	c.mut.Lock()
	defer c.mut.Unlock()

	return c.simpleCommand("PING")
}

func (c *Clamd) Version() (string, error) {
	c.mut.Lock()
	defer c.mut.Unlock()

	return c.simpleCommand("VERSION")
}

func (c *Clamd) Stats() (string, error) {
	c.mut.Lock()
	defer c.mut.Unlock()

	return c.simpleCommand("STATS")
}

// TODO private, you should use clamd.Session for session
func (c *Clamd) Idsession() error {
	c.mut.Lock()
	defer c.mut.Unlock()

	return c.sendCommand("IDSESSION")
}

// TODO private, you should use clamd.Session for session
func (c *Clamd) End() error {
	c.mut.Lock()
	defer c.mut.Unlock()

	return c.sendCommand("END")
}

func (c *Clamd) Scan(path string) (*ScanResult, error) {
	c.mut.Lock()
	defer c.mut.Unlock()

	if err := c.sendCommand("SCAN " + path); err != nil {
		return nil, err
	}

	return c.recvScanReply()
}

func (c *Clamd) Instream(r io.Reader) (*ScanResult, error) {
	c.mut.Lock()
	defer c.mut.Unlock()

	if err := c.sendCommand("INSTREAM"); err != nil {
		return nil, err
	}

	// -4 because chunk as 4 byte prefix with chunk length
	readSize := c.streamChunkSize - 4

	for {
		buf := make([]byte, c.streamChunkSize)

		// begin read with offset 4 because 4 bytes are reserved to chunk length
		n, err := r.Read(buf[4:])
		if err != nil && err != io.EOF {
			return nil, err
		}
		if n == 0 {
			// end of read
			break
		}

		if n != readSize {
			// remove bogus bytes from end of buffer
			buf = buf[:n+4]
		}

		// write the 4-byte length prefix in the buffer
		binary.BigEndian.PutUint32(buf, uint32(n))

		// send serialized chunk in the buffer
		if _, err := c.conn.Write(buf); err != nil {
			return nil, err
		}

		if err == io.EOF {
			// end of read
			break
		}
	}

	// end of streaming, signal this to clamd with a 0-length chunk
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(0))
	if _, err := c.conn.Write(buf); err != nil {
		return nil, err
	}

	return c.recvScanReply()
}

func (c *Clamd) sendCommand(command string) error {
	byteCmd := []byte(command)
	fullCmd := make([]byte, 0, len(byteCmd)+2)
	fullCmd = append(fullCmd, cmdInitializer)
	fullCmd = append(fullCmd, byteCmd...)
	fullCmd = append(fullCmd, cmdTerminator)

	if err := c.conn.SetWriteDeadline(time.Now().Add(c.writeTimeout)); err != nil {
		return fmt.Errorf("%w: unable to set write timeout: %w", ErrClamd, err)
	}

	_, err := c.conn.Write(fullCmd)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrClamd, err)
	}
	return nil
}

func (c *Clamd) recvLine() (string, error) {
	if err := c.conn.SetReadDeadline(time.Now().Add(c.writeTimeout)); err != nil {
		return "", fmt.Errorf("%w: unable to set read timeout: %w", ErrClamd, err)
	}

	r := bufio.NewReader(c.conn)

	var ignoreEOF error
	line, err := r.ReadString(cmdTerminator)
	if err == io.EOF {
		// nothing to do
		ignoreEOF = nil
	} else {
		ignoreEOF = err
	}

	return strings.TrimSuffix(line, string(cmdTerminator)), ignoreEOF
}

func (c *Clamd) recvScanReply() (*ScanResult, error) {
	statusLine, err := c.recvLine()
	if err != nil {
		return nil, err
	}

	sr, err := parseScanResult(statusLine)
	if err != nil {
		return sr, fmt.Errorf("unable to parse scan reply: %w", err)
	}

	if sr.Status == StatusError {
		// in this case, clamd sends a duplicated error message in a second line.
		// this behaviour seems a bit off, we add a very strict timeout to be sure.

		// NOTE: many clients wait for an empty reply to ensure clamd
		// sent us all it needs.  we cannot do that because we support
		// sessions, and in session that empty reply is NEVER sent to
		// us!  the only legitimate way to know a command is
		// terminated is to use the provided terminator (newline or
		// null byte).  We can't ignore the second line either: clamd
		// will not allow us to send another command in the session
		// until we read it all!
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		secondLineChan := make(chan string, 1)

		go func() {
			select {
			default:
				secondLine, err := c.recvLine()
				if err != nil {
					// ignore the second line!
					secondLine = ""
				}
				secondLineChan <- secondLine
			case <-ctx.Done():
				fmt.Println("Second line by timeout")
				return
			}
		}()

		select {
		case secondLine := <-secondLineChan:
			sr.Raw = []string{statusLine, secondLine}
			sr.Details = []string{secondLine}
		case <-time.After(100 * time.Millisecond):
			// we'll be fine without a second line!
			fmt.Println("Timed out")
		}
	}

	return sr, nil
}

func (c *Clamd) simpleCommand(command string) (string, error) {
	if err := c.sendCommand(command); err != nil {
		return "", err
	}

	content, err := c.recvLine()
	if err != nil {
		return "", err
	}

	return content, nil
}

func parseScanResult(statusLine string) (*ScanResult, error) {
	match := scanReplyRegex.FindStringSubmatch(statusLine)
	if match == nil {
		return nil, fmt.Errorf("%w: unparseable status line %s", ErrClamd, statusLine)
	}

	filename := match[2]
	msg := strings.TrimSpace(match[3])
	status := ScanStatus(match[4])

	requestID := 0
	requestIDStr := match[1]
	if requestIDStr != "" {
		parsedRequestID, err := strconv.Atoi(match[1])
		if err != nil {
			return nil, fmt.Errorf("%w, unable to parse integer request id: %s", ErrClamd, requestIDStr)
		}
		requestID = parsedRequestID
	}

	var virus string
	var errorMsg string
	if status == StatusOK {
		// no virus, no error
		virus = ""
		errorMsg = ""
	} else if status == StatusFound {
		// msg contains found virus
		virus = msg
		errorMsg = ""
	} else {
		// this should be an error
		virus = ""
		errorMsg = msg
	}

	return &ScanResult{
		Raw:       []string{statusLine},
		Status:    status,
		Error:     errorMsg,
		Virus:     virus,
		FileName:  filename,
		RequestID: requestID,
		Details:   []string{},
	}, nil
}
