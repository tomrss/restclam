package clamd

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestSession_6InStream(t *testing.T) {
	r1 := strings.NewReader("File 1 is cleannnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnn")
	r2 := strings.NewReader("File 2 is cleannnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnn")
	r3 := strings.NewReader("X5O!P%@AP[4\\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*")
	r4 := strings.NewReader("File 4 is cleannnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnn")
	r5 := strings.NewReader("File 5 is cleannnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnn")
	r6 := strings.NewReader("X5O!P%@AP[4\\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*")

	s, err := OpenSession("unix", "/tmp/clamd.sock")
	if err != nil {
		t.Error(err)
	}

	defer s.Close()

	_, scan1, err1 := s.Instream(r1)
	_, scan2, err2 := s.Instream(r2)
	_, scan3, err3 := s.Instream(r3)

	// TODO remove this, just to test the keepalive
	// time.Sleep(6 * time.Second)

	_, scan4, err4 := s.Instream(r4)
	_, scan5, err5 := s.Instream(r5)
	_, scan6, err6 := s.Instream(r6)

	if err1 != nil {
		t.Error(err1)
	}
	if err2 != nil {
		t.Error(err2)
	}
	if err3 != nil {
		t.Error(err3)
	}
	if err4 != nil {
		t.Error(err4)
	}
	if err5 != nil {
		t.Error(err5)
	}
	if err6 != nil {
		t.Error(err6)
	}

	fmt.Printf("Raw 1: '%s'\n", strings.Join(scan1.Raw, " :: "))
	fmt.Printf("Raw 2: '%s'\n", strings.Join(scan2.Raw, " :: "))
	fmt.Printf("Raw 3: '%s'\n", strings.Join(scan3.Raw, " :: "))
	fmt.Printf("Raw 4: '%s'\n", strings.Join(scan4.Raw, " :: "))
	fmt.Printf("Raw 5: '%s'\n", strings.Join(scan5.Raw, " :: "))
	fmt.Printf("Raw 6: '%s'\n", strings.Join(scan6.Raw, " :: "))

	if scan1.Status != StatusOK {
		t.Errorf("Expected status OK, got %s", scan1.Status)
	}
	if scan2.Status != StatusOK {
		t.Errorf("Expected status OK, got %s", scan2.Status)
	}
	if scan3.Status != StatusFound {
		t.Errorf("Expected status FOUND, got %s", scan3.Status)
	}
	if scan3.Virus != "Win.Test.EICAR_HDB-1" {
		t.Errorf("Expected Win.Test.EICAR_HDB-1 virus, got %s", scan3.Virus)
	}
	if scan4.Status != StatusOK {
		t.Errorf("Expected status OK, got %s", scan4.Status)
	}
	if scan5.Status != StatusOK {
		t.Errorf("Expected status OK, got %s", scan5.Status)
	}
	if scan6.Status != StatusFound {
		t.Errorf("Expected status FOUND, got %s", scan6.Status)
	}
	if scan6.Virus != "Win.Test.EICAR_HDB-1" {
		t.Errorf("Expected Win.Test.EICAR_HDB-1 virus, got %s", scan6.Virus)
	}
}

