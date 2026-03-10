// Package gguf provides GGUF (GPT-Generated Unified Format) binary file parsing.
// GGUF is a binary format used by llama.cpp for storing LLM models.
package gguf

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// GGUF file format constants
const (
	// MagicNumber is the 4-byte magic string "GGUF" in little endian
	MagicNumber = 0x46554747
	// Version is the minimum supported GGUF version
	Version = 3
	// MaxMetadataSize is the maximum size of metadata we'll read (64MB)
	MaxMetadataSize = 64 * 1024 * 1024
)

// ValueType represents the GGUF value type enumeration
type ValueType uint8

// GGUF value types as specified in the format
const (
	UINT8   ValueType = 0
	INT8    ValueType = 1
	UINT16  ValueType = 2
	INT16   ValueType = 3
	UINT32  ValueType = 4
	INT32   ValueType = 5
	FLOAT32 ValueType = 6
	BOOL    ValueType = 7
	STRING  ValueType = 8
	ARRAY   ValueType = 9
	UINT64  ValueType = 10
	INT64   ValueType = 11
	FLOAT64 ValueType = 12
)

// Header represents the GGUF file header structure
type Header struct {
	Magic           uint32 // Magic number (0x46554747 "GGUF")
	Version         uint32 // GGUF version
	TensorCount     uint64 // Number of tensors in the file
	MetadataKVCount uint64 // Number of key-value pairs in metadata
}

// String is a GGUF string type (length + data)
type String struct {
	Len  uint64
	Data string
}

// Array is a GGUF array type
type Array struct {
	Len   uint64
	Type  ValueType
	Value []interface{}
}

// KVPair represents a metadata key-value pair
type KVPair struct {
	Key   string
	Type  ValueType
	Value interface{}
}

// Errors returned by the GGUF parser
var (
	ErrInvalidMagic       = errors.New("invalid GGUF magic number")
	ErrUnsupportedVersion = errors.New("unsupported GGUF version")
	ErrInvalidValueType   = errors.New("invalid value type")
	ErrUnexpectedEOF      = errors.New("unexpected end of file")
)

// Reader wraps an io.Reader and provides GGUF-specific reading methods
type Reader struct {
	r     io.ReadSeeker
	order binary.ByteOrder
}

// NewReader creates a new GGUF reader
func NewReader(r io.ReadSeeker) *Reader {
	return &Reader{
		r:     r,
		order: binary.LittleEndian,
	}
}

// ReadUint8 reads a uint8 value
func (r *Reader) ReadUint8() (uint8, error) {
	var b [1]byte
	_, err := io.ReadFull(r.r, b[:])
	return b[0], err
}

// ReadInt8 reads an int8 value
func (r *Reader) ReadInt8() (int8, error) {
	b, err := r.ReadUint8()
	return int8(b), err
}

// ReadUint16 reads a uint16 value
func (r *Reader) ReadUint16() (uint16, error) {
	var v uint16
	err := binary.Read(r.r, r.order, &v)
	return v, err
}

// ReadInt16 reads an int16 value
func (r *Reader) ReadInt16() (int16, error) {
	var v int16
	err := binary.Read(r.r, r.order, &v)
	return v, err
}

// ReadUint32 reads a uint32 value
func (r *Reader) ReadUint32() (uint32, error) {
	var v uint32
	err := binary.Read(r.r, r.order, &v)
	return v, err
}

// ReadInt32 reads an int32 value
func (r *Reader) ReadInt32() (int32, error) {
	var v int32
	err := binary.Read(r.r, r.order, &v)
	return v, err
}

// ReadUint64 reads a uint64 value
func (r *Reader) ReadUint64() (uint64, error) {
	var v uint64
	err := binary.Read(r.r, r.order, &v)
	return v, err
}

// ReadInt64 reads an int64 value
func (r *Reader) ReadInt64() (int64, error) {
	var v int64
	err := binary.Read(r.r, r.order, &v)
	return v, err
}

// ReadFloat32 reads a float32 value
func (r *Reader) ReadFloat32() (float32, error) {
	var v float32
	err := binary.Read(r.r, r.order, &v)
	return v, err
}

// ReadFloat64 reads a float64 value
func (r *Reader) ReadFloat64() (float64, error) {
	var v float64
	err := binary.Read(r.r, r.order, &v)
	return v, err
}

// ReadBool reads a bool value (as uint8)
func (r *Reader) ReadBool() (bool, error) {
	b, err := r.ReadUint8()
	return b != 0, err
}

