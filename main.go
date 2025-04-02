package main

import (
	"errors"
	"fmt"
	"math"
	"sync"
	"unsafe"
)

// DataInput represents a heterogeneous array of supported data types.
type DataInput []interface{}

var bufPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 0, 1024) // Optimized buffer reuse
	},
}

// encode converts DataInput into a compact byte slice for network transmission.
func encode(toSend DataInput) ([]byte, error) {
	buf := bufPool.Get().([]byte)[:0] // Reset pooled buffer
	defer bufPool.Put(&buf)           // Return buffer to pool

	return encodeHelper(toSend, buf)
}

// encodeHelper recursively encodes DataInput into a byte buffer.
// It ensures that array and string size limits are respected.
func encodeHelper(data DataInput, buf []byte) ([]byte, error) {
	if len(data) > 1000 {
		return nil, errors.New("array length exceeds limit (1000)")
	}

	buf = append(buf, 'A')                     // Array identifier
	buf = appendVarint(buf, uint64(len(data))) // Encode array length

	for _, v := range data {
		switch v := v.(type) {
		case string:
			if len(v) > 1000000 {
				return nil, errors.New("string length exceeds limit (1,000,000)")
			}
			buf = append(buf, 'S') // String identifier
			buf = appendVarint(buf, uint64(len(v)))

			pos := len(buf)
			buf = append(buf, make([]byte, len(v))...) // Extend buffer
			copy(buf[pos:], v)                         // Optimized copy
		case int32:
			buf = append(buf, 'I')                                           // Int32 identifier
			buf = append(buf, byte(v>>24), byte(v>>16), byte(v>>8), byte(v)) // Direct encoding
		case float64:
			buf = append(buf, 'F') // Float identifier
			bits := math.Float64bits(v)
			buf = append(buf,
				byte(bits>>56), byte(bits>>48), byte(bits>>40), byte(bits>>32),
				byte(bits>>24), byte(bits>>16), byte(bits>>8), byte(bits)) // Float encoding
		case DataInput:
			var err error
			buf, err = encodeHelper(v, buf) // Recursive encoding
			if err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("unsupported data type: %T", v)
		}
	}
	return buf, nil
}

// decode converts a byte slice back into DataInput.
func decode(received []byte) (DataInput, error) {
	if len(received) == 0 {
		return nil, errors.New("empty input")
	}
	pos := 0
	return decodeHelper(received, &pos)
}

// decodeHelper recursively decodes the binary format into DataInput.
func decodeHelper(data []byte, pos *int) (DataInput, error) {
	if *pos >= len(data) || data[*pos] != 'A' {
		return nil, errors.New("invalid format: expected array identifier")
	}
	*pos++ // Skip 'A'

	length, bytesRead, err := readVarint(data[*pos:])
	if err != nil {
		return nil, err
	}
	*pos += bytesRead

	if length > 1000 {
		return nil, errors.New("decoded array length exceeds limit (1000)")
	}

	result := make(DataInput, 0, length)
	for i := uint64(0); i < length; i++ {
		if *pos >= len(data) {
			return nil, errors.New("unexpected end of data")
		}

		switch data[*pos] {
		case 'S': // String
			*pos++
			strLen, bytesRead, err := readVarint(data[*pos:])
			if err != nil {
				return nil, err
			}
			*pos += bytesRead

			if *pos+int(strLen) > len(data) {
				return nil, errors.New("string length exceeds available data")
			}

			result = append(result, bytesToString(data[*pos:*pos+int(strLen)]))
			*pos += int(strLen)
		case 'I': // Int32
			if *pos+4 > len(data) {
				return nil, errors.New("unexpected end of data while reading int32")
			}
			*pos++
			val := int32(data[*pos])<<24 | int32(data[*pos+1])<<16 | int32(data[*pos+2])<<8 | int32(data[*pos+3])
			*pos += 4
			result = append(result, val)
		case 'F': // Float64
			if *pos+8 > len(data) {
				return nil, errors.New("unexpected end of data while reading float64")
			}
			*pos++
			bits := uint64(data[*pos])<<56 | uint64(data[*pos+1])<<48 | uint64(data[*pos+2])<<40 | uint64(data[*pos+3])<<32 |
				uint64(data[*pos+4])<<24 | uint64(data[*pos+5])<<16 | uint64(data[*pos+6])<<8 | uint64(data[*pos+7])
			*pos += 8
			result = append(result, math.Float64frombits(bits))
		case 'A': // Nested array
			nested, err := decodeHelper(data, pos)
			if err != nil {
				return nil, err
			}
			result = append(result, nested)
		default:
			return nil, fmt.Errorf("unknown type identifier: %c", data[*pos])
		}
	}
	return result, nil
}

// appendVarint encodes a uint64 as a compact varint.
func appendVarint(buf []byte, x uint64) []byte {
	for x >= 0x80 {
		buf = append(buf, byte(x)|0x80)
		x >>= 7
	}
	return append(buf, byte(x))
}

// readVarint decodes a varint from a byte slice.
func readVarint(data []byte) (uint64, int, error) {
	var val uint64
	var shift uint
	for i, b := range data {
		val |= uint64(b&0x7F) << shift
		if b < 0x80 {
			return val, i + 1, nil
		}
		shift += 7
		if shift > 63 {
			return 0, 0, errors.New("varint too long")
		}
	}
	return 0, 0, errors.New("unexpected end of data while reading varint")
}

// bytesToString performs a zero-copy conversion from []byte to string.
func bytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// Example usage
func main() {
	originalData := DataInput{"hello", DataInput{"world", int32(123), DataInput{"Kshitij", "ClickHouse", int32(321)}}, float64(3.14)}

	encoded, err := encode(originalData)
	if err != nil {
		fmt.Println("Encoding error:", err)
		return
	}

	decoded, err := decode(encoded)
	if err != nil {
		fmt.Println("Decoding error:", err)
		return
	}

	fmt.Println("Original:", originalData)
	fmt.Println("Encoded:", encoded)
	fmt.Println("Decoded:", decoded)
}
