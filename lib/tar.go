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
	"archive/tar"
	"io"
)

type TarFile struct {
	header *tar.Header
	reader *tar.Reader
}

func (t TarFile) Name() string {
	return t.header.Name
}

func (t TarFile) UncompressedSize() int64 {
	return t.header.Size
}

func (t TarFile) IsDir() bool {
	return t.header.Typeflag == tar.TypeDir
}

func (t TarFile) GetContentReader() (ContentReader, error) {
	return TarContentReader{t.reader, t.Name()}, nil
}

func (t TarFile) GetReader() (io.ReadCloser, error) {
	return TarFileReadCloser{t.reader}, nil
}

type TarFileReadCloser struct {
	reader *tar.Reader
}

func (t TarFileReadCloser) Read(p []byte) (n int, err error) {
	return t.reader.Read(p)
}

func (t TarFileReadCloser) Close() error {
	return nil
}

type TarContentReader struct {
	reader   *tar.Reader
	filename string
}

func (r TarContentReader) GetFiles() FileIterable {
	return TarReaderFileIterable{
		reader: r.reader,
	}
}

func (r TarContentReader) Filename() string {
	return r.filename
}

func (r TarContentReader) GetReader() io.Reader {
	return r.reader
}

func (r TarContentReader) Close() error {
	return nil
}

type TarReader struct {
	reader   *tar.Reader
	file     io.ReadCloser
	filename string
}

func (r TarReader) GetFiles() FileIterable {
	return TarReaderFileIterable{reader: r.reader}
}

func (r TarReader) Filename() string {
	return r.filename
}

func (r TarReader) GetReader() io.Reader {
	return r.reader
}

func (r TarReader) Close() error {
	return r.file.Close()
}

type TarReaderFileIterable struct {
	reader *tar.Reader
}

func (i TarReaderFileIterable) Next() (interface{}, error) {
	for {
		next, err := i.reader.Next()
		if err == io.EOF {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		if next.Typeflag != tar.TypeReg || next.Size == 0 {
			continue
		}
		return TarFile{
			header: next,
			reader: i.reader,
		}, nil
	}
}
