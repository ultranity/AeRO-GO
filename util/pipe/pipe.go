package pipe

import (
	"io"
	"sync"
)

var bufSize int = 16 * 1024

func Join(left io.ReadWriteCloser, right io.ReadWriteCloser) (wn int64, rn int64, we error, re error) {
	var wait sync.WaitGroup

	pipe := func(to io.ReadWriteCloser, from io.ReadWriteCloser, written *int64, err *error) {
		defer to.Close()
		defer from.Close()
		defer wait.Done()
		buf := make([]byte, bufSize)
		*written, *err = io.CopyBuffer(to, from, buf)
	}
	wait.Add(2)
	go pipe(left, right, &wn, &we)
	go pipe(right, left, &rn, &re)
	wait.Wait()
	return
}
