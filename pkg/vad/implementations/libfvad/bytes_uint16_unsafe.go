package libfvad

import (
	"unsafe"
)

func convertBytesToInt16Slice(b []byte) []int16 {
	ptr := unsafe.SliceData(b)
	return unsafe.Slice((*int16)(unsafe.Pointer(ptr)), len(b)/2)
}