func TestSession_Mix1(t *testing.T) {
	s, err := OpenSession("unix", "/tmp/clamd.sock")
	if err != nil {
		t.Error(err)
	}

	defer s.Close()

	r1 := strings.NewReader("File 1 is cleannnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnn")
	r2 := strings.NewReader("File 2 is cleannnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnn")
	r3 := strings.NewReader("X5O!P%@AP[4\\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*")
	r4 := strings.NewReader("File 4 is cleannnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnn")
	r5 := strings.NewReader("File 5 is cleannnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnn")
	r6 := strings.NewReader("X5O!P%@AP[4\\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*")
	f7 := tempfile(t, "File scan 7 is clean")
	defer os.Remove(f7)
	f8 := tempfile(t, "File scan 8 is clean")
	defer os.Remove(f8)

	// do some operations in the session
	_, scan1, err1 := s.Instream(r1)
	_, scan2, err2 := s.Instream(r2)
	_, scan3, err3 := s.Instream(r3)
	_, scan7, err7 := s.Scan(f7)
	_, scan4, err4 := s.Instream(r4)
	_, stats, errstat := s.Stats()
	_, scanERR, errERR := s.Scan("notexisssssssstttt_______")
	_, scan5, err5 := s.Instream(r5)
	_, scan6, err6 := s.Instream(r6)
	_, v, errv := s.Version()
	_, scan8, err8 := s.Scan(f8)

	if err1 != nil {
		t.Error(err1)
	}
	if err2 != nil {
		t.Error(err2)
	}
	if err3 != nil {
		t.Error(err3)
	}
	if err4 != nil {
		t.Error(err4)
	}
	if err5 != nil {
		t.Error(err5)
	}
	if err6 != nil {
		t.Error(err6)
	}
	if err7 != nil {
		t.Error(err7)
	}
	if err8 != nil {
		t.Error(err8)
	}
	if errv != nil {
		t.Error(errv)
	}
	if errstat != nil {
		t.Error(errstat)
	}
	if errERR != nil {
		t.Error(errERR)
	}

	fmt.Printf("Raw 1: '%s'\n", strings.Join(scan1.Raw, " :: "))
	fmt.Printf("Raw 2: '%s'\n", strings.Join(scan2.Raw, " :: "))
	fmt.Printf("Raw 3: '%s'\n", strings.Join(scan3.Raw, " :: "))
	fmt.Printf("Raw 4: '%s'\n", strings.Join(scan4.Raw, " :: "))
	fmt.Printf("Raw 5: '%s'\n", strings.Join(scan5.Raw, " :: "))
	fmt.Printf("Raw 6: '%s'\n", strings.Join(scan6.Raw, " :: "))
	fmt.Printf("Raw 7: '%s'\n", strings.Join(scan7.Raw, " :: "))
	fmt.Printf("Raw 8: '%s'\n", strings.Join(scan8.Raw, " :: "))
	fmt.Printf("Raw ERR: '%s'\n", strings.Join(scanERR.Raw, " :: "))

	if scan1.Status != StatusOK {
		t.Errorf("Expected status OK, got %s", scan1.Status)
	}
	if scan2.Status != StatusOK {
		t.Errorf("Expected status OK, got %s", scan2.Status)
	}
	if scan3.Status != StatusFound {
		t.Errorf("Expected status FOUND, got %s", scan3.Status)
	}
	if scan3.Virus != "Win.Test.EICAR_HDB-1" {
		t.Errorf("Expected Win.Test.EICAR_HDB-1 virus, got %s", scan3.Virus)
	}
	if scan4.Status != StatusOK {
		t.Errorf("Expected status OK, got %s", scan4.Status)
	}
	if scan5.Status != StatusOK {
		t.Errorf("Expected status OK, got %s", scan5.Status)
	}
	if scan6.Status != StatusFound {
		t.Errorf("Expected status FOUND, got %s", scan6.Status)
	}
	if scan6.Virus != "Win.Test.EICAR_HDB-1" {
		t.Errorf("Expected Win.Test.EICAR_HDB-1 virus, got %s", scan6.Virus)
	}
	if scan7.Status != StatusOK {
		t.Errorf("Expected status OK, got %s", scan7.Status)
	}
	if scan8.Status != StatusOK {
		t.Errorf("Expected status OK, got %s", scan8.Status)
	}
	if !strings.HasPrefix(v, "ClamAV 1.") {
		t.Errorf("Expected version ClamAV 1.*, got %s", v)
	}
	if !strings.HasPrefix(stats, "POOLS") || !strings.HasSuffix(stats, "END") {
		t.Errorf("Invalid stats: %s", stats)
	}
	if scanERR.Status != StatusError {
		t.Errorf("Expected err")
	}
}

func tempfile(t *testing.T, content string) string {
	f1, err := os.CreateTemp("", "testscan")
	if err != nil {
		t.Error(err)
	}
	_, err = f1.WriteString(content)
	if err != nil {
		t.Error(err)
	}
	_ = f1.Close()

	return f1.Name()
}
