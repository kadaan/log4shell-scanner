// Copyright © 2021 Joel Baranick <jbaranick@gmail.com>
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
// 	  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package lib

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"sync"
)

var bufferedReaderPool = sync.Pool{
	New: func() interface{} {
		return bufio.NewReader(nil)
	},
}

type Closer func() error

func (c Closer) Close() error {
	return c()
}

type ReaderAtCloser interface {
	io.Reader
	io.ReaderAt
	io.Closer
}

type BufferedReader interface {
	io.Reader
	Peek(n int) ([]byte, error)
}

type MaybeBufferedRead interface {
	io.Reader
	IsBuffered() bool
}

type MaybeBufferedReadCloser interface {
	MaybeBufferedRead
	io.Closer
}

func NewBufferedReadCloser(r io.ReadCloser) MaybeBufferedReadCloser {
	reader := bufferedReaderPool.Get().(*bufio.Reader)
	reader.Reset(r)
	return &bufferedReadCloser{
		reader,
		Closer(func() error {
			bufferedReaderPool.Put(reader)
			return r.Close()
		}),
	}
}

type bufferedReadCloser struct {
	BufferedReader
	closer io.Closer
}

func (bufferedReadCloser) IsBuffered() bool {
	return true
}

func (c *bufferedReadCloser) Close() error {
	return c.closer.Close()
}

func NewUnbufferedReadCloser(r io.ReadCloser) MaybeBufferedReadCloser {
	return &unbufferedReadCloser{r}
}

func NewNopUnbufferedCloser(r io.Reader) MaybeBufferedReadCloser {
	return nopUnbufferedCloser{r}
}

type nopUnbufferedCloser struct {
	io.Reader
}

func (nopUnbufferedCloser) IsBuffered() bool {
	return false
}

func (nopUnbufferedCloser) Close() error { return nil }

type unbufferedReadCloser struct {
	io.ReadCloser
}

func (unbufferedReadCloser) IsBuffered() bool {
	return false
}

type BufferedTeeReader interface {
	BufferedReader
	SeekToEnd() (n int64, err error)
}

func NewBufferedTeeReader(r BufferedReader, w io.Writer) BufferedTeeReader {
	return &bufferedTeeReader{r, w}
}

type bufferedTeeReader struct {
	r BufferedReader
	w io.Writer
}

func (t *bufferedTeeReader) Read(p []byte) (n int, err error) {
	n, err = t.r.Read(p)
	if n > 0 {
		if n, err := t.w.Write(p[:n]); err != nil {
			return n, err
		}
	}
	return
}

func (t *bufferedTeeReader) SeekToEnd() (n int64, err error) {
	return io.Copy(t.w, t.r)
}

func (t *bufferedTeeReader) Peek(n int) ([]byte, error) {
	return t.r.Peek(n)
}

type ContentFile interface {
	Name() string
	IsDir() bool
	UncompressedSize() int64
	Reader() ContentFileReader
	Close() error
}

type ContentReader interface {
	Files() FileIterable
	Filename() string
	Hash() (string, error)
	Close() error
}

type FileIterable interface {
	Next() (interface{}, error)
}

type ContentFileReader interface {
	io.ReadCloser
	Filename() string
	Size() int64
	Header() []byte
	Hash() (string, error)
}

type contentFileReader struct {
	filename         string
	size             int64
	header           []byte
	bufferedReader   BufferedTeeReader
	underlyingReader io.ReadCloser
	hasher           hash.Hash
	hash             *string
	eof              bool
}

func NewContentFileReader(filename string, size int64, reader MaybeBufferedReadCloser) (ContentFileReader, error) {
	hasher := sha256.New()
	var bufferedReader BufferedReader
	if reader.IsBuffered() {
		bufferedReader = reader.(BufferedReader)
	} else {
		bufferedReader = NewBufferedReadCloser(reader).(BufferedReader)
	}
	bufferedTeeReader := NewBufferedTeeReader(bufferedReader, hasher)
	header, err := bufferedReader.Peek(262)
	if err != nil && err != io.EOF {
		return nil, err
	}
	return &contentFileReader{
		filename:         filename,
		size:             size,
		header:           header,
		bufferedReader:   bufferedTeeReader,
		underlyingReader: reader,
		hasher:           hasher,
		hash:             nil,
		eof:              false,
	}, nil
}

func (b *contentFileReader) Filename() string {
	return b.filename
}

func (b *contentFileReader) Size() int64 {
	return b.size
}

func (b *contentFileReader) Header() []byte {
	return b.header
}

func (b *contentFileReader) Hash() (string, error) {
	if b.hash == nil {
		if !b.eof {
			_, err := b.bufferedReader.SeekToEnd()
			if err != nil {
				return "", err
			}
		}
		shaSum := fmt.Sprintf("%x", b.hasher.Sum(nil))
		b.hash = &shaSum
	}
	return *b.hash, nil
}

func (b *contentFileReader) Read(p []byte) (n int, err error) {
	read, err := b.bufferedReader.Read(p)
	if read == 0 || err == io.EOF {
		b.eof = true
	}
	return read, err
}

func (b *contentFileReader) Close() error {
	return b.underlyingReader.Close()
}
