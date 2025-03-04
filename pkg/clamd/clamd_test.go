package clamd

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestGenericRegex_Session_Pong(t *testing.T) {
	statusLine := "123: PONG"
	requestID, reply, err := parseGenericReply(statusLine)
	if err != nil {
		t.Error(err)
	}

	if requestID != 123 {
		t.Errorf("wrong requestid: %d", requestID)
	}
	if reply != "PONG" {
		t.Errorf("wrong reply: %s", reply)
	}
}

func TestGenericRegex_Session_Stats(t *testing.T) {
	statusLine := `123: POOLS: 1

STATE: VALID PRIMARY
THREADS: live 1 idle 0 max 12 idle-timeout 30
QUEUE: 0 items
STATS 0.000011

MEMSTATS: heap N/A mmap N/A used N/A free N/A releasable N/A pools 1 pools_used 1369.103M pools_total 1369.152M
END`
	requestID, reply, err := parseGenericReply(statusLine)
	if err != nil {
		t.Error(err)
	}

	if requestID != 123 {
		t.Errorf("wrong requestid: %d", requestID)
	}
	if !strings.HasPrefix(reply, "POOLS") || !strings.HasSuffix(reply, "END") {
		t.Errorf("wrong reply: %s", reply)
	}
}

func TestScanRegex_OK(t *testing.T) {
	statusLine := "/my/test/file.txt: OK"
	_, res, err := parseScanResult(statusLine)
	if err != nil {
		t.Error(err)
	}

	if res.FileName != "/my/test/file.txt" {
		t.Errorf("wrong res.FileName: %s", res.FileName)
	}
	if res.Error != "" {
		t.Errorf("wrong err: %s", res.Error)
	}
	if res.Virus != "" {
		t.Errorf("wrong virus: %s", res.Virus)
	}
	if res.Status != StatusOK {
		t.Errorf("wrong status: %s", res.Status)
	}
}

func TestScanRegex_Found(t *testing.T) {
	statusLine := "/my/test/file.txt: VIRUSSS FOUND"
	_, res, err := parseScanResult(statusLine)
	if err != nil {
		t.Error(err)
	}

	if res.FileName != "/my/test/file.txt" {
		t.Errorf("wrong res.FileName: %s", res.FileName)
	}
	if res.Virus != "VIRUSSS" {
		t.Errorf("wrong virus: %s", res.Virus)
	}
	if res.Status != StatusFound {
		t.Errorf("wrong status: %s", res.Status)
	}
}

func TestScanRegex_Error(t *testing.T) {
	statusLine := "/my/test/file.txt: Error: File not found ERROR"
	_, res, err := parseScanResult(statusLine)
	if err != nil {
		t.Error(err)
	}

	if res.FileName != "/my/test/file.txt" {
		t.Errorf("wrong res.FileName: %s", res.FileName)
	}
	if res.Error != "Error: File not found" {
		t.Errorf("wrong err: %s", res.Error)
	}
	if res.Status != StatusError {
		t.Errorf("wrong status: %s", StatusOK)
	}
}

func TestScanRegex_Session_OK(t *testing.T) {
	statusLine := "1: /my/test/file.txt: OK"
	requestID, res, err := parseScanResult(statusLine)
	if err != nil {
		t.Error(err)
	}

	if requestID != 1 {
		t.Errorf("wrong res.RequestID: %d", requestID)
	}
	if res.FileName != "/my/test/file.txt" {
		t.Errorf("wrong res.FileName: %s", res.FileName)
	}
	if res.Error != "" {
		t.Errorf("wrong err: %s", res.Error)
	}
	if res.Virus != "" {
		t.Errorf("wrong virus: %s", res.Virus)
	}
	if res.Status != StatusOK {
		t.Errorf("wrong status: %s", StatusOK)
	}
}

func TestScanRegex_Session_Found(t *testing.T) {
	statusLine := "2: /my/test/file.txt: VIRUSSS FOUND"
	requestID, res, err := parseScanResult(statusLine)
	if err != nil {
		t.Error(err)
	}

	if requestID != 2 {
		t.Errorf("wrong res.RequestID: %d", requestID)
	}
	if res.FileName != "/my/test/file.txt" {
		t.Errorf("wrong res.FileName: %s", res.FileName)
	}
	if res.Virus != "VIRUSSS" {
		t.Errorf("wrong msg: %s", res.Virus)
	}
	if res.Status != StatusFound {
		t.Errorf("wrong status: %s", res.Status)
	}
}

