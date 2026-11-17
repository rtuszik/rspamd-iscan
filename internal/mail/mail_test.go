package mail

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/fho/rspamd-iscan/internal/testutils/mail"
)

func AssertNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func AssertErr(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error but got none")
	}
}

func TestAddHeaders(t *testing.T) {
	const expected = "From: someone@example.com\r\nTo: someone_else@example.com\r\nSubject: An RFC 822 formatted message\r\nNew-Header1: v1\r\nNew-Header2: v2\r\n\r\nThis is the plain text body of the message. Note the blank line\r\nbetween the header information and the body of the message.\r\n"

	tmpdir := t.TempDir()
	fd, err := os.CreateTemp(tmpdir, t.Name())
	AssertNoErr(t, err)

	exampleFd, err := os.Open(mail.TestHamMailPath(t))
	AssertNoErr(t, err)

	_, err = io.Copy(fd, exampleFd)
	AssertNoErr(t, err)

	err = fd.Close()
	AssertNoErr(t, err)
	_ = exampleFd.Close()

	hdrs, err := AsHeaders([]*Header{
		{Name: "New-Header1", Body: "v1"},
		{Name: "New-Header2", Body: "v2"},
	})
	AssertNoErr(t, err)
	err = AddHeaders(fd.Name(), hdrs)
	AssertNoErr(t, err)

	result, err := os.ReadFile(fd.Name())
	AssertNoErr(t, err)

	if !bytes.Equal(result, []byte(expected)) {
		t.Errorf("Got:\n%q\nExpected:\n%q\n", string(result), expected)
	}
}

func TestReplaceHeader(t *testing.T) {
	const expected = "From: newperson@example.com\r\nTo: someone_else@example.com\r\nSubject: An RFC 822 formatted message\r\n\r\nThis is the plain text body of the message. Note the blank line\r\nbetween the header information and the body of the message.\r\n"

	tmpdir := t.TempDir()
	fd, err := os.CreateTemp(tmpdir, t.Name())
	AssertNoErr(t, err)

	exampleFd, err := os.Open(mail.TestHamMailPath(t))
	AssertNoErr(t, err)

	_, err = io.Copy(fd, exampleFd)
	AssertNoErr(t, err)

	err = fd.Close()
	AssertNoErr(t, err)
	_ = exampleFd.Close()

	// Replace the From header
	err = ReplaceHeader(fd.Name(), Header{
		Name: "From",
		Body: "newperson@example.com",
	})
	AssertNoErr(t, err)

	result, err := os.ReadFile(fd.Name())
	AssertNoErr(t, err)

	if !bytes.Equal(result, []byte(expected)) {
		t.Errorf("Got:\n%q\nExpected:\n%q\n", string(result), expected)
	}
}

func TestAddHeadersEdgeCases(t *testing.T) {
	const hdrs = "X: y\r\n"
	longValue := strings.Repeat("a", 8000)
	// "\r\n\r\n" starts at offset 4094, spanning a 4096 byte boundary
	boundaryValue := strings.Repeat("a", 4094-len("A: "))

	tests := []struct {
		name     string
		in       string
		expected string
	}{
		{
			name:     "crlf",
			in:       "A: 1\r\nB: 2\r\n\r\nbody\r\n\r\nmore\r\n",
			expected: "A: 1\r\nB: 2\r\nX: y\r\n\r\nbody\r\n\r\nmore\r\n",
		},
		{
			name:     "bare lf",
			in:       "A: 1\nB: 2\n\nbody\n\nmore\n",
			expected: "A: 1\nB: 2\nX: y\n\nbody\n\nmore\n",
		},
		{
			name:     "separator spans read buffer boundary",
			in:       "A: " + boundaryValue + "\r\n\r\nbody\r\n",
			expected: "A: " + boundaryValue + "\r\nX: y\r\n\r\nbody\r\n",
		},
		{
			name:     "header line longer than read buffer",
			in:       "A: " + longValue + "\r\n\r\nbody\r\n",
			expected: "A: " + longValue + "\r\nX: y\r\n\r\nbody\r\n",
		},
		{
			name:     "no body",
			in:       "A: 1\r\nB: 2\r\n",
			expected: "A: 1\r\nB: 2\r\nX: y\r\n",
		},
		{
			name:     "no body unterminated last line",
			in:       "A: 1\r\nB: 2",
			expected: "A: 1\r\nB: 2\r\nX: y\r\n",
		},
		{
			name:     "no body bare lf unterminated last line",
			in:       "A: 1\nB: 2",
			expected: "A: 1\nB: 2\nX: y\n",
		},
		{
			name:     "empty",
			in:       "",
			expected: "X: y\r\n",
		},
		{
			name:     "no headers",
			in:       "\r\nbody\r\n",
			expected: "X: y\r\n\r\nbody\r\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			err := addHeaders(strings.NewReader(tc.in), &out, []byte(hdrs))
			AssertNoErr(t, err)

			if out.String() != tc.expected {
				t.Errorf("Got:\n%q\nExpected:\n%q\n", out.String(), tc.expected)
			}
		})
	}
}

func TestAddHeadersReadError(t *testing.T) {
	readErr := errors.New("read failed")
	err := addHeaders(iotest.ErrReader(readErr), io.Discard, []byte("X: y\r\n"))
	if !errors.Is(err, readErr) {
		t.Fatalf("expected error wrapping %v, got: %v", readErr, err)
	}
}
