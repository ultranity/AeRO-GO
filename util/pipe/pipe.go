// Copyright 2017 fatedier, fatedier@gmail.com
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pipe

import (
	"io"
	"net"
	"sync"

	"AeRO/proxy/util/pool"

	"github.com/golang/snappy"
)

func pipe(to io.ReadWriteCloser, from io.ReadWriteCloser, wait *sync.WaitGroup, written *int64, err *error) {
	defer to.Close()
	defer from.Close()
	defer wait.Done()

	buf := pool.GetBuf()
	defer pool.PutBuf(buf)
	*written, *err = io.CopyBuffer(to, from, buf)
	//todo: close conn
}

func Join(left net.Conn, right io.ReadWriteCloser) (rn int64, wn int64, we error, re error) {
	var wait sync.WaitGroup
	wait.Add(2)
	go pipe(left, right, &wait, &wn, &we)
	go pipe(right, left, &wait, &rn, &re)
	wait.Wait()
	return
}

func WithCompression(rwc io.ReadWriteCloser) io.ReadWriteCloser {
	sr := snappy.NewReader(rwc)
	sw := snappy.NewBufferedWriter(rwc)
	return WrapReadWriteCloser(sr, sw, rwc)
}

// WithCompressionFromPool will get snappy reader and writer from pool.
// You can recycle the snappy reader and writer by calling the returned recycle function, but it is not necessary.
func WithCompressionFromPool(rwc io.ReadWriteCloser) (out io.ReadWriteCloser, recycle func()) {
	sr := pool.GetSnappyReader(rwc)
	sw := pool.GetSnappyWriter(rwc)
	out = WrapReadWriteCloser(sr, sw, rwc)
	recycle = func() {
		pool.PutSnappyReader(sr)
		pool.PutSnappyWriter(sw)
	}
	return
}

type ReadWriteCloser struct {
	r   io.Reader
	w   io.Writer
	ori io.ReadWriteCloser

	closed bool
	mu     sync.Mutex
}

// closeFn will be called only once
func WrapReadWriteCloser(r io.Reader, w io.Writer, ori io.ReadWriteCloser) io.ReadWriteCloser {
	return &ReadWriteCloser{
		r:      r,
		w:      w,
		ori:    ori,
		closed: false,
	}
}

func (rwc *ReadWriteCloser) Read(p []byte) (n int, err error) {
	return rwc.r.Read(p)
}

func (rwc *ReadWriteCloser) Write(p []byte) (n int, err error) {
	return rwc.w.Write(p)
}

func (rwc *ReadWriteCloser) Close() error {
	rwc.mu.Lock()
	if rwc.closed {
		rwc.mu.Unlock()
		return nil
	}
	rwc.closed = true
	rwc.mu.Unlock()

	if rwc.ori != nil {
		return rwc.ori.Close()
	}
	return nil
}
