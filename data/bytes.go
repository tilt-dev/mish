package data

import (
	"bytes"
	"fmt"
	"io"
)

// An immutable byte slice
type Bytes struct {
	data []byte
}

func BytesFromString(s string) Bytes {
	return NewBytesWithBacking([]byte(s))
}

func NewEmptyBytes() Bytes {
	return NewBytes(nil)
}

// Create a new Bytes. Copies the backing bytes.
func NewBytes(b []byte) Bytes {
	return Bytes{data: copyByteSlice(b)}
}

// Create a new Bytes. Only use this if you know the slice
// will not be modified.
func NewBytesWithBacking(b []byte) Bytes {
	return Bytes{data: b}
}

// Returns a reference to the internal byte slice.
// Only use this if you know for sure that the caller
// will not modify the slice.
func (b Bytes) InternalByteSlice() []byte {
	return b.data
}

func (b Bytes) Reader() io.Reader {
	return bytes.NewReader(b.data)
}

// Returns a copy of the internal byte slice
func (b Bytes) ToByteSlice() []byte {
	return copyByteSlice(b.data)
}

func (b Bytes) Len() int {
	return len(b.data)
}

func (b Bytes) String() string {
	return string(b.data)
}

func (b Bytes) Extend(newB Bytes) Bytes {
	return b.InsertAt(len(b.data), newB)
}

func (b Bytes) InsertAt(index int, newB Bytes) Bytes {
	if len(b.data) == 0 {
		return newB
	} else if len(newB.data) == 0 {
		return b
	}

	data := make([]byte, 0, len(b.data)+newB.Len())
	data = append(data, b.data[:index]...)
	data = append(data, newB.data...)
	data = append(data, b.data[index:]...)
	return NewBytesWithBacking(data)
}

func (b Bytes) RemoveAt(index int, deleteCount int) Bytes {
	data := make([]byte, 0, len(b.data)-deleteCount)
	data = append(data, b.data[:index]...)
	data = append(data, b.data[index+deleteCount:]...)
	return NewBytesWithBacking(data)
}

func (b Bytes) ApplySplices(splices []EditFileSplice) Bytes {
	newData := make([]byte, 0, b.Len())
	oldDataIndex := int64(0)
	newDataIndex := int64(0)

	for i, s := range splices {
		switch s := s.(type) {
		case *InsertBytesSplice:
			delta := s.Index - newDataIndex
			if delta < 0 {
				newData = append(newData, b.data[oldDataIndex:]...)
				return NewBytesWithBacking(newData).ApplySplices(splices[i:])
			}
			newData = append(newData, b.data[oldDataIndex:(oldDataIndex+delta)]...)
			newData = append(newData, s.Data.InternalByteSlice()...)
			oldDataIndex = oldDataIndex + delta
			newDataIndex = s.Index + int64(s.Data.Len())
		case *DeleteBytesSplice:
			delta := s.Index - newDataIndex
			if delta < 0 {
				newData = append(newData, b.data[oldDataIndex:]...)
				return NewBytesWithBacking(newData).ApplySplices(splices[i:])
			}
			newData = append(newData, b.data[oldDataIndex:(oldDataIndex+delta)]...)
			oldDataIndex = oldDataIndex + delta + s.DeleteCount
			newDataIndex = s.Index
		default:
			panic(fmt.Sprintf("Unrecognized splice type %T", s))
		}
	}

	newData = append(newData, b.data[oldDataIndex:]...)
	return NewBytesWithBacking(newData)
}

func (b Bytes) Equal(b2 Bytes) bool {
	return bytes.Equal(b.data, b2.data)
}

func copyByteSlice(data []byte) []byte {
	result := make([]byte, len(data))
	copy(result, data)
	return result
}
