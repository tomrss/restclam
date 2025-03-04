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
	"time"
)

var (
	scanReplyRegex    = regexp.MustCompile(`^([0-9]+)?:?\s*(.+?):\s+(.+)?\s?(OK|FOUND|ERROR)$`)
	genericReplyRegex = regexp.MustCompile(`(?s)^([0-9]+)?:?\s*(.+?)$`)
)

const (
	cmdInitializer byte = 'z'
	cmdTerminator  byte = 0x00
)

var ErrClamd = errors.New("clamd error")

type ScanStatus string

const (
	StatusOK    ScanStatus = "OK"
	StatusFound ScanStatus = "FOUND"
	StatusError ScanStatus = "ERROR"
)

type ScanResult struct {
	Raw      []string
	Status   ScanStatus
	Error    string
	Virus    string
	FileName string
	Details  []string
}

type Connection struct {
	readTimeout     time.Duration
	writeTimeout    time.Duration
	streamChunkSize int

	conn net.Conn
}

func (c *Connection) Close() error {
	return c.conn.Close()
}

//*********************
// BEGIN clamd commands

func (c *Connection) Ping() (int, string, error) {
	return c.simpleCommand("PING")
}

func (c *Connection) Version() (int, string, error) {
	return c.simpleCommand("VERSION")
}

func (c *Connection) Stats() (int, string, error) {
	return c.simpleCommand("STATS")
}

func (c *Connection) Scan(path string) (int, *ScanResult, error) {
	if err := c.sendCommand("SCAN " + path); err != nil {
		return -1, nil, err
	}

	return c.recvScanReply()
}

func (c *Connection) Instream(r io.Reader) (int, *ScanResult, error) {
	if err := c.sendCommand("INSTREAM"); err != nil {
		return -1, nil, err
	}

	// -4 because chunk as 4 byte prefix with chunk length
	readSize := c.streamChunkSize - 4

	for {
		buf := make([]byte, c.streamChunkSize)

		// begin read with offset 4 because 4 bytes are reserved to chunk length
		n, err := r.Read(buf[4:])
		if err != nil && err != io.EOF {
			return -1, nil, fmt.Errorf("%w: error reading stream chunk: %w", ErrClamd, err)
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
			return -1, nil, fmt.Errorf("%w: error writing stream chunk: %w", ErrClamd, err)
		}

		if err == io.EOF {
			// this is the read error. end of read
			break
		}
	}

	// end of streaming, signal this to clamd with a 0-length chunk
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(0))
	if _, err := c.conn.Write(buf); err != nil {
		return -1, nil, fmt.Errorf("%w: error writing stream finalizer: %w", ErrClamd, err)
	}

	return c.recvScanReply()
}

func (c *Connection) Idsession() error {
	return c.sendCommand("IDSESSION")
}

func (c *Connection) End() error {
	return c.sendCommand("END")
}

// END clamd commands
//*********************

func (c *Connection) sendCommand(command string) error {
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

func (c *Connection) recvLine() (string, error) {
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

func (c *Connection) recvScanReply() (int, *ScanResult, error) {
	statusLine, err := c.recvLine()
	if err != nil {
		return -1, nil, err
	}

	requestID, sr, err := parseScanResult(statusLine)
	if err != nil {
		return -1, sr, fmt.Errorf("%w: unable to parse scan reply: %w", ErrClamd, err)
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

	return requestID, sr, nil
}

func (c *Connection) simpleCommand(command string) (int, string, error) {
	if err := c.sendCommand(command); err != nil {
		return -1, "", err
	}

	reply, err := c.recvLine()
	if err != nil {
		return -1, "", err
	}

	return parseGenericReply(reply)
}

func parseGenericReply(reply string) (int, string, error) {
	if reply == "" {
		return -1, "", fmt.Errorf("%w: empty reply from clamd", ErrClamd)
	}

	match := genericReplyRegex.FindStringSubmatch(reply)
	if match == nil {
		return -1, "", fmt.Errorf("%w: unparseable reply '%s'", ErrClamd, reply)
	}

	requestID := 0
	requestIDStr := match[1]
	if requestIDStr != "" {
		parsedRequestID, err := strconv.Atoi(match[1])
		if err != nil {
			return -1, "", fmt.Errorf("%w: non-integer request id: '%s'", ErrClamd, requestIDStr)
		}
		requestID = parsedRequestID
	}
	content := match[2]

	return requestID, content, nil
}

func parseScanResult(statusLine string) (int, *ScanResult, error) {
	if statusLine == "" {
		return -1, nil, fmt.Errorf("%w: empty reply from clamd", ErrClamd)
	}

	match := scanReplyRegex.FindStringSubmatch(statusLine)
	if match == nil {
		return -1, nil, fmt.Errorf("%w: unparseable status line '%s'", ErrClamd, statusLine)
	}

	filename := match[2]
	msg := strings.TrimSpace(match[3])
	status := ScanStatus(match[4])

	requestID := 0
	requestIDStr := match[1]
	if requestIDStr != "" {
		parsedRequestID, err := strconv.Atoi(match[1])
		if err != nil {
			return -1, nil, fmt.Errorf("%w: non-integer request id: '%s'", ErrClamd, requestIDStr)
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

	scanResult := ScanResult{
		Raw:      []string{statusLine},
		Status:   status,
		Error:    errorMsg,
		Virus:    virus,
		FileName: filename,
		Details:  []string{},
	}

	return requestID, &scanResult, nil
}
