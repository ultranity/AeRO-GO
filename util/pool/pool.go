package pool

import (
	"sync"
)

const bufSize int = 16 * 1024

var bufPool = sync.Pool{
	New: func() any {
		return make([]byte, bufSize)
	},
}

func GetBuf() []byte {
	var buf = bufPool.Get().([]byte)
	return buf
}

func PutBuf(buf []byte) {
	bufPool.Put(buf)
}
