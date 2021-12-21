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
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"github.com/h2non/filetype"
	"io"
	"io/ioutil"
	"os"
)

func GetContentReaderFromFile(filename string) (ContentReader, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}
	reader, err := GetContentReader(filename, stat.Size(), f)
	if err != nil || reader == nil {
		_ = f.Close()
	}
	return reader, err
}

func GetContentReader(filename string, size int64, reader ReadReaderAtCloser) (ContentReader, error) {
	var header = make([]byte, 262)
	_, err := reader.ReadAt(header, 0)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("unable to read header from %s:\n%v", filename, err)
	}
	kind, _ := filetype.Match(header)
	switch kind.Extension {
	case "tar":
		tarReader := tar.NewReader(reader)
		_, err := tarReader.Next()
		if err != nil {
			return nil, fmt.Errorf("unable to open tar file %s:\n%v", filename, err)
		}
		return &TarReader{
			reader:   tarReader,
			file:     reader,
			filename: filename,
		}, nil
	case "gz":
		uncompressedStream, err := gzip.NewReader(reader)
		if err != nil {
			return nil, fmt.Errorf("unable to open gzip file %s:\n%v", filename, err)
		}
		buffer, err := ioutil.ReadAll(uncompressedStream)
		if err != nil {
			return nil, fmt.Errorf("unable to buffer data from %s:\n%v", filename, err)
		}
		defer func(reader ReadReaderAtCloser) {
			_ = reader.Close()
		}(reader)
		return GetContentReader(filename, int64(len(buffer)), bufferedReadReaderAtCloser{bytes.NewReader(buffer)})
	case "zip":
		r, err := zip.NewReader(reader, size)
		if err != nil {
			return nil, fmt.Errorf("unable to open zip file %s:\n%v", filename, err)
		}
		return &ZipReader{r, reader, filename}, nil
	}
	return nil, nil
}

type ReadReaderAt interface {
	io.Reader
	io.ReaderAt
}

type ReadReaderAtCloser interface {
	ReadReaderAt
	io.Closer
}

type bufferedReadReaderAtCloser struct {
	bytesReader *bytes.Reader
}

func (b bufferedReadReaderAtCloser) Read(p []byte) (n int, err error) {
	return b.bytesReader.Read(p)
}

func (b bufferedReadReaderAtCloser) ReadAt(p []byte, off int64) (n int, err error) {
	return b.bytesReader.ReadAt(p, off)
}

func (b bufferedReadReaderAtCloser) Close() error {
	return nil
}
