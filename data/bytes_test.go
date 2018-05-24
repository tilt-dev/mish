package data

import (
	"testing"
)

func TestInsert(t *testing.T) {
	assertBytes(t, "hello", b("heo").InsertAt(2, b("ll")))
}

func TestDelete(t *testing.T) {
	assertBytes(t, "by", b("bye").RemoveAt(2, 1))
}

func TestSplices(t *testing.T) {
	assertBytes(t, "hello", b("hello").ApplySplices(splices()))
	assertBytes(t, "hexxllo", b("hello").ApplySplices(splices(ins(2, "xx"))))
	assertBytes(t, "hexxo", b("hello").ApplySplices(splices(ins(2, "xx"), del(4, 2))))
	assertBytes(t, "hexxoy", b("hello").ApplySplices(splices(ins(2, "xx"), del(4, 2), ins(5, "y"))))
	assertBytes(t, "hexxoy", b("hello").ApplySplices(splices(ins(5, "y"), ins(2, "xx"), del(4, 2))))
	assertBytes(t, "elxxl", b("hello").ApplySplices(splices(del(0, 1), ins(2, "xx"), del(5, 1))))
}

func b(str string) Bytes {
	return BytesFromString(str)
}

func assertBytes(t *testing.T, expected string, b Bytes) {
	actual := b.String()
	if expected != actual {
		t.Errorf("Expected %q. Actual %q.", expected, actual)
	}
}

func ins(index int64, bytes string) EditFileSplice {
	return &InsertBytesSplice{Index: index, Data: NewBytes([]byte(bytes))}
}

func del(index int64, count int64) EditFileSplice {
	return &DeleteBytesSplice{Index: index, DeleteCount: count}
}

func splices(s ...EditFileSplice) []EditFileSplice {
	return s
}
