package rnnoise

import "encoding/binary"

type endian int

const (
	endianUndefined = endian(iota)
	endianBig
	endianLittle
)

func getEndian() endian {
	v := binary.NativeEndian.Uint16([]byte{1, 2})
	switch v {
	case 0x0102:
		return endianBig
	case 0x0201:
		return endianLittle
	}
	return endianUndefined
}
