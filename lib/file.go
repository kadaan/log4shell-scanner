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
	fileReader, err := NewContentFileReader(f)
	if err != nil {
		return nil, err
	}
	reader, err := GetContentReader(filename, stat.Size(), fileReader)
	if err != nil || reader == nil {
		_ = f.Close()
	}
	return reader, err
}

func GetContentReader(filename string, size int64, reader ContentFileReader) (ContentReader, error) {
	kind, _ := filetype.Match(reader.GetHeader())
	switch kind.Extension {
	case "tar":
		tarReader := tar.NewReader(reader)
		_, err := tarReader.Next()
		if err != nil {
			return nil, fmt.Errorf("unable to open tar file: %v", err)
		}
		return NewTarReader(filename, tarReader, reader), nil
	case "gz":
		uncompressedStream, err := gzip.NewReader(reader)
		if err != nil {
			return nil, fmt.Errorf("unable to open gzip file: %v", err)
		}
		buffer, err := ioutil.ReadAll(uncompressedStream)
		if err != nil {
			return nil, fmt.Errorf("unable to buffer data from gzip stream: %v", err)
		}
		defer func(reader ContentFileReader) {
			_ = reader.Close()
		}(reader)
		underlyingReader := NoopCloseReader{bytes.NewReader(buffer)}
		contentFileReader, err := NewContentFileReader(&underlyingReader)
		if err != nil {
			return nil, fmt.Errorf("unable to create content reader: %v", err)
		}
		return GetContentReader(filename, int64(len(buffer)), contentFileReader)
	case "zip":
		buffer, err := ioutil.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("unable to buffer data zip stream: %v", err)
		}
		defer func(reader ContentFileReader) {
			_ = reader.Close()
		}(reader)
		bytesReader := bytes.NewReader(buffer)
		r, err := zip.NewReader(bytesReader, size)
		if err != nil {
			return nil, fmt.Errorf("unable to open zip file: %v", err)
		}
		return NewZipReader(filename, r, reader), nil
	}
	return nil, nil
}

type NoopCloseReader struct {
	reader io.Reader
}

func (r NoopCloseReader) Read(p []byte) (n int, err error) {
	return r.reader.Read(p)
}

func (r NoopCloseReader) Close() error {
	return nil
}
