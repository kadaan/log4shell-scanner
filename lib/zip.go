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
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
)

func NewEmbeddedZipReader(filename string, uncompressedSize int64, rc io.Reader) (*ZipContentReader, error) {
	buffer, err := ioutil.ReadAll(rc)
	if err != nil {
		return nil, err
	}

	in := bytes.NewReader(buffer)
	zr, err := zip.NewReader(in, uncompressedSize)
	if err != nil {
		return nil, err
	}
	return &ZipContentReader{zr, &rc, filename}, nil
}

type ZipReader struct {
	zipReader *zip.Reader
	reader    io.ReadCloser
	filename  string
}

func (r *ZipReader) GetFiles() FileIterable {
	return &ZipReaderFileIterable{
		index: 0,
		files: r.zipReader.File,
	}
}

func (r *ZipReader) Filename() string {
	return r.filename
}

func (r *ZipReader) GetReader() io.Reader {
	return r.reader
}

func (r *ZipReader) Close() error {
	err := r.reader.Close()
	if err != nil {
		return fmt.Errorf("failed to close zip reader:\n%v", err)
	}
	return nil
}

type ZipContentReader struct {
	zipReader *zip.Reader
	reader    *io.Reader
	filename  string
}

func (r *ZipContentReader) GetFiles() FileIterable {
	return &ZipReaderFileIterable{
		index: 0,
		files: r.zipReader.File,
	}
}

func (r *ZipContentReader) Filename() string {
	return r.filename
}

func (r *ZipContentReader) GetReader() io.Reader {
	return *r.reader
}

func (r *ZipContentReader) Close() error {
	if c, ok := (*r.reader).(io.Closer); ok {
		return c.Close()
	}
	return nil
}

type ZipFile struct {
	file *zip.File
}

func (z *ZipFile) Name() string {
	return z.file.Name
}

func (z *ZipFile) IsDir() bool {
	return z.file.FileInfo().IsDir()
}

func (z *ZipFile) UncompressedSize() int64 {
	return int64(z.file.UncompressedSize64)
}

func (z *ZipFile) GetContentReader() (ContentReader, error) {
	rc, err := z.file.Open()
	if err != nil {
		return nil, err
	}
	return NewEmbeddedZipReader(z.Name(), int64(z.file.UncompressedSize64), rc.(io.Reader))
}

func (z *ZipFile) GetReader() (io.ReadCloser, error) {
	raw, err := z.file.Open()
	if err != nil {
		return nil, err
	}
	return raw, nil
}

type ZipReaderFileIterable struct {
	index int
	files []*zip.File
}

func (i *ZipReaderFileIterable) Next() (interface{}, error) {
	for {
		if i.index >= len(i.files)-1 {
			return nil, nil
		}
		current := i.index
		i.index += 1
		if i.files[current].FileInfo().IsDir() {
			continue
		}
		return &ZipFile{i.files[current]}, nil
	}
}
