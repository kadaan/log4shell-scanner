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
)

type ContentFile interface {
	Name() string
	IsDir() bool
	UncompressedSize() int64
	GetReader() ContentFileReader
	Close() error
}

type ContentReader interface {
	GetFiles() FileIterable
	Filename() string
	GetHash() (string, error)
	Close() error
}

type FileIterable interface {
	Next() (interface{}, error)
}

type ContentFileReader interface {
	io.ReadCloser
	GetHeader() []byte
	GetHash() (string, error)
}

type contentFileReader struct {
	header           []byte
	bufferedReader   *bufio.Reader
	underlyingReader io.ReadCloser
	hasher           hash.Hash
	hash             *string
	eof              bool
}

func NewContentFileReader(reader io.ReadCloser) (ContentFileReader, error) {
	hasher := sha256.New()
	bufferedReader := bufio.NewReader(io.TeeReader(reader, hasher))
	header, err := bufferedReader.Peek(262)
	if err != nil && err != io.EOF {
		return nil, err
	}
	return &contentFileReader{
		header:           header,
		bufferedReader:   bufferedReader,
		underlyingReader: reader,
		hasher:           hasher,
		hash:             nil,
		eof:              false,
	}, nil
}

func (b *contentFileReader) GetHeader() []byte {
	return b.header
}

func (b *contentFileReader) GetHash() (string, error) {
	if b.hash == nil {
		if !b.eof {
			_, err := io.ReadAll(b)
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
