package io

import (
	"bytes"
	"testing"
)

func TestTruncate(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	w := NewTruncateWriter(buf, 5, nil)
	l, err := w.Write([]byte("hello world"))
	if err != nil {
		t.Fatal(err)
	} else if l != 11 {
		t.Errorf("Expected %d, actual %d", 11, l)
	}
	actual := buf.String()
	if actual != "hello" {
		t.Errorf("Expected %q, actual %q", "hello", actual)
	}
}

func TestTruncate2(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	w := NewTruncateWriter(buf, 5, nil)
	l, err := w.Write([]byte("hello"))
	if err != nil {
		t.Fatal(err)
	} else if l != 5 {
		t.Errorf("Expected %d, actual %d", 5, l)
	}
	l, err = w.Write([]byte(" world"))
	if err != nil {
		t.Fatal(err)
	} else if l != 6 {
		t.Errorf("Expected %d, actual %d", 5, l)
	}
	actual := buf.String()
	if actual != "hello" {
		t.Errorf("Expected %q, actual %q", "hello", actual)
	}
}

func TestTruncate3(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	w := NewTruncateWriter(buf, 5, []byte("END\n"))
	l, err := w.Write([]byte("hello world"))
	if err != nil {
		t.Fatal(err)
	} else if l != 11 {
		t.Errorf("Expected %d, actual %d", 11, l)
	}
	actual := buf.String()
	if actual != "helloEND\n" {
		t.Errorf("Expected %q, actual %q", "helloEND\n", actual)
	}
}
