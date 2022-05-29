package flexiwriter

/*
 * flexiwriter_test.go
 * Tests for flexiwriter
 * By J. Stuart McMurray
 * Created 20220526
 * Last Modified 20220528
 */

import (
	"bytes"
	"io"
	"testing"
	"time"
)

func TestFlexiwriter(t *testing.T) {
	pr1, pw1 := io.Pipe()
	pr2, pw2 := io.Pipe()
	fw := New()
	r1, e1 := fw.Add(pw1)
	_, e2 := fw.Add(pw2)
	have := []byte(time.Now().String())
	go fw.Write(have)
	for i, c := range []struct {
		r io.Reader
		e <-chan error
	}{{pr1, e1}, {pr2, e2}} {
		got := make([]byte, len(have))
		if _, err := c.r.Read(got); nil != err {
			t.Fatalf("Reod from r%d: %s", i+1, err)
		}
		if !bytes.Equal(got, have) {
			t.Fatalf(
				"Read from r%d: got:%q want:%q",
				i+1,
				got,
				have,
			)
		}
		select {
		case err, ok := <-c.e:
			if !ok {
				t.Fatalf("Channel e%d close early", i+1)
			}
			t.Fatalf("Unexpected error from e%d: %s", i+1, err)
		default:
		}
	}
	r1()
	select {
	case err, ok := <-e1:
		if !ok {
			t.Fatalf("Close before receive from e1")
		}
		if nil != err {
			t.Fatalf("Error from e1: %s", err)
		}
	default:
		t.Fatalf("No receive from e1")
	}
	select {
	case err, ok := <-e1:
		if ok {
			t.Fatalf("No close after receive from 1")
		}
		if nil != err {
			t.Fatalf("Extra error from e1: %s", err)
		}
	default:
		t.Fatalf("No receive from e1")
	}
}