// ReadString reads a GGUF string (length + data)
func (r *Reader) ReadString() (string, error) {
	length, err := r.ReadUint64()
	if err != nil {
		return "", err
	}
	if length == 0 {
		return "", nil
	}
	// Limit string size to prevent excessive memory allocation
	// 增加限制到 100MB，支持新的 GGUF v3 格式和大型元数据
	const maxStringSize = 100 * 1024 * 1024 // 100MB
	if length > maxStringSize {
		return "", fmt.Errorf("string size %d exceeds maximum allowed size %d", length, maxStringSize)
	}

	data := make([]byte, length)
	_, err = io.ReadFull(r.r, data)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ReadType reads a value type enum
func (r *Reader) ReadType() (ValueType, error) {
	t, err := r.ReadUint8()
	return ValueType(t), err
}

// ReadValue reads a value of the specified type
func (r *Reader) ReadValue(vt ValueType) (interface{}, error) {
	switch vt {
	case UINT8:
		return r.ReadUint8()
	case INT8:
		return r.ReadInt8()
	case UINT16:
		return r.ReadUint16()
	case INT16:
		return r.ReadInt16()
	case UINT32:
		return r.ReadUint32()
	case INT32:
		return r.ReadInt32()
	case FLOAT32:
		return r.ReadFloat32()
	case BOOL:
		return r.ReadBool()
	case STRING:
		return r.ReadString()
	case UINT64:
		return r.ReadUint64()
	case INT64:
		return r.ReadInt64()
	case FLOAT64:
		return r.ReadFloat64()
	case ARRAY:
		return r.ReadArray()
	default:
		return nil, ErrInvalidValueType
	}
}

// ReadArray reads a GGUF array
func (r *Reader) ReadArray() (*Array, error) {
	length, err := r.ReadUint64()
	if err != nil {
		return nil, err
	}
	arrType, err := r.ReadType()
	if err != nil {
		return nil, err
	}

	// Limit array size to prevent excessive memory allocation
	const maxArrayLength = 1000000
	if length > maxArrayLength {
		return nil, errors.New("array length exceeds maximum allowed size")
	}

	arr := &Array{
		Len:   length,
		Type:  arrType,
		Value: make([]interface{}, length),
	}

	// For large arrays, only store metadata, not full data
	const skipArrayThreshold = 10000
	if length > skipArrayThreshold {
		// Skip the array data
		for i := uint64(0); i < length; i++ {
			_, err := r.skipValue(arrType)
			if err != nil {
				return nil, err
			}
		}
		arr.Value = nil // Indicate data was skipped
		return arr, nil
	}

	for i := uint64(0); i < length; i++ {
		val, err := r.ReadValue(arrType)
		if err != nil {
			return nil, err
		}
		arr.Value[i] = val
	}

	return arr, nil
}

// skipValue skips a value without storing it (for large arrays)
func (r *Reader) skipValue(vt ValueType) (interface{}, error) {
	switch vt {
	case UINT8, INT8, BOOL:
		return r.ReadUint8()
	case UINT16, INT16:
		return r.ReadUint16()
	case UINT32, INT32, FLOAT32:
		return r.ReadUint32()
	case UINT64, INT64, FLOAT64:
		return r.ReadUint64()
	case STRING:
		return r.ReadString()
	case ARRAY:
		length, err := r.ReadUint64()
		if err != nil {
			return nil, err
		}
		arrType, err := r.ReadType()
		if err != nil {
			return nil, err
		}
		for i := uint64(0); i < length; i++ {
			_, err := r.skipValue(arrType)
			if err != nil {
				return nil, err
			}
		}
		return nil, nil
	default:
		return nil, ErrInvalidValueType
	}
}

// Seek sets the offset for the next Read
func (r *Reader) Seek(offset int64, whence int) (int64, error) {
	if seeker, ok := r.r.(io.Seeker); ok {
		return seeker.Seek(offset, whence)
	}
	return 0, errors.New("reader does not support seeking")
}

// ReadHeader reads and parses the GGUF header
func (r *Reader) ReadHeader() (*Header, error) {
	var header Header

	// Read magic number (4 bytes)
	magic, err := r.ReadUint32()
	if err != nil {
		return nil, err
	}
	if magic != MagicNumber {
		return nil, ErrInvalidMagic
	}
	header.Magic = magic

	// Read version (4 bytes)
	version, err := r.ReadUint32()
	if err != nil {
		return nil, err
	}
	if version < Version {
		return nil, ErrUnsupportedVersion
	}
	header.Version = version

	// Read tensor count (8 bytes)
	tensorCount, err := r.ReadUint64()
	if err != nil {
		return nil, err
	}
	header.TensorCount = tensorCount

	// Read metadata KV count (8 bytes)
	metadataKVCount, err := r.ReadUint64()
	if err != nil {
		return nil, err
	}
	header.MetadataKVCount = metadataKVCount

	return &header, nil
}