func TestScanRegex_Session_Error(t *testing.T) {
	statusLine := "123: /my/test/file.txt: Error: File not found ERROR"
	requestID, res, err := parseScanResult(statusLine)
	if err != nil {
		t.Error(err)
	}

	if requestID != 123 {
		t.Errorf("wrong res.RequestID: %d", requestID)
	}
	if res.FileName != "/my/test/file.txt" {
		t.Errorf("wrong res.FileName: %s", res.FileName)
	}
	if res.Error != "Error: File not found" {
		t.Errorf("wrong msg: %s", res.Error)
	}
	if res.Status != StatusError {
		t.Errorf("wrong status: %s", StatusOK)
	}
}
func TestPing(t *testing.T) {
	c, err := Connect("unix", "/tmp/clamd.sock")
	if err != nil {
		t.Error(err)
	}

	_, pong, err := c.Ping()
	if err != nil {
		t.Error(err)
	}
	if pong != "PONG" {
		t.Errorf("Expected 'PONG', got %s, len %d", pong, len(pong))
	}
}

func TestVersion(t *testing.T) {
	c, err := Connect("unix", "/tmp/clamd.sock")
	if err != nil {
		t.Error(err)
	}

	_, version, err := c.Version()
	if err != nil {
		t.Error(err)
	}
	if !strings.HasPrefix(version, "ClamAV 1.") {
		t.Errorf("Expected starting with 'ClamaAV 1.', got %s, len %d", version, len(version))
	}
}

func TestStats(t *testing.T) {
	c, err := Connect("unix", "/tmp/clamd.sock")
	if err != nil {
		t.Error(err)
	}

	_, stats, err := c.Stats()
	if err != nil {
		t.Error(err)
	}
	if !strings.HasPrefix(stats, "POOLS: ") {
		t.Errorf("Expected starting with 'POOLS: ', got %s, len %d", stats, len(stats))
	}
	if !strings.HasSuffix(stats, "END") {
		t.Errorf("Expected ending with 'END', got %s, len %d", stats, len(stats))
	}
}

func TestScan(t *testing.T) {
	fileToScan, err := os.CreateTemp("", "testscan")
	if err != nil {
		t.Error(err)
	}
	defer os.Remove(fileToScan.Name())

	_, err = fileToScan.WriteString("TEST FILE; SHOULD CONTAIN NO VIRUS\n")
	if err != nil {
		t.Error(err)
	}
	_ = fileToScan.Close()

	c, err := Connect("unix", "/tmp/clamd.sock")
	if err != nil {
		t.Error(err)
	}

	_, scan, err := c.Scan(fileToScan.Name())
	if err != nil {
		t.Error(err)
	}

	if scan.Status != StatusOK {
		t.Errorf("Expected status OK, got %s", scan.Status)
	}
	if scan.Error != "" {
		t.Errorf("Expected no error, got %s", scan.Error)
	}
	if scan.Virus != "" {
		t.Errorf("Expected no virus, got %s", scan.Error)
	}
	if scan.FileName != fileToScan.Name() {
		t.Errorf("Expected res.FileName %s, got %s", fileToScan.Name(), scan.FileName)
	}
	fmt.Println(strings.Join(scan.Raw, "\n"))
}

func TestInstream(t *testing.T) {
	r := strings.NewReader("TEST FILE; SHOULD CONTAIN NO VIRUS\n")

	c, err := Connect("unix", "/tmp/clamd.sock")
	if err != nil {
		t.Error(err)
	}

	_, scan, err := c.Instream(r)
	if err != nil {
		t.Error(err)
	}

	if scan.Status != StatusOK {
		t.Errorf("Expected status OK, got %s", scan.Status)
	}
	if scan.Error != "" {
		t.Errorf("Expected no error, got %s", scan.Error)
	}
	if scan.Virus != "" {
		t.Errorf("Expected no virus, got %s", scan.Error)
	}
	if scan.FileName != "stream" {
		t.Errorf("Expected res.FileName %s, got %s", "stream", scan.FileName)
	}
	fmt.Println(strings.Join(scan.Raw, "\n"))
}

func TestInstream_Virus(t *testing.T) {
	r := strings.NewReader("X5O!P%@AP[4\\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*")

	c, err := Connect("unix", "/tmp/clamd.sock")
	if err != nil {
		t.Error(err)
	}

	_, scan, err := c.Instream(r)
	if err != nil {
		t.Error(err)
	}

	if scan.Status != StatusFound {
		t.Errorf("Expected status FOUND, got %s", scan.Status)
	}
	if scan.Error != "" {
		t.Errorf("Expected no error, got %s", scan.Error)
	}
	if scan.Virus != "Win.Test.EICAR_HDB-1" {
		t.Errorf("Expected no virus, got %s", scan.Virus)
	}
	if scan.FileName != "stream" {
		t.Errorf("Expected res.FileName %s, got %s", "stream", scan.FileName)
	}
	fmt.Println(strings.Join(scan.Raw, "\n"))
}
